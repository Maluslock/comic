<template>
  <view class="page">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />
    <view class="nav-bar" :style="{ paddingTop: statusBarHeight + 'px' }">
      <text class="back-arrow" @click="goBack">‹</text>
      <text class="nav-title">认证申请</text>
    </view>

    <view class="body">
      <view v-if="status === 'loading'" class="loading-tip">加载中...</view>

      <view v-else-if="status === 'pending'" class="status-card pending">
        <text class="status-icon">◇</text>
        <text class="status-title">审核中</text>
        <text class="status-meta">提交时间：{{ formatTime(application?.createdAt) }}</text>
        <text class="status-hint">管理员审核通过后，黄V认证将立即生效</text>
      </view>

      <view v-else-if="status === 'approved'" class="status-card approved">
        <text class="status-icon">★</text>
        <text class="status-title">已通过 · 黄V已生效</text>
        <text class="status-hint">恭喜！你的摄影师主页已展示认证标识</text>
      </view>

      <view v-else-if="status === 'rejected' && !reapplying" class="status-card rejected">
        <text class="status-icon">⚠</text>
        <text class="status-title">已驳回</text>
        <text class="status-reason">驳回理由：{{ application?.reviewReason || '暂无' }}</text>
        <view class="btn-reapply" @click="reapplying = true">重新申请</view>
      </view>

      <view v-else class="form-card">
        <view class="form-header">
          <text class="form-title">摄影师认证申请</text>
          <text class="form-sub">提交样片链接与作品说明，审核通过后获得黄V认证</text>
        </view>

        <view class="field">
          <text class="field-label">样片链接（至少 1 张，最多 5 张）</text>
          <view v-for="(img, i) in form.evidenceImages" :key="i" class="img-row">
            <input
              v-model="form.evidenceImages[i]"
              class="field-input"
              placeholder="https://picsum.photos/seed/xxx/600/400"
              placeholder-style="color: #64748b"
            />
            <view class="img-del" @click="removeImage(i)">×</view>
          </view>
          <view class="img-actions">
            <view v-if="form.evidenceImages.length < 5" class="btn-add" @click="addImage">+ 填链接</view>
            <view v-if="form.evidenceImages.length < 5" class="btn-add btn-upload" @click="uploadImages">从相册上传</view>
          </view>
        </view>

        <view class="field">
          <text class="field-label">作品说明</text>
          <textarea
            v-model="form.evidenceDesc"
            class="field-textarea"
            placeholder="介绍你的作品数量、拍摄题材、成片质量…"
            placeholder-style="color: #64748b"
            maxlength="300"
          />
        </view>

        <view class="btn-submit" :class="{ disabled: !canSubmit }" @click="submit">
          {{ submitting ? '提交中...' : '提交认证申请' }}
        </view>
      </view>

      <view v-if="history.length" class="history-card">
        <text class="history-title">申请历史</text>
        <view v-for="h in history" :key="h.id" class="history-item">
          <view class="history-head">
            <text class="history-status" :class="h.status">{{ statusLabel(h.status) }}</text>
            <text class="history-time">{{ formatTime(h.createdAt) }}</text>
          </view>
          <text v-if="h.reviewReason" class="history-reason">{{ h.reviewReason }}</text>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { reactive, ref, computed } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { applyCertification, getMyCertApplication, getMyCertApplications, pickAndUploadImages, type CertApplication } from '@/api/index'
import { ApiError } from '@/api/client'
import { useUserStore } from '@/stores/user'

const statusBarHeight = ref(44)
const userStore = useUserStore()

const status = ref<'loading' | 'not-applied' | 'pending' | 'approved' | 'rejected'>('loading')
const application = ref<CertApplication | null>(null)
const history = ref<CertApplication[]>([])
const reapplying = ref(false)
const submitting = ref(false)

const form = reactive({
  evidenceImages: ['', '', ''] as string[],
  evidenceDesc: ''
})

const nonEmptyImages = computed(() => form.evidenceImages.map(s => s.trim()).filter(Boolean))
const canSubmit = computed(
  () => nonEmptyImages.value.length >= 1 && form.evidenceDesc.trim() !== '' && !submitting.value
)

onShow(() => {
  const sysInfo = uni.getSystemInfoSync()
  statusBarHeight.value = sysInfo.statusBarHeight || 44
  if (!userStore.user) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => {
      uni.navigateTo({ url: '/pages/login/index?redirect=/pages/photographer/cert-apply' })
    }, 800)
    return
  }
  loadStatus()
})

async function loadStatus() {
  status.value = 'loading'
  try {
    const res = await getMyCertApplication()
    application.value = res.application
    status.value = res.application ? (res.application.status as typeof status.value) : 'not-applied'
  } catch (e) {
    if (e instanceof ApiError && e.status === 403) {
      uni.showToast({ title: '仅摄影师可申请认证', icon: 'none' })
      setTimeout(() => uni.navigateBack(), 1200)
    }
    status.value = 'not-applied'
  }

  try {
    const res = await getMyCertApplications()
    history.value = res.list || []
  } catch {
    history.value = []
  }
}

function statusLabel(s: string): string {
  if (s === 'approved') return '已通过'
  if (s === 'rejected') return '已驳回'
  if (s === 'pending') return '审核中'
  return s
}

function addImage() {
  if (form.evidenceImages.length < 5) {
    form.evidenceImages.push('')
  }
}

function removeImage(i: number) {
  if (form.evidenceImages.length > 1) {
    form.evidenceImages.splice(i, 1)
  }
}

async function uploadImages() {
  const remain = 5 - form.evidenceImages.map(s => s.trim()).filter(Boolean).length
  if (remain <= 0) return
  try {
    const urls = await pickAndUploadImages(remain)
    const merged = [...form.evidenceImages.map(s => s.trim()).filter(Boolean), ...urls].slice(0, 5)
    form.evidenceImages = merged.length ? merged : ['']
  } catch {
    uni.showToast({ title: '上传失败', icon: 'none' })
  }
}

async function submit() {
  if (!canSubmit.value) return
  submitting.value = true
  try {
    await applyCertification(nonEmptyImages.value, form.evidenceDesc.trim())
    uni.showToast({ title: '已提交，等待审核', icon: 'success' })
    reapplying.value = false
    await loadStatus()
  } catch (e) {
    uni.showToast({
      title: e instanceof ApiError && e.status === 409 ? '已有申请在审核中' : '提交失败',
      icon: 'none'
    })
  } finally {
    submitting.value = false
  }
}

function formatTime(iso?: string | null) {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
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

.loading-tip { text-align: center; padding: 120rpx 0; color: $dark-text-tertiary; font-size: 26rpx; }

.status-card { margin-top: 32rpx; background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 24rpx; padding: 64rpx 40rpx; display: flex; flex-direction: column; align-items: center; box-shadow: 0 4rpx 24rpx rgba(0, 0, 0, 0.25);
  .status-icon { font-size: 96rpx; margin-bottom: 24rpx; }
  .status-title { font-size: 36rpx; font-weight: 700; margin-bottom: 16rpx; }
  .status-meta { font-size: 24rpx; color: $dark-text-tertiary; margin-bottom: 12rpx; }
  .status-hint { font-size: 24rpx; color: $dark-text-secondary; text-align: center; margin-top: 8rpx; }
  .status-reason { font-size: 26rpx; color: $dark-text-secondary; text-align: center; margin: 8rpx 0 40rpx; }
  &.pending { border-color: rgba(245, 158, 11, 0.4);
    .status-icon, .status-title { color: $warning-color; text-shadow: 0 0 16rpx rgba(245, 158, 11, 0.4); }
  }
  &.approved { border-color: rgba(34, 197, 94, 0.4);
    .status-icon, .status-title { color: $success-color; text-shadow: 0 0 16rpx rgba(34, 197, 94, 0.4); }
  }
  &.rejected { border-color: rgba(239, 68, 68, 0.4);
    .status-icon, .status-title { color: $error-color; text-shadow: 0 0 16rpx rgba(239, 68, 68, 0.4); }
  }
}

.btn-reapply { padding: 16rpx 64rpx; background: $neon-gradient; @include on-neon-fill; border-radius: 32rpx; font-size: 28rpx; box-shadow: 0 0 20rpx $neon-purple-glow;
  &:active { transform: scale(0.97); }
}

.form-card { background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 24rpx; padding: 40rpx 32rpx; box-shadow: 0 4rpx 24rpx rgba(0, 0, 0, 0.25); }
.form-header { margin-bottom: 40rpx; }
.form-title { display: block; font-size: 34rpx; font-weight: 700; color: $dark-text-primary; margin-bottom: 12rpx; }
.form-sub { display: block; font-size: 24rpx; color: $dark-text-tertiary; }

.field { margin-bottom: 36rpx; }
.field-label { display: block; font-size: 26rpx; color: $dark-text-secondary; margin-bottom: 16rpx; }
.img-row { display: flex; align-items: center; gap: 16rpx; margin-bottom: 16rpx;
  .field-input { flex: 1; }
}
.img-del { width: 64rpx; height: 64rpx; flex-shrink: 0; display: flex; align-items: center; justify-content: center; background: $dark-bg-secondary; border: 1rpx solid $dark-border; border-radius: 12rpx; color: $dark-text-tertiary; font-size: 36rpx;
  &:active { color: $error-color; }
}
.btn-add { margin-top: 8rpx; text-align: center; padding: 16rpx 0; border: 2rpx dashed $dark-border; border-radius: 16rpx; color: $neon-cyan; font-size: 26rpx; }
.img-actions { display: flex; gap: 16rpx; }
.img-actions .btn-add { flex: 1; }

.field-input { height: 88rpx; background: $dark-bg-secondary; border: 1rpx solid $dark-border; border-radius: 16rpx; padding: 0 24rpx; font-size: 28rpx; color: $dark-text-primary; }
.field-textarea { width: 100%; height: 200rpx; background: $dark-bg-secondary; border: 1rpx solid $dark-border; border-radius: 16rpx; padding: 20rpx 24rpx; font-size: 28rpx; color: $dark-text-primary; box-sizing: border-box; }

.btn-submit { margin-top: 48rpx; text-align: center; padding: 24rpx 0; background: $neon-gradient; @include on-neon-fill; border-radius: 44rpx; font-size: 30rpx; font-weight: 600; box-shadow: 0 0 24rpx $neon-purple-glow;
  &:active { transform: scale(0.97); box-shadow: 0 0 12rpx $neon-purple-glow; }
  &.disabled { opacity: 0.4; box-shadow: none; }
}

.history-card { margin-top: 32rpx; background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 24rpx; padding: 32rpx; box-shadow: 0 4rpx 24rpx rgba(0, 0, 0, 0.25); }
.history-title { display: block; font-size: 28rpx; font-weight: 600; color: $dark-text-primary; margin-bottom: 20rpx; }
.history-item { padding: 20rpx 0; border-top: 1rpx solid $dark-border;
  &:first-of-type { border-top: none; }
}
.history-head { display: flex; justify-content: space-between; align-items: center; }
.history-status { font-size: 26rpx; color: $dark-text-secondary;
  &.approved { color: $success-color; }
  &.rejected { color: $error-color; }
  &.pending { color: $neon-cyan; }
}
.history-time { font-size: 22rpx; color: $dark-text-tertiary; }
.history-reason { display: block; font-size: 24rpx; color: $dark-text-tertiary; margin-top: 10rpx; }
</style>
