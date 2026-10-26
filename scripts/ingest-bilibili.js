#!/usr/bin/env node
/**
 * 从 B 站会员购「漫展」票务接口抓真实漫展数据与海报，落到 comic_events。
 * 用法（容器内）：node /app/scripts/ingest-bilibili.js
 * 产出：/app/src/static/img/events/<id>.jpg + /scripts/import.sql
 */
const https = require('https')
const fs = require('fs')
const path = require('path')

const UA = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36'
const PAGES = Number(process.env.PAGES || 3)
const OUT_DIR = '/app/server/static/remote/covers'
const SQL_PATH = process.env.SQL_PATH || '/scripts/import.sql'
const KEEP = /漫展|动漫|嘉年华|同人|comic|chinajoy|萤火虫|cp\d|only|祭|acg|二次元|漫|展/i
const DROP = /咖啡|cafe|主题店|快闪|餐厅|奶茶|酒吧|影院|电影|演唱会|音乐会|话剧|展览馆?票/i

function get (url, binary) {
  return new Promise((resolve, reject) => {
    const req = https.get(url, { headers: { 'User-Agent': UA, Referer: 'https://show.bilibili.com/', Accept: binary ? '*/*' : 'application/json' } }, res => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        res.resume()
        return resolve(get(res.headers.location.startsWith('http') ? res.headers.location : 'https://show.bilibili.com' + res.headers.location, binary))
      }
      const chunks = []
      res.on('data', d => chunks.push(d))
      res.on('end', () => resolve({ status: res.statusCode, buf: Buffer.concat(chunks), headers: res.headers }))
    })
    req.on('error', reject)
    req.setTimeout(20000, () => req.destroy(new Error('timeout')))
  })
}
const sleep = ms => new Promise(r => setTimeout(r, ms))

async function getJSON (url) {
  const r = await get(url)
  if (r.status !== 200) throw new Error('HTTP ' + r.status + ' ' + url)
  return JSON.parse(r.buf.toString('utf8'))
}

const esc = s => String(s == null ? '' : s).replace(/'/g, "''")
const clean = s => String(s || '').replace(/^bilibili会员购票务漫展[:：]\s*/, '').trim()

async function main () {
  fs.mkdirSync(OUT_DIR, { recursive: true })
  const list = []
  for (let p = 1; p <= PAGES; p++) {
    const j = await getJSON(`https://show.bilibili.com/api/ticket/project/listV2?version=134&page=${p}&pagesize=20&area=0&filter=0&platform=web`)
    const arr = (j.data && (j.data.result || j.data.list)) || []
    console.log('list page', p, arr.length)
    list.push(...arr)
    await sleep(200)
  }

  const picked = []
  const seen = new Set()
  for (const it of list) {
    try {
      const j = await getJSON(`https://show.bilibili.com/api/ticket/project/getV2?version=134&id=${it.id}&platform=web`)
      const d = j.data || {}
      const name = clean(d.name)
      if (!name || seen.has(name)) continue
      if (!KEEP.test(name) || DROP.test(name)) continue
      const v = d.venue_info || {}
      const city = (v.city_name || it.city || '').replace(/市$/, '')
      const addr = [v.province_name, v.city_name, v.district_name, v.address_detail].filter(Boolean).join(' ')
      picked.push({
        id: it.id,
        name,
        city,
        venue: v.name || it.venue_name || '',
        address: addr,
        start: Number(d.start_time || it.start_unix || 0),
        end: Number(d.end_time || 0),
        cover: d.cover || it.cover || '',
        desc: clean(d.description).slice(0, 300),
        price: d.price_low || it.price_low || null
      })
      seen.add(name)
    } catch (e) {
      console.log('detail fail', it.id, e.message)
    }
    await sleep(120)
  }
  console.log('picked', picked.length)

  const now = Math.floor(Date.now() / 1000)
  const rows = []
  for (const e of picked) {
    if (!e.start || !e.end || e.end < now) continue
    let local = ''
    if (e.cover) {
      const url = e.cover.startsWith('//') ? 'https:' + e.cover : e.cover
      const ext = (url.split('?')[0].match(/\.(png|jpe?g|webp)$/i) || ['.jpg'])[0].toLowerCase()
      const file = `${e.id}${ext === '.jpeg' ? '.jpg' : ext}`
      const dest = path.join(OUT_DIR, file)
      try {
        const r = await get(url, true)
        if (r.status === 200 && r.buf.length > 2000) {
          fs.writeFileSync(dest, r.buf)
          local = '/static/remote/covers/' + file
        } else console.log('img fail', e.id, r.status, r.buf.length)
      } catch (err) { console.log('img err', e.id, err.message) }
    }
    const status = e.start > now ? 'upcoming' : 'ongoing'
    rows.push({ ...e, local, status })
  }
  console.log('rows', rows.length)

  const ids = rows.map(r => r.id).join(',') || '0'
  const values = rows.map(r => `(${r.id},'${esc(r.name)}','${esc(r.city)}','${esc(r.venue)}','${esc(r.address)}',to_timestamp(${r.start}),to_timestamp(${r.end}),'${esc(r.local)}',ARRAY[]::text[],'${r.status}','https://show.bilibili.com/platform/detail.html?id=${r.id}','{"source":"bilibili","id":${r.id},"price_low":${r.price || 0},"desc":"${esc(r.desc).replace(/"/g,'')}"}'::jsonb,now())`).join(',\n')
  const sql = `BEGIN;
UPDATE comic_events SET del_flag = true WHERE allcpp_id NOT IN (${ids}) AND del_flag = false;
INSERT INTO comic_events (allcpp_id,name,location,venue,address,start_date,end_date,cover_url,tags,status,source_url,raw_data,synced_at)
VALUES
${values}
ON CONFLICT (allcpp_id) DO UPDATE SET name=EXCLUDED.name,location=EXCLUDED.location,venue=EXCLUDED.venue,address=EXCLUDED.address,start_date=EXCLUDED.start_date,end_date=EXCLUDED.end_date,cover_url=EXCLUDED.cover_url,status=EXCLUDED.status,source_url=EXCLUDED.source_url,raw_data=EXCLUDED.raw_data,synced_at=now(),del_flag=false;
COMMIT;`
  fs.writeFileSync(SQL_PATH, sql)
  fs.writeFileSync('/logs/events.json', JSON.stringify(rows, null, 2))
  console.log('sql written', SQL_PATH, 'bytes', sql.length)
}

main().catch(e => { console.error(e); process.exit(1) })
