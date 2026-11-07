<template>
  <page-meta
    root-background-color="#0c0d14"
    background-color="#0c0d14"
    background-color-top="#0c0d14"
    background-color-bottom="#0c0d14"
  />
  <view class="page" :style="{ paddingTop: (statusBarHeight + 44) + 'px' }">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />
    <view class="nav-bar" :style="{ marginTop: statusBarHeight + 'px' }">
      <text class="back-arrow" @click="goBack">‹</text>
      <text class="nav-title">我的套餐</text>
    </view>

    <view class="body">
      <!--
        预填结果提示条：预填/按模板填入保存后只生成「草稿」，草稿不可被预约。
        这条提示必须常驻在页面顶部（用户可手动关掉），不能只用 toast 一闪而过 ——
        用户之前的原话是「预填后返回、回来发现有用户预约了怎么办」，一闪而过的
        toast 无法承担这个信息。
      -->
      <view v-if="draftNotice" class="draft-notice">
        <view class="draft-notice-body">
          <text class="draft-notice-title">已生成草稿</text>
          <text class="draft-notice-text">{{ draftNotice }}</text>
        </view>
        <text class="draft-notice-close" @click="dismissNotice">知道了</text>
      </view>

      <view class="toolbar">
        <text class="toolbar-count">共 {{ services.length }} 个套餐</text>
        <view class="toolbar-actions">
          <view class="btn-tpl" :class="{ disabled: prefilling }" @click="prefillTemplates">一键预填平台模板</view>
          <view class="btn-add" @click="openForm">＋ 新增套餐</view>
        </view>
      </view>

      <view v-if="mode === 'list'">
        <!--
          定价引导：用户原话「我三天去三场漫展，我自己怎么定价？」——
          答案就是「一场一个套餐，或者三天打个包」。这条放在 loading/空态/列表
          三个分支之外，所以空态和已有套餐两种情况下都在最顶部可见：
          第一次进来的人（空态）先学会怎么拆档，老手改价前也能看到建议。
        -->
        <view class="pricing-guide">
          <text class="pricing-guide-title">怎么定价</text>
          <text class="pricing-guide-body">接了几天漫展就建几个场次套餐；多日连拍可建一个打包套餐。</text>
          <text class="pricing-guide-example">例：单场半天 ￥299 · 单日全天 ￥599 · 三天通拍 ￥1499</text>
        </view>

        <view v-if="loading" class="service-list">
          <ListSkeleton :rows="4" variant="card" />
        </view>

        <view v-else-if="services.length === 0" class="empty">
          <text class="empty-icon">◇</text>
          <text class="empty-text">暂无套餐</text>
          <text class="empty-hint">用「一键预填平台模板」先生成草稿（不会上架、不会被预约），确认价格后再逐个上架</text>
          <view class="btn-add empty-cta" @click="openForm">＋ 新增套餐</view>
          <view class="btn-tpl empty-cta-tpl" @click="prefillTemplates">从平台模板生成草稿</view>
        </view>

        <view v-else class="service-list">
          <view
            v-for="s in services"
            :key="s.id"
            class="service-card"
            :class="{ 'is-draft': isDraft(s) }"
          >
            <view class="service-head">
              <text class="service-name">{{ s.name }}</text>
              <text class="price-tag" :class="priceClass(s.price)">{{ renderPrice(s) }}</text>
            </view>
            <view class="service-meta">
              <text class="meta-item">时长 {{ formatDuration(s.duration) }}</text>
              <text class="status-tag" :class="statusClass(s)">{{ statusText(s) }}</text>
            </view>
            <text v-if="s.description" class="service-desc">{{ s.description }}</text>
            <text v-if="isDraft(s)" class="draft-hint">草稿不会被预约，确认价格后点上架才会被 coser 看到并下单</text>
            <view class="service-actions">
              <view
                class="btn-toggle"
                :class="{ 'is-busy': togglingId === s.id, 'is-publish': !isActive(s) }"
                @click="toggleActive(s)"
              >{{ isActive(s) ? '下架' : '上架' }}</view>
              <view class="btn-edit" @click="openEdit(s)">编辑</view>
              <view class="btn-del" @click="confirmDelete(s)">删除</view>
            </view>
          </view>
        </view>
      </view>

      <view v-else class="form-card">
        <view class="form-header">
          <text class="form-title">{{ editingId ? '编辑套餐' : '新增套餐' }}</text>
          <text class="form-sub">套餐会展示在你的摄影师主页，供 coser 选择预约</text>
        </view>

        <view v-if="!editingId && formSource === 'template'" class="form-draft-tip">
          <text class="form-draft-tip-title">按模板填入（尚未保存）</text>
          <text class="form-draft-tip-text">保存后只生成草稿，不会上架；回到列表点「上架」确认后才会被 coser 预约。</text>
        </view>

        <view class="field">
          <text class="field-label">套餐名称</text>
          <input
            v-model="form.name"
            class="field-input"
            placeholder="例如：基础单人约拍"
            placeholder-style="color: #64748b"
            maxlength="50"
          />
        </view>

        <view class="field">
          <text class="field-label">计价方式</text>
          <view class="mode-row">
            <view
              v-for="m in priceUnitModes"
              :key="m.value"
              class="mode-item"
              :class="{ active: form.priceUnit === m.value }"
              @click="form.priceUnit = m.value"
            >{{ m.label }}</view>
          </view>
        </view>

        <view class="field">
          <text class="field-label">{{ priceLabel }}</text>
          <input
            v-model="form.price"
            class="field-input"
            :class="{ 'has-error': priceError }"
            type="number"
            placeholder="留空 = 面议"
            placeholder-style="color: #64748b"
            @input="priceError = ''"
          />
          <text class="field-hint">{{ priceHint }}</text>
          <text v-if="priceError" class="field-error">{{ priceError }}</text>
        </view>

        <view v-if="form.priceUnit === 'per_photo'" class="field">
          <text class="field-label">包含张数（可不填）</text>
          <input
            v-model="form.photoCount"
            class="field-input"
            :class="{ 'has-error': photoCountError }"
            type="number"
            placeholder="留空 = 按需议定"
            placeholder-style="color: #64748b"
            @input="photoCountError = ''"
          />
          <text class="field-hint">一场拍摄包含/起拍的张数；留空 = 按需议定</text>
          <text v-if="photoCountError" class="field-error">{{ photoCountError }}</text>
        </view>

        <view class="field">
          <text class="field-label">时长（分钟）</text>
          <input
            v-model="form.duration"
            class="field-input"
            :class="{ 'has-error': durationError }"
            type="number"
            placeholder="例如：60（分钟）"
            placeholder-style="color: #64748b"
            @input="durationError = ''"
          />
          <text v-if="durationError" class="field-error">{{ durationError }}</text>
        </view>

        <view class="field">
          <text class="field-label">套餐说明（选填）</text>
          <textarea
            v-model="form.description"
            class="field-textarea"
            placeholder="说明拍摄内容、成片数量、交付周期…"
            placeholder-style="color: #64748b"
            maxlength="200"
          />
        </view>

        <view class="btn-submit" :class="{ disabled: !canSubmit }" @click="submit">
          {{ submitting ? '保存中...' : submitLabel }}
        </view>
        <view class="btn-cancel" @click="cancelForm">返回列表</view>
      </view>
    </view>

    <!--
      平台模板面板：打开面板本身不写库。
      「填入表单」= 只填表单不请求接口；底部按钮 = 全部生成为草稿（isActive: false）。
    -->
    <view v-if="tplPanelOpen" class="tpl-mask" @click="closeTemplatePanel">
      <view class="tpl-sheet" @click.stop>
        <view class="tpl-sheet-head">
          <text class="tpl-sheet-title">平台模板</text>
          <text class="tpl-sheet-close" @click="closeTemplatePanel">✕</text>
        </view>
        <text class="tpl-sheet-sub">
          「填入表单」只填不保存；底部按钮会把模板生成为草稿 —— 草稿不上架、coser 看不到也不会被预约。
        </text>
        <text v-if="skippedCount > 0" class="tpl-sheet-skip">
          其中 {{ skippedCount }} 个模板与你已有套餐同名，会跳过（不覆盖已有套餐）。
        </text>

        <view v-if="tplLoading" class="tpl-state">模板加载中...</view>
        <view v-else-if="templates.length === 0" class="tpl-state">暂无平台模板</view>
        <view v-else class="tpl-list">
          <view v-for="t in templates" :key="t.id" class="tpl-row">
            <view class="tpl-row-main">
              <text class="tpl-row-name">{{ t.name }}</text>
              <text class="tpl-row-meta">{{ renderPrice(t) }} · {{ formatDuration(t.duration) }}</text>
            </view>
            <view class="tpl-row-fill" @click="fillFromTemplate(t)">填入表单</view>
          </view>
        </view>

        <view
          class="tpl-primary"
          :class="{ disabled: prefilling || tplLoading || pendingCount === 0 }"
          @click="createDraftsFromTemplates"
        >{{ tplPrimaryLabel }}</view>
        <view class="tpl-cancel" @click="closeTemplatePanel">取消</view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { reactive, ref, computed } from 'vue'
import ListSkeleton from '@/components/ListSkeleton.vue'
import { useStatusBar } from '@/composables/useStatusBar'
import { onShow } from '@dcloudio/uni-app'
import {
  getMyServices,
  getServiceTemplates,
  createService,
  updateService,
  deleteService,
  type MyService,
  type ServiceTemplate,
  type ServiceUpsertPayload
} from '@/api/index'
import { ApiError } from '@/api/client'
import { navTo } from '@/utils/nav'
import { formatDuration } from '@/utils/duration'
import { describeServicePrice, type ServicePrice } from '@/utils/price'
import type { PriceUnit } from '@/types'
import { useUserStore } from '@/stores/user'

const { statusBarHeight } = useStatusBar()
const userStore = useUserStore()

const mode = ref<'list' | 'form'>('list')
const editingId = ref<number | null>(null)
const loading = ref(true)
const submitting = ref(false)
const prefilling = ref(false)
const services = ref<MyService[]>([])
const togglingId = ref<number | null>(null)

/** 表单来源：blank（用户自己填）/ template（从平台模板填入，保存后落为草稿） */
const formSource = ref<'blank' | 'template'>('blank')

/** 平台模板面板（打开面板不写库） */
const templates = ref<ServiceTemplate[]>([])
const tplPanelOpen = ref(false)
const tplLoading = ref(false)

/**
 * 草稿态：后端只有 is_active 一个开关，没有「草稿 / 已下架」的区分字段。
 * 所以「草稿」由前端本地记录：预填（或按模板填入保存）时拿到的 id 记入草稿集合，
 * 用户真正点「上架」成功后从集合里移除 —— 之后被下架的就显示「下架」而不是「草稿」。
 * 集合按账号（photographerId 优先）分 key，避免换号后串到别人的套餐。
 */
const draftIds = ref<number[]>([])
const draftNotice = ref('')

const form = reactive({
  name: '',
  price: '',
  duration: '60',
  description: '',
  priceUnit: 'fixed' as PriceUnit,
  photoCount: ''
})
const priceError = ref('')
const durationError = ref('')
const photoCountError = ref('')

/** 计价方式分段：与后端 price_unit 常量一一对应 */
const priceUnitModes: Array<{ value: PriceUnit; label: string }> = [
  { value: 'fixed', label: '整套价' },
  { value: 'per_photo', label: '按张' },
  { value: 'per_hour', label: '按小时' },
  { value: 'per_day', label: '按天' }
]

const priceLabel = computed(() => {
  switch (form.priceUnit) {
    case 'per_photo':
      return '单价（元/张）'
    case 'per_hour':
      return '单价（元/小时）'
    case 'per_day':
      return '单价（元/天）'
    default:
      return '价格（元）'
  }
})

const priceHint = computed(() => {
  switch (form.priceUnit) {
    case 'per_photo':
      return '单张价格，例：填 10 = ¥10/张；按张计价必填正数'
    case 'per_hour':
      return '留空 = 面议，填 0 = 互勉，填正数 = 每小时价格'
    case 'per_day':
      return '留空 = 面议，填 0 = 互勉，填正数 = 每天价格'
    default:
      return '留空 = 面议，填 0 = 互勉，填正数 = 固定价格'
  }
})


const canSubmit = computed(() => form.name.trim() !== '' && !submitting.value)

const submitLabel = computed(() => {
  if (submitting.value) return '保存中...'
  if (editingId.value) return '保存修改'
  return formSource.value === 'template' ? '保存为草稿' : '创建套餐'
})

const draftCount = computed(() => services.value.filter((s) => isDraft(s)).length)

/** 与已有套餐同名的模板会被跳过（预填只新增，绝不覆盖已有套餐） */
const skippedCount = computed(() => {
  if (templates.value.length === 0) return 0
  const existing = new Set(services.value.map((s) => s.name))
  return templates.value.filter((t) => existing.has(t.name)).length
})

/** 真正会被生成的草稿数（去掉同名跳过的） */
const pendingCount = computed(() => templates.value.length - skippedCount.value)

const tplPrimaryLabel = computed(() => {
  if (prefilling.value) return '生成中...'
  if (tplLoading.value) return '模板加载中...'
  if (templates.value.length === 0) return '暂无平台模板'
  if (pendingCount.value === 0) return '全部模板都已存在'
  return `全部生成为草稿（${pendingCount.value} 个）`
})

onShow(() => {
  if (!userStore.user) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => {
      navTo('/pages/login/index?redirect=/pages/photographer/services')
    }, 800)
    return
  }
  if (mode.value === 'list') {
    loadServices()
  }
})

/* ───────────── 草稿本地状态 ───────────── */

function ownerKey(): string {
  const u = userStore.user
  const key = u?.photographerId ?? u?.id ?? 'anon'
  return String(key)
}

function draftIdsKey(): string {
  return `serviceDraftIds:${ownerKey()}`
}

function draftNoticeKey(): string {
  return `serviceDraftNotice:${ownerKey()}`
}

function loadDraftState() {
  try {
    const raw: unknown = uni.getStorageSync(draftIdsKey())
    draftIds.value = Array.isArray(raw)
      ? (raw as unknown[]).filter((v): v is number => typeof v === 'number')
      : []
  } catch {
    draftIds.value = []
  }
  try {
    const note: unknown = uni.getStorageSync(draftNoticeKey())
    draftNotice.value = typeof note === 'string' ? note : ''
  } catch {
    draftNotice.value = ''
  }
}

function saveDraftIds() {
  try {
    uni.setStorageSync(draftIdsKey(), draftIds.value)
  } catch {
    // 本地写失败不影响主流程：草稿在本次会话内仍然生效
  }
}

function saveNotice() {
  try {
    if (draftNotice.value) {
      uni.setStorageSync(draftNoticeKey(), draftNotice.value)
    } else {
      uni.removeStorageSync(draftNoticeKey())
    }
  } catch {
    // 忽略
  }
}

function markDrafts(ids: number[]) {
  if (ids.length === 0) return
  const next = new Set(draftIds.value)
  ids.forEach((id) => next.add(id))
  draftIds.value = Array.from(next)
  saveDraftIds()
}

function unmarkDraft(id: number) {
  draftIds.value = draftIds.value.filter((x) => x !== id)
  saveDraftIds()
}

function setNotice(text: string) {
  draftNotice.value = text
  saveNotice()
}

function dismissNotice() {
  draftNotice.value = ''
  saveNotice()
}

/**
 * 用服务端真实列表校正本地草稿集合：只在列表加载成功时调用。
 * 加载失败时不能清理，否则一次网络抖动就会把草稿标记全抹掉。
 */
function syncDraftState() {
  const alive = draftIds.value.filter((id) =>
    services.value.some((s) => s.id === id && s.isActive === false)
  )
  if (alive.length !== draftIds.value.length) {
    draftIds.value = alive
    saveDraftIds()
  }
  if (alive.length === 0 && draftNotice.value) {
    draftNotice.value = ''
    saveNotice()
  }
}

function isDraft(s: MyService): boolean {
  return s.isActive === false && draftIds.value.indexOf(s.id) >= 0
}

/* ───────────── 列表 ───────────── */

async function loadServices() {
  loading.value = true
  loadDraftState()
  let ok = false
  try {
    const res = await getMyServices()
    services.value = res?.list || []
    ok = true
  } catch (e) {
    if (e instanceof ApiError && e.status === 403) {
      uni.showToast({ title: '仅摄影师可管理套餐', icon: 'none' })
      setTimeout(() => uni.navigateBack(), 1200)
      return
    }
    services.value = []
  } finally {
    loading.value = false
  }
  if (ok) syncDraftState()
}

function renderPrice(x: ServicePrice): string {
  return describeServicePrice(x)
}

function priceClass(price: number | null): string {
  if (price === null || price === undefined) return 'is-negotiable'
  if (price === 0) return 'is-mutual'
  return 'is-fixed'
}

/** 上架确认里用来描述「按什么价格成交」（带计价单位后缀） */
function dealText(s: MyService): string {
  if (s.price === null || s.price === undefined) return '面议（coser 可发起预约后议价）'
  if (s.price === 0) return '互勉（0 元）'
  return describeServicePrice(s)
}

function isActive(s: MyService): boolean {
  return s.isActive
}

function statusText(s: MyService): string {
  if (isActive(s)) return '上架'
  return isDraft(s) ? '草稿' : '下架'
}

function statusClass(s: MyService): string {
  if (isActive(s)) return 'is-active'
  return isDraft(s) ? 'is-draft' : 'is-down'
}

/**
 * 上下架只翻转 isActive，其余字段原样回传（后端要求 name 必填）。刻意不带
 * sortOrder —— 后端对 isActive / sortOrder 做 COALESCE，省略即保留数据库现值。
 *
 * 上架（isActive: true）是唯一会让套餐变成「可被 coser 直接下单」的动作，
 * 所以必须先过一次二次确认；下架是回收动作，保持一键生效。
 */
async function toggleActive(s: MyService) {
  if (togglingId.value !== null) return
  const next = !isActive(s)
  if (next) {
    const confirmed = await confirmPublish(s)
    if (!confirmed) return
  }
  togglingId.value = s.id
  try {
    await updateService(s.id, {
      name: s.name,
      price: s.price,
      description: s.description || '',
      duration: s.duration,
      priceUnit: s.priceUnit,
      photoCount: s.photoCount ?? null,
      isActive: next
    })
    s.isActive = next
    if (next) {
      unmarkDraft(s.id)
      if (draftCount.value === 0) {
        draftNotice.value = ''
        saveNotice()
      }
      uni.showToast({ title: '已上架，coser 现在可以下单', icon: 'none' })
    } else {
      uni.showToast({ title: '已下架，不再接受新预约', icon: 'none' })
    }
  } catch (e) {
    if (e instanceof ApiError && e.status === 403) {
      uni.showToast({ title: '仅本人套餐可操作', icon: 'none' })
    } else {
      uni.showToast({ title: '操作失败', icon: 'none' })
    }
  } finally {
    togglingId.value = null
  }
}

function confirmPublish(s: MyService): Promise<boolean> {
  return new Promise((resolve) => {
    uni.showModal({
      title: `上架「${s.name}」`,
      content: `上架后 coser 会在你的主页看到该套餐，并可直接按「${dealText(s)}」预约成交。确认价格无误再上架？`,
      confirmText: '确认上架',
      cancelText: '再想想',
      success: (res) => resolve(!!res.confirm),
      fail: () => resolve(false)
    })
  })
}

/* ───────────── 表单 ───────────── */

function resetForm() {
  editingId.value = null
  formSource.value = 'blank'
  form.name = ''
  form.price = ''
  form.duration = '60'
  form.description = ''
  form.priceUnit = 'fixed'
  form.photoCount = ''
  priceError.value = ''
  durationError.value = ''
  photoCountError.value = ''
}

function openForm() {
  resetForm()
  mode.value = 'form'
}

function openEdit(s: MyService) {
  editingId.value = s.id
  formSource.value = 'blank'
  form.name = s.name
  form.price = s.price === null || s.price === undefined ? '' : String(s.price)
  form.duration = s.duration > 0 ? String(s.duration) : ''
  form.description = s.description || ''
  form.priceUnit = s.priceUnit || 'fixed'
  form.photoCount = s.photoCount == null ? '' : String(s.photoCount)
  priceError.value = ''
  durationError.value = ''
  photoCountError.value = ''
  mode.value = 'form'
}

function cancelForm() {
  resetForm()
  mode.value = 'list'
}

function normalizedPrice(): number | null | undefined {
  const raw = form.price.trim()
  if (raw === '') return null
  const n = Number(raw)
  if (!isFinite(n) || n < 0) return undefined
  return Math.round(n)
}

/** 包含张数：空 = 按需议定（null）；正整数 = 具体张数；非法 = undefined（报错） */
function normalizedPhotoCount(): number | null | undefined {
  const raw = form.photoCount.trim()
  if (raw === '') return null
  const n = Number(raw)
  if (!isFinite(n) || n <= 0) return undefined
  return Math.round(n)
}

function normalizedDuration(): number | undefined {
  const raw = form.duration.trim()
  if (raw === '') return undefined
  const n = Number(raw)
  if (!isFinite(n) || n <= 0) return undefined
  return Math.round(n)
}

async function submit() {
  if (!canSubmit.value) return
  const price = normalizedPrice()
  if (price === undefined) {
    priceError.value = '价格需为不小于 0 的数字'
    uni.showToast({ title: '价格需为不小于 0 的数字', icon: 'none' })
    return
  }
  priceError.value = ''
  // 按张计价：单价必须为正数（后端 CHECK: price_unit <> 'per_photo' OR price > 0）
  if (form.priceUnit === 'per_photo' && !(price !== null && price > 0)) {
    priceError.value = '按张计价的单价必须为正数'
    uni.showToast({ title: '按张计价的单价必须为正数', icon: 'none' })
    return
  }
  // 包含张数：按张时可空（= 按需议定），填了必须是正整数；其它计价方式一律不传
  let photoCount: number | null = null
  if (form.priceUnit === 'per_photo') {
    const pc = normalizedPhotoCount()
    if (pc === undefined) {
      photoCountError.value = '张数需为正整数，或留空按需议定'
      uni.showToast({ title: '张数需为正整数，或留空', icon: 'none' })
      return
    }
    photoCount = pc
  }
  const duration = normalizedDuration()
  if (duration === undefined) {
    durationError.value = '时长需为大于 0 的分钟数'
    uni.showToast({ title: '时长需为大于 0 的分钟数', icon: 'none' })
    return
  }
  durationError.value = ''
  const id = editingId.value
  // 从平台模板填进来的新套餐一律落为草稿：模板价只是建议价，未经用户确认不能直接可售。
  const asDraft = !id && formSource.value === 'template'
  submitting.value = true
  uni.showLoading({ title: '保存中...' })
  try {
    const payload: ServiceUpsertPayload = {
      name: form.name.trim(),
      price,
      description: form.description.trim(),
      duration,
      priceUnit: form.priceUnit,
      photoCount
    }
    if (asDraft) payload.isActive = false
    if (id) {
      await updateService(id, payload)
    } else {
      const res = await createService(payload)
      if (asDraft && res && typeof res.id === 'number') {
        markDrafts([res.id])
        setNotice(`已生成 1 个草稿套餐，确认价格后点上架才会被预约`)
      }
    }
    if (id) {
      uni.showToast({ title: '已保存', icon: 'success' })
    } else if (asDraft) {
      uni.showToast({ title: '已保存为草稿', icon: 'success' })
    } else {
      uni.showToast({ title: '创建成功', icon: 'success' })
    }
    resetForm()
    mode.value = 'list'
    await loadServices()
  } catch (e) {
    if (e instanceof ApiError && e.status === 403) {
      uni.showToast({ title: '仅摄影师可管理套餐', icon: 'none' })
    } else if (e instanceof ApiError && e.status === 400) {
      uni.showToast({ title: '价格需为不小于 0 的数字', icon: 'none' })
    } else {
      uni.showToast({ title: id ? '保存失败' : '创建失败', icon: 'none' })
    }
  } finally {
    submitting.value = false
    uni.hideLoading()
  }
}

/* ───────────── 平台模板 ───────────── */

/** 工具栏「一键预填平台模板」= 打开模板面板（打开本身不写库、不落任何套餐） */
function prefillTemplates() {
  if (prefilling.value) return
  tplPanelOpen.value = true
  if (templates.value.length === 0) {
    void loadTemplates()
  }
}

function closeTemplatePanel() {
  tplPanelOpen.value = false
}

async function loadTemplates() {
  tplLoading.value = true
  try {
    const res = await getServiceTemplates()
    templates.value = res || []
  } catch (e) {
    templates.value = []
    if (e instanceof ApiError && e.status === 403) {
      uni.showToast({ title: '仅摄影师可管理套餐', icon: 'none' })
    } else {
      uni.showToast({ title: '模板加载失败，请重试', icon: 'none' })
    }
  } finally {
    tplLoading.value = false
  }
}

/** 单个模板 → 只填表单，不请求接口（用户自己点保存） */
function fillFromTemplate(t: ServiceTemplate) {
  editingId.value = null
  formSource.value = 'template'
  form.name = t.name
  form.price = t.price === null || t.price === undefined ? '' : String(t.price)
  form.duration = t.duration > 0 ? String(t.duration) : ''
  form.description = t.description || ''
  form.priceUnit = 'fixed'
  form.photoCount = ''
  priceError.value = ''
  durationError.value = ''
  photoCountError.value = ''
  tplPanelOpen.value = false
  mode.value = 'form'
  uni.showToast({ title: '已填入表单，还没保存', icon: 'none' })
}

/** 同名套餐已存在就跳过：预填只新增，绝不改动/覆盖已有套餐 */
function pendingTemplates(): ServiceTemplate[] {
  const existing = new Set(services.value.map((s) => s.name))
  return templates.value.filter((t) => !existing.has(t.name))
}

function createDraftsFromTemplates() {
  if (prefilling.value || tplLoading.value) return
  if (templates.value.length === 0) return
  const pending = pendingTemplates()
  const skipped = templates.value.length - pending.length
  if (pending.length === 0) {
    uni.showToast({ title: '模板对应的套餐都已存在', icon: 'none' })
    return
  }
  const skipText = skipped > 0 ? `（已存在 ${skipped} 个同名套餐会跳过，不会覆盖）` : ''
  uni.showModal({
    title: '生成草稿套餐',
    content: `按 ${pending.length} 个平台模板生成草稿${skipText}。草稿不上架，coser 看不到也无法下单；确认价格后逐个上架才会被预约。继续？`,
    confirmText: '生成草稿',
    success: async (res) => {
      if (!res.confirm) return
      await runDraftPrefill(pending)
    }
  })
}

async function runDraftPrefill(list: ServiceTemplate[]) {
  prefilling.value = true
  uni.showLoading({ title: '生成草稿...' })
  const createdIds: number[] = []
  try {
    for (const t of list) {
      try {
        const res = await createService({
          name: t.name,
          price: t.price === null || t.price === undefined ? null : t.price,
          description: t.description || '',
          duration: t.duration || 0,
          priceUnit: 'fixed',
          photoCount: null,
          // 关键：预填只建草稿，落库即 is_active=false，GetBookableServicePhotographerID
          // 查不到这一行，所以预填完不可能被 coser 下单。
          isActive: false
        })
        if (res && typeof res.id === 'number') createdIds.push(res.id)
      } catch {
        // 单个模板失败不影响其余模板
      }
    }
    if (createdIds.length > 0) {
      markDrafts(createdIds)
      setNotice(
        `已生成 ${createdIds.length} 个草稿套餐，确认价格后点上架才会被预约` +
          (createdIds.length < list.length ? `（${list.length - createdIds.length} 个生成失败）` : '')
      )
      tplPanelOpen.value = false
    }
    uni.showToast({
      title: createdIds.length ? `已生成 ${createdIds.length} 个草稿` : '生成失败，请重试',
      icon: 'none'
    })
    await loadServices()
  } catch (e) {
    if (e instanceof ApiError && e.status === 403) {
      uni.showToast({ title: '仅摄影师可管理套餐', icon: 'none' })
      setTimeout(() => uni.navigateBack(), 1200)
      return
    }
    uni.showToast({ title: '生成失败', icon: 'none' })
  } finally {
    prefilling.value = false
    uni.hideLoading()
  }
}

function confirmDelete(s: MyService) {
  uni.showModal({
    title: '删除套餐',
    content: `确定删除「${s.name}」吗？删除后不可恢复，历史订单不受影响`,
    confirmColor: '#ef4444',
    success: async (res) => {
      if (!res.confirm) return
      uni.showLoading({ title: '删除中...' })
      try {
        await deleteService(s.id)
        unmarkDraft(s.id)
        uni.showToast({ title: '已删除', icon: 'success' })
        await loadServices()
      } catch (e) {
        if (e instanceof ApiError && e.status === 403) {
          uni.showToast({ title: '仅摄影师可管理套餐', icon: 'none' })
        } else {
          uni.showToast({ title: '删除失败', icon: 'none' })
        }
      } finally {
        uni.hideLoading()
      }
    }
  })
}

function goBack() {
  uni.navigateBack()
}
</script>

<style lang="scss" scoped>
.page { min-height: 100vh; background: $dark-bg-primary; }
.status-bar { position: fixed; top: 0; left: 0; right: 0; z-index: 100; }
.nav-bar { position: fixed; top: 0; left: 0; right: 0; z-index: 99; height: 44px; display: flex; align-items: center; background: $dark-bg-primary; border-bottom: 1rpx solid $dark-border; }
.back-arrow { font-size: 48rpx; color: $dark-text-primary; padding: 0 24rpx; }
.nav-title { font-size: 34rpx; font-weight: 600; color: $dark-text-primary; }
.body { padding: 24rpx 32rpx 48rpx; }

/* 预填结果提示条：常驻可见（可手动关闭），不是 toast */
.draft-notice {
  display: flex;
  align-items: flex-start;
  gap: 16rpx;
  margin-bottom: 20rpx;
  padding: 20rpx 24rpx;
  background: rgba(251, 191, 36, 0.1);
  border: 1rpx solid rgba(251, 191, 36, 0.4);
  border-radius: 20rpx;
  opacity: 0;
  animation: guide-in $motion-base both;
}
.draft-notice-body { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.draft-notice-title { font-size: 26rpx; font-weight: 600; color: $warning-color; }
.draft-notice-text { margin-top: 8rpx; font-size: 24rpx; line-height: 1.5; color: $dark-text-primary; }
.draft-notice-close {
  flex-shrink: 0;
  padding: 6rpx 20rpx;
  font-size: 24rpx;
  color: $warning-color;
  background: rgba(251, 191, 36, 0.12);
  border: 1rpx solid rgba(251, 191, 36, 0.4);
  border-radius: 999rpx;
}

.toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24rpx; gap: 16rpx; }
.toolbar-count { flex-shrink: 0; font-size: 24rpx; color: $dark-text-secondary; }
.toolbar-actions { display: flex; align-items: center; gap: 16rpx; }
.btn-tpl { display: inline-flex; align-items: center; justify-content: center; min-height: 88rpx; padding: 0 24rpx; background: rgba(6, 182, 212, 0.1); border: 2rpx solid $neon-cyan; color: $neon-cyan; border-radius: 28rpx; font-size: 24rpx;
  &:active { transform: scale(0.97); }
  &.disabled { opacity: 0.5; }
}
.btn-add { display: inline-flex; align-items: center; justify-content: center; min-height: 88rpx; padding: 0 28rpx; background: $neon-gradient; color: $dark-bg-primary; border-radius: 28rpx; font-size: 26rpx; font-weight: 600; box-shadow: 0 0 16rpx $neon-purple-glow;
  &:active { transform: scale(0.97); }
}


/*
 * 定价引导卡：进页第一眼回答「我该怎么定价」。
 * 用青色描边（= 平台建议/参考色，与「一键预填平台模板」按钮同色系）而非品牌橙，
 * 因为橙色在本页已经属于主行动按钮（新增套餐），引导卡不能抢主 CTA 的注意力。
 */
.pricing-guide {
  display: flex;
  flex-direction: column;
  margin-bottom: 24rpx;
  padding: 24rpx;
  background: rgba(6, 182, 212, 0.08);
  border: 1rpx solid rgba(6, 182, 212, 0.28);
  border-radius: 20rpx;
  opacity: 0;
  animation: guide-in $motion-base both;
}

@keyframes guide-in {
  from {
    opacity: 0;
    transform: translateY(-8rpx);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.pricing-guide-title {
  font-size: 26rpx;
  font-weight: 600;
  color: $neon-cyan;
}

.pricing-guide-body {
  margin-top: 10rpx;
  font-size: 26rpx;
  line-height: 1.5;
  color: $dark-text-primary;
}

.pricing-guide-example {
  margin-top: 8rpx;
  font-size: 24rpx;
  line-height: 1.5;
  color: $dark-text-secondary;
}

.empty { display: flex; flex-direction: column; align-items: center; padding: 120rpx 0; }

.empty-icon { font-size: 80rpx; color: $neon-purple; text-shadow: 0 0 24rpx $neon-purple-glow; }
.empty-text { font-size: 30rpx; color: $dark-text-secondary; margin-top: 24rpx; }
.empty-hint { font-size: 24rpx; color: $dark-text-secondary; margin-top: 12rpx; padding: 0 40rpx; text-align: center; line-height: 1.5; }
.empty-cta { margin-top: 40rpx; }
.empty-cta-tpl { margin-top: 20rpx; min-height: 80rpx; padding: 0 28rpx; display: inline-flex; align-items: center; justify-content: center; font-size: 24rpx; color: $neon-cyan; background: rgba(6, 182, 212, 0.1); border: 1rpx solid rgba(6, 182, 212, 0.4); border-radius: 24rpx; }

.service-list { display: flex; flex-direction: column; gap: 24rpx; }
.service-card { background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 20rpx; padding: 28rpx; box-shadow: 0 4rpx 24rpx rgba(0, 0, 0, 0.25); }
.service-card.is-draft { border-color: rgba(251, 191, 36, 0.35); }
.service-head { display: flex; align-items: center; justify-content: space-between; gap: 16rpx; }
.service-name { flex: 1; min-width: 0; font-size: 28rpx; font-weight: 600; color: $dark-text-primary; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.price-tag { flex-shrink: 0; font-size: 30rpx; font-weight: 700; padding: 6rpx 20rpx; border-radius: 24rpx;
  &.is-fixed { color: $neon-cyan; background: rgba(6, 182, 212, 0.12); border: 1rpx solid rgba(6, 182, 212, 0.4); }
  &.is-mutual { color: $success-color; background: rgba(34, 197, 94, 0.12); border: 1rpx solid rgba(34, 197, 94, 0.4); }
  &.is-negotiable { color: $neon-purple-bright; background: $neon-purple-dim; border: 1rpx solid rgba(168, 85, 247, 0.4); }
}
.service-meta { display: flex; align-items: center; gap: 16rpx; margin-top: 16rpx; }
.meta-item { font-size: 24rpx; color: $dark-text-secondary; }
.status-tag { flex-shrink: 0; font-size: 24rpx; padding: 4rpx 14rpx; border-radius: 20rpx;
  &.is-active { color: $success-color; background: rgba(34, 197, 94, 0.12); border: 1rpx solid rgba(34, 197, 94, 0.4); }
  &.is-down { color: $dark-text-secondary; background: rgba(100, 116, 139, 0.12); border: 1rpx solid rgba(100, 116, 139, 0.4); }
  /* 草稿：琥珀虚线，和「下架」的灰实线一眼可分 */
  &.is-draft { color: $warning-color; background: rgba(251, 191, 36, 0.12); border: 1rpx dashed rgba(251, 191, 36, 0.6); }
}
.service-desc { display: block; font-size: 24rpx; color: $dark-text-secondary; margin-top: 14rpx; line-height: 1.5; }
.draft-hint { display: block; font-size: 22rpx; color: $warning-color; margin-top: 14rpx; line-height: 1.5; }
.service-actions { margin-top: 24rpx; display: flex; justify-content: flex-end; gap: 16rpx; }
.btn-toggle { display: inline-flex; align-items: center; justify-content: center; min-height: 88rpx; padding: 0 28rpx; font-size: 26rpx; color: $dark-text-primary; background: rgba(255, 255, 255, 0.06); border: 1rpx solid $dark-border; border-radius: 24rpx;
  &.is-busy { opacity: 0.5; }
  &.is-publish { color: $warning-color; background: rgba(251, 191, 36, 0.12); border-color: rgba(251, 191, 36, 0.5); }
  &:active { transform: scale(0.95); }
}
.btn-edit { display: inline-flex; align-items: center; justify-content: center; min-height: 88rpx; padding: 0 28rpx; font-size: 26rpx; color: $neon-cyan; background: rgba(6, 182, 212, 0.1); border: 1rpx solid rgba(6, 182, 212, 0.4); border-radius: 24rpx;
  &:active { transform: scale(0.95); }
}
.btn-del { display: inline-flex; align-items: center; justify-content: center; min-height: 88rpx; padding: 0 28rpx; font-size: 26rpx; color: $error-color; background: rgba(239, 68, 68, 0.1); border: 1rpx solid rgba(239, 68, 68, 0.4); border-radius: 24rpx;
  &:active { transform: scale(0.95); }
}

.form-card { background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 24rpx; padding: 40rpx 32rpx; box-shadow: 0 4rpx 24rpx rgba(0, 0, 0, 0.25); }
.form-header { margin-bottom: 32rpx; }
.form-title { display: block; font-size: 32rpx; font-weight: 600; color: $dark-text-primary; }
.form-sub { display: block; font-size: 24rpx; color: $dark-text-secondary; margin-top: 8rpx; }
.form-draft-tip { display: flex; flex-direction: column; margin-bottom: 28rpx; padding: 20rpx 24rpx; background: rgba(251, 191, 36, 0.1); border: 1rpx solid rgba(251, 191, 36, 0.4); border-radius: 16rpx; }
.form-draft-tip-title { font-size: 26rpx; font-weight: 600; color: $warning-color; }
.form-draft-tip-text { margin-top: 8rpx; font-size: 24rpx; line-height: 1.5; color: $dark-text-primary; }
.field { margin-bottom: 32rpx; }
.field-label { display: block; font-size: 26rpx; color: $dark-text-secondary; margin-bottom: 16rpx; }
.field-input { height: 88rpx; background: $dark-bg-secondary; border: 1rpx solid $dark-border; border-radius: 16rpx; padding: 0 24rpx; font-size: 28rpx; color: $dark-text-primary; box-sizing: border-box; }
.field-textarea { width: 100%; height: 200rpx; background: $dark-bg-secondary; border: 1rpx solid $dark-border; border-radius: 16rpx; padding: 20rpx 24rpx; font-size: 28rpx; color: $dark-text-primary; box-sizing: border-box; }
.field-hint { display: block; font-size: 24rpx; color: $dark-text-secondary; margin-top: 12rpx; }

.mode-row { display: flex; gap: 16rpx; }
.mode-item { flex: 1; text-align: center; padding: 20rpx 0; background: $dark-bg-secondary; border: 2rpx solid $dark-border; border-radius: 16rpx; font-size: 28rpx; color: $dark-text-secondary;
  &.active { border-color: $neon-purple; color: $neon-purple; background: $neon-purple-dim; box-shadow: 0 0 12rpx $neon-purple-glow; font-weight: 500; }
}
.field-error { display: block; font-size: 24rpx; color: $error-color; margin-top: 12rpx; }
.field-input.has-error { border-color: rgba(239, 68, 68, 0.6); }

.btn-submit { margin-top: 40rpx; display: flex; align-items: center; justify-content: center; min-height: 96rpx; padding: 0; background: $neon-gradient; color: $dark-bg-primary; border-radius: 44rpx; font-size: 30rpx; font-weight: 700; box-shadow: 0 0 24rpx $neon-purple-glow;
  &:active { transform: scale(0.97); }
  &.disabled { opacity: 0.5; }
}
.btn-cancel { margin-top: 20rpx; display: flex; align-items: center; justify-content: center; min-height: 88rpx; font-size: 26rpx; color: $dark-text-secondary; }

/* 平台模板面板（底部弹出，遮罩层点掉） */
.tpl-mask { position: fixed; top: 0; left: 0; right: 0; bottom: 0; z-index: 200; display: flex; align-items: flex-end; background: rgba(0, 0, 0, 0.6); }
.tpl-sheet { width: 100%; padding: 32rpx 32rpx calc(32rpx + env(safe-area-inset-bottom)); background: $dark-bg-secondary; border-top: 1rpx solid $dark-border; border-top-left-radius: 28rpx; border-top-right-radius: 28rpx; box-sizing: border-box; animation: sheet-in $motion-base both; }

@keyframes sheet-in {
  from { opacity: 0; transform: translateY(48rpx); }
  to { opacity: 1; transform: translateY(0); }
}

.tpl-sheet-head { display: flex; align-items: center; justify-content: space-between; }
.tpl-sheet-title { font-size: 32rpx; font-weight: 600; color: $dark-text-primary; }
.tpl-sheet-close { font-size: 32rpx; color: $dark-text-secondary; padding: 0 12rpx; }
.tpl-sheet-sub { display: block; margin-top: 12rpx; font-size: 24rpx; line-height: 1.5; color: $dark-text-secondary; }
.tpl-sheet-skip { display: block; margin-top: 8rpx; font-size: 24rpx; line-height: 1.5; color: $warning-color; }
.tpl-state { padding: 48rpx 0; text-align: center; font-size: 26rpx; color: $dark-text-secondary; }
.tpl-list { margin-top: 16rpx; }
.tpl-row { display: flex; align-items: center; justify-content: space-between; gap: 16rpx; padding: 20rpx 0; border-bottom: 1rpx solid $dark-border; }
.tpl-row-main { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.tpl-row-name { font-size: 28rpx; color: $dark-text-primary; }
.tpl-row-meta { margin-top: 6rpx; font-size: 24rpx; color: $dark-text-secondary; }
.tpl-row-fill { flex-shrink: 0; display: inline-flex; align-items: center; justify-content: center; min-height: 72rpx; padding: 0 24rpx; font-size: 24rpx; color: $neon-cyan; background: rgba(6, 182, 212, 0.1); border: 1rpx solid rgba(6, 182, 212, 0.4); border-radius: 20rpx;
  &:active { transform: scale(0.96); }
}
.tpl-primary { margin-top: 28rpx; display: flex; align-items: center; justify-content: center; min-height: 92rpx; background: $neon-gradient; color: $dark-bg-primary; border-radius: 44rpx; font-size: 28rpx; font-weight: 700; box-shadow: 0 0 24rpx $neon-purple-glow;
  &:active { transform: scale(0.97); }
  &.disabled { opacity: 0.5; }
}
.tpl-cancel { margin-top: 12rpx; display: flex; align-items: center; justify-content: center; min-height: 80rpx; font-size: 26rpx; color: $dark-text-secondary; }
</style>
