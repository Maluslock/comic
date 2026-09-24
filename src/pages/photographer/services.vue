<template>
  <view class="page">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />
    <view class="nav-bar" :style="{ paddingTop: statusBarHeight + 'px' }">
      <text class="back-arrow" @click="goBack">‹</text>
      <text class="nav-title">我的套餐</text>
    </view>

    <view class="body">
      <view class="toolbar">
        <text class="toolbar-count">共 {{ services.length }} 个套餐</text>
        <view class="toolbar-actions">
          <view class="btn-tpl" :class="{ disabled: prefilling }" @click="prefillTemplates">一键预填平台模板</view>
          <view class="btn-add" @click="openForm">＋ 新增套餐</view>
        </view>
      </view>

      <view v-if="mode === 'list'">
        <view v-if="loading" class="loading-tip">加载中...</view>

        <view v-else-if="services.length === 0" class="empty">
          <text class="empty-icon">◇</text>
          <text class="empty-text">暂无套餐</text>
          <text class="empty-hint">点击下方按钮，或使用「一键预填平台模板」开始定价</text>
          <view class="btn-add empty-cta" @click="openForm">＋ 新增套餐</view>
        </view>

        <view v-else class="service-list">
          <view v-for="s in services" :key="s.id" class="service-card">
            <view class="service-head">
              <text class="service-name">{{ s.name }}</text>
              <text class="price-tag" :class="priceClass(s.price)">{{ renderPrice(s.price) }}</text>
            </view>
            <view class="service-meta">
              <text class="meta-item">时长 {{ formatDuration(s.duration) }}</text>
              <text class="status-tag" :class="isActive(s) ? 'is-active' : 'is-down'">
                {{ isActive(s) ? '上架' : '下架' }}
              </text>
            </view>
            <text v-if="s.description" class="service-desc">{{ s.description }}</text>
            <view class="service-actions">
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
          <text class="field-label">价格（元）</text>
          <input
            v-model="form.price"
            class="field-input"
            :class="{ 'has-error': priceError }"
            type="number"
            placeholder="留空 = 面议"
            placeholder-style="color: #64748b"
            @input="priceError = ''"
          />
          <text class="field-hint">留空 = 面议，填 0 = 互勉，填正数 = 固定价格</text>
          <text v-if="priceError" class="field-error">{{ priceError }}</text>
        </view>

        <view class="field">
          <text class="field-label">时长（分钟）</text>
          <input
            v-model="form.duration"
            class="field-input"
            type="number"
            placeholder="例如：60（分钟）"
            placeholder-style="color: #64748b"
          />
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
          {{ submitting ? '保存中...' : (editingId ? '保存修改' : '创建套餐') }}
        </view>
        <view class="btn-cancel" @click="cancelForm">返回列表</view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { reactive, ref, computed } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import {
  getMyServices,
  getServiceTemplates,
  createService,
  updateService,
  deleteService,
  type MyService
} from '@/api/index'
import { ApiError } from '@/api/client'
import { useUserStore } from '@/stores/user'

const statusBarHeight = ref(44)
const userStore = useUserStore()

const mode = ref<'list' | 'form'>('list')
const editingId = ref<number | null>(null)
const loading = ref(true)
const submitting = ref(false)
const prefilling = ref(false)
const services = ref<MyService[]>([])

const form = reactive({ name: '', price: '', duration: '60', description: '' })
const priceError = ref('')

const canSubmit = computed(() => form.name.trim() !== '' && !submitting.value)

onShow(() => {
  const sysInfo = uni.getSystemInfoSync()
  statusBarHeight.value = sysInfo.statusBarHeight || 44
  if (!userStore.user) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => {
      uni.navigateTo({ url: '/pages/login/index?redirect=/pages/photographer/services' })
    }, 800)
    return
  }
  if (mode.value === 'list') {
    loadServices()
  }
})

async function loadServices() {
  loading.value = true
  try {
    const res = await getMyServices()
    services.value = res?.list || []
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
}

function renderPrice(price: number | null): string {
  if (price === null || price === undefined) return '面议'
  if (price === 0) return '互勉'
  return `¥${price}`
}

function priceClass(price: number | null): string {
  if (price === null || price === undefined) return 'is-negotiable'
  if (price === 0) return 'is-mutual'
  return 'is-fixed'
}

function isActive(s: MyService): boolean {
  return s.isActive !== false
}

function formatDuration(minutes: number): string {
  if (!minutes || minutes <= 0) return '未填写'
  if (minutes < 60) return `${minutes} 分钟`
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  return m === 0 ? `${h} 小时` : `${h} 小时 ${m} 分钟`
}

function openForm() {
  editingId.value = null
  form.name = ''
  form.price = ''
  form.duration = '60'
  form.description = ''
  mode.value = 'form'
}

function openEdit(s: MyService) {
  editingId.value = s.id
  form.name = s.name
  form.price = s.price === null || s.price === undefined ? '' : String(s.price)
  form.duration = String(s.duration || 0)
  form.description = s.description || ''
  mode.value = 'form'
}

function cancelForm() {
  editingId.value = null
  mode.value = 'list'
}

function normalizedPrice(): number | null | undefined {
  const raw = form.price.trim()
  if (raw === '') return null
  const n = Number(raw)
  if (!isFinite(n) || n < 0) return undefined
  return Math.round(n)
}

function normalizedDuration(): number {
  const n = Number(form.duration.trim())
  if (!isFinite(n) || n < 0) return 0
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
  const id = editingId.value
  submitting.value = true
  uni.showLoading({ title: '保存中...' })
  try {
    const payload = {
      name: form.name.trim(),
      price,
      description: form.description.trim(),
      duration: normalizedDuration()
    }
    if (id) {
      await updateService(id, payload)
    } else {
      await createService(payload)
    }
    uni.showToast({ title: id ? '已保存' : '创建成功', icon: 'success' })
    editingId.value = null
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

function prefillTemplates() {
  if (prefilling.value) return
  uni.showModal({
    title: '预填平台模板',
    content: '将平台模板创建为你的套餐，可再次编辑价格与说明。继续？',
    success: async (res) => {
      if (!res.confirm) return
      await runPrefill()
    }
  })
}

async function runPrefill() {
  prefilling.value = true
  uni.showLoading({ title: '预填中...' })
  try {
    const templates = await getServiceTemplates()
    const list = templates || []
    if (list.length === 0) {
      uni.showToast({ title: '暂无平台模板', icon: 'none' })
      return
    }
    let created = 0
    for (const t of list) {
      try {
        await createService({
          name: t.name,
          price: t.price === null || t.price === undefined ? null : t.price,
          description: t.description || '',
          duration: t.duration || 0
        })
        created++
      } catch {
        // skip a single failed template so the rest still prefill
      }
    }
    uni.showToast({ title: created ? `已预填 ${created} 个套餐` : '预填失败', icon: 'none' })
    await loadServices()
  } catch (e) {
    if (e instanceof ApiError && e.status === 403) {
      uni.showToast({ title: '仅摄影师可管理套餐', icon: 'none' })
      setTimeout(() => uni.navigateBack(), 1200)
      return
    }
    uni.showToast({ title: '预填失败', icon: 'none' })
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
.nav-bar { position: fixed; top: 0; left: 0; right: 0; z-index: 99; height: 88rpx; display: flex; align-items: center; background: $dark-bg-primary; border-bottom: 1rpx solid $dark-border; }
.back-arrow { font-size: 48rpx; color: $dark-text-primary; padding: 0 24rpx; }
.nav-title { font-size: 34rpx; font-weight: 600; color: $dark-text-primary; }
.body { padding: 100rpx 32rpx 48rpx; }

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

.loading-tip { text-align: center; padding: 80rpx 0; font-size: 26rpx; color: $dark-text-secondary; }

.empty { display: flex; flex-direction: column; align-items: center; padding: 120rpx 0; }
.empty-icon { font-size: 80rpx; color: $neon-purple; text-shadow: 0 0 24rpx $neon-purple-glow; }
.empty-text { font-size: 30rpx; color: $dark-text-secondary; margin-top: 24rpx; }
.empty-hint { font-size: 24rpx; color: $dark-text-secondary; margin-top: 12rpx; }
.empty-cta { margin-top: 40rpx; }

.service-list { display: flex; flex-direction: column; gap: 24rpx; }
.service-card { background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 20rpx; padding: 28rpx; box-shadow: 0 4rpx 24rpx rgba(0, 0, 0, 0.25); }
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
}
.service-desc { display: block; font-size: 24rpx; color: $dark-text-secondary; margin-top: 14rpx; line-height: 1.5; }
.service-actions { margin-top: 24rpx; display: flex; justify-content: flex-end; gap: 16rpx; }
.btn-edit { display: inline-flex; align-items: center; justify-content: center; min-height: 88rpx; padding: 0 28rpx; font-size: 26rpx; color: $neon-cyan; background: rgba(6, 182, 212, 0.1); border: 1rpx solid rgba(6, 182, 212, 0.4); border-radius: 24rpx;
  &:active { transform: scale(0.95); }
}
.btn-del { display: inline-flex; align-items: center; justify-content: center; min-height: 88rpx; padding: 0 28rpx; font-size: 26rpx; color: $error-bright; background: rgba(239, 68, 68, 0.1); border: 1rpx solid rgba(239, 68, 68, 0.4); border-radius: 24rpx;
  &:active { transform: scale(0.95); }
}

.form-card { background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 24rpx; padding: 40rpx 32rpx; box-shadow: 0 4rpx 24rpx rgba(0, 0, 0, 0.25); }
.form-header { margin-bottom: 32rpx; }
.form-title { display: block; font-size: 32rpx; font-weight: 600; color: $dark-text-primary; }
.form-sub { display: block; font-size: 24rpx; color: $dark-text-secondary; margin-top: 8rpx; }
.field { margin-bottom: 32rpx; }
.field-label { display: block; font-size: 26rpx; color: $dark-text-secondary; margin-bottom: 16rpx; }
.field-input { height: 88rpx; background: $dark-bg-secondary; border: 1rpx solid $dark-border; border-radius: 16rpx; padding: 0 24rpx; font-size: 28rpx; color: $dark-text-primary; box-sizing: border-box; }
.field-textarea { width: 100%; height: 200rpx; background: $dark-bg-secondary; border: 1rpx solid $dark-border; border-radius: 16rpx; padding: 20rpx 24rpx; font-size: 28rpx; color: $dark-text-primary; box-sizing: border-box; }
.field-hint { display: block; font-size: 24rpx; color: $dark-text-secondary; margin-top: 12rpx; }
.field-error { display: block; font-size: 24rpx; color: $error-color; margin-top: 12rpx; }
.field-input.has-error { border-color: rgba(239, 68, 68, 0.6); }

.btn-submit { margin-top: 40rpx; display: flex; align-items: center; justify-content: center; min-height: 96rpx; padding: 0; background: $neon-gradient; color: $dark-bg-primary; border-radius: 44rpx; font-size: 30rpx; font-weight: 700; box-shadow: 0 0 24rpx $neon-purple-glow;
  &:active { transform: scale(0.97); }
  &.disabled { opacity: 0.5; }
}
.btn-cancel { margin-top: 20rpx; display: flex; align-items: center; justify-content: center; min-height: 88rpx; font-size: 26rpx; color: $dark-text-secondary; }
</style>
