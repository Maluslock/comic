<template>
  <view class="page">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />
    <view class="nav-bar" :style="{ paddingTop: statusBarHeight + 'px' }">
      <text class="back-arrow" @click="goBack">‹</text>
      <text class="nav-title">主页管理</text>
    </view>

    <view class="body">
      <view v-if="loading" class="empty-hint">加载中...</view>

      <view v-else class="form-card">
        <view class="field">
          <text class="field-label">头像</text>
          <view class="avatar-row">
            <image
              v-if="form.avatar"
              :src="form.avatar"
              class="avatar-preview"
              mode="aspectFill"
              @error="form.avatar = DEFAULT_AVATAR"
            />
            <view v-else class="avatar-preview avatar-empty">
              <text class="avatar-empty-mark">◇</text>
            </view>
            <view class="avatar-actions">
              <view class="btn-avatar" @click="pickAvatar">从相册上传</view>
              <input
                v-model="form.avatar"
                class="field-input avatar-input"
                placeholder="或粘贴图片链接"
                placeholder-style="color: #64748b"
              />
            </view>
          </view>
        </view>

        <view class="field">
          <text class="field-label">昵称</text>
          <input
            v-model="form.name"
            class="field-input"
            placeholder="例如：阿米摄影"
            placeholder-style="color: #64748b"
            maxlength="20"
          />
        </view>

        <view class="field">
          <text class="field-label">简介</text>
          <textarea
            v-model="form.description"
            class="field-textarea"
            placeholder="介绍你的拍摄风格、器材、经验…"
            placeholder-style="color: #64748b"
            maxlength="200"
          />
        </view>

        <view class="field">
          <text class="field-label">城市</text>
          <picker mode="selector" :range="cities" :value="cityIndex" @change="onCityChange">
            <view class="field-input picker-value" :class="{ 'is-placeholder': !form.location }">
              {{ form.location || '请选择城市' }}
            </view>
          </picker>
        </view>

        <view class="field">
          <text class="field-label">接单模式</text>
          <view class="mode-row">
            <view
              v-for="m in modes"
              :key="m.value"
              class="mode-item"
              :class="{ active: form.mode === m.value }"
              @click="form.mode = m.value"
            >
              {{ m.label }}
            </view>
          </view>
          <text class="field-hint">{{ modeHint }}</text>
        </view>

        <view class="field">
          <text class="field-label">互勉说明</text>
          <textarea
            v-model="form.mutualIntro"
            class="field-textarea"
            placeholder="互勉合作说明、期望的创作类型…"
            placeholder-style="color: #64748b"
            maxlength="300"
          />
        </view>

        <view class="btn-submit" @click="save">保存</view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { reactive, ref, computed } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { ApiError } from '@/api/client'
import { getMyPhotographerProfile, updatePhotographerProfile, pickAndUploadImages } from '@/api/index'
import { useUserStore } from '@/stores/user'

const statusBarHeight = ref(44)
const loading = ref(true)
const userStore = useUserStore()
const DEFAULT_AVATAR = '/static/img/avatar-user.svg'

const modes = [
  { value: 'free', label: '互勉' },
  { value: 'pay', label: '收费' },
  { value: 'both', label: '两者' }
]

const cities = [
  '北京', '上海', '广州', '深圳', '杭州', '成都', '重庆', '武汉', '西安', '南京',
  '长沙', '厦门', '天津', '苏州', '郑州', '昆明', '青岛', '大连', '合肥', '福州',
  '济南', '沈阳', '哈尔滨', '南宁', '贵阳', '南昌', '太原', '兰州', '石家庄', '海口'
]

const form = reactive({
  name: '',
  description: '',
  location: '',
  mode: 'both',
  mutualIntro: '',
  avatar: ''
})

const modeHint = computed(() => {
  const map: Record<string, string> = {
    free: '互勉：免费互相创作，积累作品',
    pay: '收费：有偿接单约拍',
    both: '两者：互勉与收费都可接受'
  }
  return map[form.mode] || ''
})

const cityIndex = computed(() => {
  const i = cities.indexOf(form.location)
  return i >= 0 ? i : 0
})

function onCityChange(e: any) {
  const idx = Number(e.detail.value)
  form.location = cities[idx] || ''
}

async function pickAvatar() {
  try {
    const urls = await pickAndUploadImages(1)
    if (urls[0]) form.avatar = urls[0]
  } catch {
    uni.showToast({ title: '上传失败', icon: 'none' })
  }
}

onShow(() => {
  const sysInfo = uni.getSystemInfoSync()
  statusBarHeight.value = sysInfo.statusBarHeight || 44
  if (!userStore.user) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => {
      uni.navigateTo({ url: '/pages/login/index?redirect=/pages/photographer/profile-edit' })
    }, 800)
    return
  }
  loadProfile()
})

async function loadProfile() {
  loading.value = true
  try {
    const res = await getMyPhotographerProfile()
    form.name = res.name || ''
    form.description = res.description || ''
    form.location = res.location || ''
    form.mode = res.mode || 'both'
    form.mutualIntro = res.mutualIntro || ''
    form.avatar = res.avatar || ''
  } catch (e) {
    const code = e instanceof ApiError ? e.status : 0
    uni.showToast({ title: code === 403 ? '仅摄影师可编辑主页' : '加载失败', icon: 'none' })
    setTimeout(() => uni.navigateBack(), 800)
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!form.name.trim()) {
    uni.showToast({ title: '请填写昵称', icon: 'none' })
    return
  }
  uni.showLoading({ title: '保存中...' })
  try {
    await updatePhotographerProfile({
      name: form.name.trim(),
      description: form.description,
      location: form.location,
      mode: form.mode,
      mutualIntro: form.mutualIntro,
      avatar: form.avatar.trim()
    })
    uni.showToast({ title: '保存成功', icon: 'success' })
    setTimeout(() => uni.navigateBack(), 800)
  } catch (e) {
    const code = e instanceof ApiError ? e.status : 0
    uni.showToast({ title: code === 400 ? '参数有误' : code === 403 ? '仅摄影师可编辑主页' : '保存失败', icon: 'none' })
  } finally {
    uni.hideLoading()
  }
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

.empty-hint { text-align: center; padding-top: 200rpx; color: $dark-text-tertiary; }

.form-card { background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 24rpx; padding: 40rpx 32rpx; box-shadow: 0 4rpx 24rpx rgba(0, 0, 0, 0.25); }
.field { margin-bottom: 36rpx; }
.field-label { display: block; font-size: 26rpx; color: $dark-text-secondary; margin-bottom: 16rpx; }
.field-input { height: 88rpx; background: $dark-bg-secondary; border: 1rpx solid $dark-border; border-radius: 16rpx; padding: 0 24rpx; font-size: 28rpx; color: $dark-text-primary; }
.field-textarea { width: 100%; height: 200rpx; background: $dark-bg-secondary; border: 1rpx solid $dark-border; border-radius: 16rpx; padding: 20rpx 24rpx; font-size: 28rpx; color: $dark-text-primary; box-sizing: border-box; }
.field-hint { display: block; font-size: 22rpx; color: $dark-text-tertiary; margin-top: 12rpx; }
.picker-value { display: flex; align-items: center;
  &.is-placeholder { color: $dark-text-tertiary; }
}

.avatar-row { display: flex; gap: 24rpx; align-items: center; }
.avatar-preview { width: 140rpx; height: 140rpx; border-radius: 50%; border: 3rpx solid $neon-purple; box-shadow: 0 0 16rpx $neon-purple-glow; flex-shrink: 0; }
.avatar-empty { display: flex; align-items: center; justify-content: center; background: $dark-bg-secondary; border-style: dashed; border-color: $dark-border; box-shadow: none; }
.avatar-empty-mark { font-size: 48rpx; color: $dark-text-tertiary; }
.avatar-actions { flex: 1; display: flex; flex-direction: column; gap: 16rpx; min-width: 0; }
.btn-avatar { align-self: flex-start; padding: 12rpx 32rpx; font-size: 24rpx; color: $neon-purple; border: 1rpx solid $neon-purple; border-radius: 999rpx; box-shadow: 0 0 12rpx $neon-purple-glow; }
.avatar-input { width: 100%; }

.mode-row { display: flex; gap: 16rpx; }
.mode-item { flex: 1; text-align: center; padding: 20rpx 0; background: $dark-bg-secondary; border: 2rpx solid $dark-border; border-radius: 16rpx; font-size: 28rpx; color: $dark-text-secondary;
  &.active { border-color: $neon-purple; color: $neon-purple; background: $neon-purple-dim; box-shadow: 0 0 12rpx $neon-purple-glow; font-weight: 500; }
}

.btn-submit { margin-top: 48rpx; text-align: center; padding: 24rpx 0; background: $neon-gradient; @include on-neon-fill; border-radius: 44rpx; font-size: 30rpx; font-weight: 600; box-shadow: 0 0 24rpx $neon-purple-glow;
  &:active { transform: scale(0.97); box-shadow: 0 0 12rpx $neon-purple-glow; }
}
</style>
