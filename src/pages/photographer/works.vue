<template>
  <view class="page">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />
    <view class="nav-bar" :style="{ paddingTop: statusBarHeight + 'px' }">
      <text class="back-arrow" @click="goBack">‹</text>
      <text class="nav-title">作品管理</text>
    </view>

    <view class="body">
      <view class="toolbar">
        <text class="toolbar-count">共 {{ works.length }} 件作品</text>
        <view class="btn-add" @click="openForm">＋ 新增作品</view>
      </view>

      <view v-if="mode === 'list'">
        <view v-if="loading" class="loading-tip">加载中...</view>

        <view v-else-if="works.length === 0" class="empty">
          <text class="empty-icon">◆</text>
          <text class="empty-text">暂无作品</text>
          <text class="empty-hint">点击上方「新增作品」发布第一件作品</text>
        </view>

        <view v-else class="work-list">
          <view v-for="w in works" :key="w.id" class="work-card">
            <image :src="w.images[0]" class="work-thumb" mode="aspectFill" />
            <view class="work-info">
              <view class="work-title-row">
                <text class="work-title">{{ w.title }}</text>
                <text class="status-tag" :class="(w.status || 'active') === 'active' ? 'is-active' : 'is-down'">
                  {{ (w.status || 'active') === 'active' ? '上架' : '下架' }}
                </text>
              </view>
              <text v-if="w.description" class="work-desc">{{ w.description }}</text>
              <text class="work-meta">发布于 {{ formatTime(w.created_at || w.createdAt) }} · {{ (w.images || []).length }} 图</text>
              <view class="work-actions">
                <view class="btn-edit" @click="openEdit(w)">编辑</view>
                <view class="btn-del" @click="confirmDelete(w)">删除</view>
              </view>
            </view>
          </view>
        </view>
      </view>

      <view v-else class="form-card">
        <view class="form-header">
          <text class="form-title">{{ editingId ? '编辑作品' : '新增作品' }}</text>
          <text class="form-sub">发布后立即展示在你的摄影师主页作品集</text>
        </view>

        <view class="field">
          <text class="field-label">作品标题</text>
          <input
            v-model="form.title"
            class="field-input"
            placeholder="例如：原神-雷电将军 正片"
            placeholder-style="color: #64748b"
            maxlength="50"
          />
        </view>

        <view class="field">
          <text class="field-label">图片链接（至少 1 张，最多 5 张）</text>
          <view v-for="(img, i) in form.images" :key="i" class="img-row">
            <input
              v-model="form.images[i]"
              class="field-input"
              placeholder="https://picsum.photos/seed/xxx/600/400"
              placeholder-style="color: #64748b"
            />
            <view v-if="form.images.length > 1" class="img-del" @click="removeImage(i)">×</view>
          </view>
          <view class="img-actions">
            <view v-if="form.images.length < 5" class="btn-add-img" @click="addImage">+ 填链接</view>
            <view v-if="form.images.length < 5" class="btn-add-img btn-upload-img" @click="uploadImages">从相册上传</view>
          </view>
        </view>

        <view class="field">
          <text class="field-label">作品说明（选填）</text>
          <textarea
            v-model="form.description"
            class="field-textarea"
            placeholder="介绍这组作品的拍摄主题、服装、场地…"
            placeholder-style="color: #64748b"
            maxlength="300"
          />
        </view>

        <view class="btn-submit" :class="{ disabled: !canSubmit }" @click="submit">
          {{ submitting ? (editingId ? '保存中...' : '发布中...') : (editingId ? '保存修改' : '发布作品') }}
        </view>
        <view class="btn-cancel" @click="cancelForm">返回列表</view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { reactive, ref, computed } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { uploadWork, updateWork, getMyWorks, deleteWork, pickAndUploadImages, type MyWork } from '@/api/index'
import { ApiError } from '@/api/client'
import { useUserStore } from '@/stores/user'

const statusBarHeight = ref(44)
const userStore = useUserStore()

const mode = ref<'list' | 'form'>('list')
const editingId = ref<number | null>(null)
const loading = ref(true)
const works = ref<MyWork[]>([])
const submitting = ref(false)

const form = reactive({
  title: '',
  images: [''] as string[],
  description: ''
})

const nonEmptyImages = computed(() => form.images.map(s => s.trim()).filter(Boolean))
const canSubmit = computed(
  () => form.title.trim() !== '' && nonEmptyImages.value.length >= 1 && !submitting.value
)

onShow(() => {
  const sysInfo = uni.getSystemInfoSync()
  statusBarHeight.value = sysInfo.statusBarHeight || 44
  if (!userStore.user) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => {
      uni.navigateTo({ url: '/pages/login/index?redirect=/pages/photographer/works' })
    }, 800)
    return
  }
  if (mode.value === 'list') {
    loadWorks()
  }
})

async function loadWorks() {
  loading.value = true
  try {
    const res = await getMyWorks()
    // backend marshals empty slice as null — normalize to []
    works.value = (res || []).map((w) => ({ ...w, images: w.images || [] }))
  } catch (e) {
    if (e instanceof ApiError && e.status === 403) {
      uni.showToast({ title: '仅摄影师可管理作品', icon: 'none' })
      setTimeout(() => uni.navigateBack(), 1200)
      return
    }
    works.value = []
  } finally {
    loading.value = false
  }
}

function openForm() {
  editingId.value = null
  form.title = ''
  form.images = ['']
  form.description = ''
  mode.value = 'form'
}

function openEdit(w: MyWork) {
  editingId.value = w.id
  form.title = w.title
  form.images = w.images && w.images.length ? [...w.images] : ['']
  form.description = w.description || ''
  mode.value = 'form'
}

function cancelForm() {
  editingId.value = null
  mode.value = 'list'
}

function addImage() {
  if (form.images.length < 5) {
    form.images.push('')
  }
}

function removeImage(i: number) {
  form.images.splice(i, 1)
}

async function uploadImages() {
  const remain = 5 - form.images.map(s => s.trim()).filter(Boolean).length
  if (remain <= 0) return
  try {
    const urls = await pickAndUploadImages(remain)
    const merged = [...form.images.map(s => s.trim()).filter(Boolean), ...urls].slice(0, 5)
    form.images = merged.length ? merged : ['']
  } catch {
    uni.showToast({ title: '上传失败', icon: 'none' })
  }
}

async function submit() {
  if (!canSubmit.value) return
  const id = editingId.value
  submitting.value = true
  uni.showLoading({ title: id ? '保存中...' : '发布中...' })
  try {
    const payload = {
      title: form.title.trim(),
      images: nonEmptyImages.value,
      description: form.description.trim() || undefined
    }
    if (id) {
      await updateWork(id, payload)
    } else {
      await uploadWork(payload)
    }
    uni.showToast({ title: id ? '已保存' : '发布成功', icon: 'success' })
    editingId.value = null
    mode.value = 'list'
    await loadWorks()
  } catch {
    uni.showToast({ title: id ? '保存失败' : '发布失败', icon: 'none' })
  } finally {
    submitting.value = false
    uni.hideLoading()
  }
}

function confirmDelete(w: MyWork) {
  uni.showModal({
    title: '删除作品',
    content: `确定删除「${w.title}」吗？删除后不可恢复`,
    confirmColor: '#ef4444',
    success: async (res) => {
      if (!res.confirm) return
      uni.showLoading({ title: '删除中...' })
      try {
        await deleteWork(w.id)
        uni.showToast({ title: '已删除', icon: 'success' })
        await loadWorks()
      } catch {
        uni.showToast({ title: '删除失败', icon: 'none' })
      } finally {
        uni.hideLoading()
      }
    }
  })
}

function formatTime(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
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

.toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24rpx; }
.toolbar-count { font-size: 24rpx; color: $dark-text-tertiary; }
.btn-add { padding: 12rpx 28rpx; background: $neon-gradient; color: #fff; border-radius: 28rpx; font-size: 24rpx; box-shadow: 0 0 16rpx $neon-purple-glow;
  &:active { transform: scale(0.97); }
}

.loading-tip { text-align: center; padding: 80rpx 0; font-size: 26rpx; color: $dark-text-tertiary; }

.empty { display: flex; flex-direction: column; align-items: center; padding: 120rpx 0; }
.empty-icon { font-size: 80rpx; color: $neon-purple; text-shadow: 0 0 24rpx $neon-purple-glow; }
.empty-text { font-size: 30rpx; color: $dark-text-secondary; margin-top: 24rpx; }
.empty-hint { font-size: 24rpx; color: $dark-text-tertiary; margin-top: 12rpx; }

.work-list { display: flex; flex-direction: column; gap: 24rpx; }
.work-card { display: flex; background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 20rpx; overflow: hidden; box-shadow: 0 4rpx 24rpx rgba(0, 0, 0, 0.25); }
.work-thumb { width: 220rpx; height: 220rpx; flex-shrink: 0; background: $dark-bg-secondary; }
.work-info { flex: 1; padding: 20rpx 24rpx; display: flex; flex-direction: column; min-width: 0; }
.work-title-row { display: flex; align-items: center; gap: 12rpx; }
.work-title { font-size: 28rpx; font-weight: 600; color: $dark-text-primary; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.status-tag { flex-shrink: 0; font-size: 20rpx; padding: 4rpx 14rpx; border-radius: 20rpx;
  &.is-active { color: $success-color; background: rgba(34, 197, 94, 0.12); border: 1rpx solid rgba(34, 197, 94, 0.4); }
  &.is-down { color: $dark-text-tertiary; background: rgba(100, 116, 139, 0.12); border: 1rpx solid rgba(100, 116, 139, 0.4); }
}
.work-desc { font-size: 22rpx; color: $dark-text-tertiary; margin-top: 10rpx; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.work-meta { font-size: 20rpx; color: $dark-text-tertiary; margin-top: 10rpx; }
.work-actions { margin-top: auto; display: flex; justify-content: flex-end; gap: 16rpx; }
.btn-edit { padding: 8rpx 24rpx; font-size: 22rpx; color: $neon-cyan; background: rgba(6, 182, 212, 0.1); border: 1rpx solid rgba(6, 182, 212, 0.4); border-radius: 24rpx;
  &:active { transform: scale(0.95); }
}
.btn-del { padding: 8rpx 24rpx; font-size: 22rpx; color: $error-color; background: rgba(239, 68, 68, 0.1); border: 1rpx solid rgba(239, 68, 68, 0.4); border-radius: 24rpx;
  &:active { transform: scale(0.95); }
}

.form-card { background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 24rpx; padding: 40rpx 32rpx; box-shadow: 0 4rpx 24rpx rgba(0, 0, 0, 0.25); }
.form-header { margin-bottom: 32rpx; }
.form-title { display: block; font-size: 32rpx; font-weight: 600; color: $dark-text-primary; }
.form-sub { display: block; font-size: 22rpx; color: $dark-text-tertiary; margin-top: 8rpx; }
.field { margin-bottom: 32rpx; }
.field-label { display: block; font-size: 26rpx; color: $dark-text-secondary; margin-bottom: 16rpx; }
.field-input { height: 88rpx; background: $dark-bg-secondary; border: 1rpx solid $dark-border; border-radius: 16rpx; padding: 0 24rpx; font-size: 28rpx; color: $dark-text-primary; }
.field-textarea { width: 100%; height: 200rpx; background: $dark-bg-secondary; border: 1rpx solid $dark-border; border-radius: 16rpx; padding: 20rpx 24rpx; font-size: 28rpx; color: $dark-text-primary; box-sizing: border-box; }
.img-row { display: flex; align-items: center; gap: 12rpx; margin-bottom: 16rpx;
  .field-input { flex: 1; }
}
.img-del { width: 56rpx; height: 56rpx; display: flex; align-items: center; justify-content: center; font-size: 36rpx; color: $error-color; background: rgba(239, 68, 68, 0.1); border-radius: 50%; flex-shrink: 0; }
.btn-add-img { padding: 14rpx 0; text-align: center; font-size: 24rpx; color: $neon-cyan; border: 2rpx dashed rgba(6, 182, 212, 0.4); border-radius: 16rpx; }
.img-actions { display: flex; gap: 16rpx; }
.img-actions .btn-add-img { flex: 1; }

.btn-submit { margin-top: 40rpx; text-align: center; padding: 24rpx 0; background: $neon-gradient; color: #fff; border-radius: 44rpx; font-size: 30rpx; font-weight: 600; box-shadow: 0 0 24rpx $neon-purple-glow;
  &:active { transform: scale(0.97); }
  &.disabled { opacity: 0.5; }
}
.btn-cancel { margin-top: 20rpx; text-align: center; padding: 20rpx 0; font-size: 26rpx; color: $dark-text-tertiary; }
</style>
