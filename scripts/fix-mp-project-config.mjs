#!/usr/bin/env node
/**
 * 构建后修正 mp-weixin 产物的 project.config.json。
 *
 * 背景：`npx uni build -p mp-weixin` 会重新生成 dist/build/mp-weixin/project.config.json，
 * 其中 setting.minified / setting.ignoreUploadUnusedFiles 固定为 false（硬编码于
 * @dcloudio/uni-mp-weixin/dist/uni.compiler.js 的 source 模板），微信开发者工具的
 * 「JS 未压缩 / 无依赖文件」检测按产物该文件判定，导致假阳性。
 *
 * manifest.json 的 mp-weixin.setting 透传（parseMiniProgramProjectJson 的
 * recursive merge）正常时应覆盖模板值；本脚本作为兜底，保证最终产物一定为：
 *   setting.minified = true
 *   setting.ignoreUploadUnusedFiles = true
 *
 * 用法：node scripts/fix-mp-project-config.mjs [--sync]
 *   --sync  额外把 dist/build/mp-weixin 同步到 ../artifacts/mp-weixin
 *           （delete-before-copy，保证产物与 dist 完全一致）
 */
import { readFileSync, writeFileSync, existsSync, mkdirSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { execFileSync } from 'node:child_process'

const scriptDir = dirname(fileURLToPath(import.meta.url))
const projectRoot = resolve(scriptDir, '..')
const distConfig = join(projectRoot, 'dist/build/mp-weixin/project.config.json')

if (!existsSync(distConfig)) {
  console.error(`[fix-mp-project-config] 未找到 ${distConfig}，请先执行 uni build -p mp-weixin`)
  process.exit(1)
}

const config = JSON.parse(readFileSync(distConfig, 'utf8'))
config.setting = config.setting || {}
const before = {
  minified: config.setting.minified,
  ignoreUploadUnusedFiles: config.setting.ignoreUploadUnusedFiles,
}
// 不覆盖为 false：仅强制这两个键为 true（manifest 透传已是 true 时保持幂等）
config.setting.minified = true
config.setting.ignoreUploadUnusedFiles = true
writeFileSync(distConfig, JSON.stringify(config, null, 2) + '\n')
console.log('[fix-mp-project-config] dist/build/mp-weixin/project.config.json:')
console.log(`  minified: ${before.minified} -> true`)
console.log(`  ignoreUploadUnusedFiles: ${before.ignoreUploadUnusedFiles} -> true`)

if (process.argv.includes('--sync')) {
  const artifactsDir = resolve(projectRoot, '../artifacts/mp-weixin')
  mkdirSync(artifactsDir, { recursive: true })
  // delete-before-copy：先清空再复制，避免遗留已删除文件
  execFileSync('rsync', ['-a', '--delete', '--exclude=.git', `${join(projectRoot, 'dist/build/mp-weixin')}/`, `${artifactsDir}/`], {
    stdio: 'inherit',
  })
  const synced = JSON.parse(readFileSync(join(artifactsDir, 'project.config.json'), 'utf8'))
  console.log(`[fix-mp-project-config] synced -> ${artifactsDir}`)
  console.log(`  artifacts minified=${synced.setting.minified}, ignoreUploadUnusedFiles=${synced.setting.ignoreUploadUnusedFiles}`)
  if (synced.setting.minified !== true || synced.setting.ignoreUploadUnusedFiles !== true) {
    console.error('[fix-mp-project-config] 同步后校验失败')
    process.exit(1)
  }
}
