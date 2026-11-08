<template>
  <page-meta
    root-background-color="#0c0d14"
    background-color="#0c0d14"
    background-color-top="#0c0d14"
    background-color-bottom="#0c0d14"
  />
  <view class="page">
    <template v-if="firstLoading">
      <view class="skeleton-page">
        <view class="sk-card sk-pad"><view class="sk-row"><view class="sk-circle sk-avatar-120" /><view class="sk-line sk-w-40" /></view></view>
        <ListSkeleton :rows="3" variant="card" />
      </view>
    </template>
    <template v-else>
    <view class="avatar-card">
      <image v-if="form.avatar" :src="form.avatar" class="avatar-preview" mode="aspectFill" />
      <view v-else class="avatar-preview avatar-empty">
        <text class="avatar-empty-mark">◇</text>
      </view>
      <view class="avatar-upload" @click="pickAvatar">从相册上传</view>
    </view>

    <!-- 一键填入微信昵称与头像（仅小程序端；H5 走「从相册上传 + 手填」） -->
    <!-- #ifdef MP-WEIXIN -->
    <view class="wx-fill-card">
      <text class="wx-fill-title">一键填入微信昵称与头像</text>
      <button class="wx-avatar-btn" open-type="chooseAvatar" @chooseavatar="onPickWxAvatar">
        <text class="wx-avatar-btn-text">填入微信头像</text>
      </button>
      <input
        v-model="wxNickName"
        class="field-input wx-nick-input"
        type="nickname"
        placeholder="点输入框一键填入微信昵称"
        placeholder-style="color: #64748b"
        maxlength="20"
      />
    </view>
    <!-- #endif -->

    <view class="form-card">
      <view class="field">
        <text class="field-label">昵称</text>
        <input
          v-model="form.name"
          class="field-input"
          placeholder="请输入昵称"
          placeholder-style="color: #64748b"
          maxlength="20"
        />
        <!-- 摄影师账号：账号昵称不对外展示，摄影师列表里显示的是独立的艺名。
             不给这句提示，用户改完账号昵称会以为「搜不到自己」是 bug。 -->
        <view v-if="isPhotographerAccount" class="name-hint">
          <text class="name-hint-text">这里是账号昵称，不对外展示。摄影师列表里显示的是你的艺名。</text>
          <text class="name-hint-link" @click="goPhotographerProfileEdit">去改艺名 ›</text>
        </view>
      </view>

      <view class="field">
        <text class="field-label">个人简介</text>
        <textarea
          v-model="form.bio"
          class="field-textarea"
          placeholder="介绍一下自己（选填）"
          placeholder-style="color: #64748b"
          maxlength="200"
        />
      </view>

      <view class="btn-submit" @click="save">保存</view>
    </view>
    </template>
  </view>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { ApiError } from '@/api/client'
import { updateUserProfile, pickAndUploadImages, uploadFile } from '@/api/index'
import { useUserStore } from '@/stores/user'
import { navTo } from '@/utils/nav'
import ListSkeleton from '@/components/ListSkeleton.vue'

/** 首屏深色骨架：数据就绪前置 true，消除切页白屏（B 档） */
const firstLoading = ref(true)

const userStore = useUserStore()

// 摄影师账号才有「对外艺名」这个独立概念：账号昵称不进摄影师列表。
const isPhotographerAccount = computed(() => !!userStore.user?.photographerId)

function goPhotographerProfileEdit() {
  navTo('/pages/photographer/profile-edit')
}

const form = reactive({
  name: '',
  avatar: '',
  bio: ''
})

onShow(() => {
  const user = userStore.user
  if (!user) {
    firstLoading.value = false
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => {
      navTo('/pages/login/index?redirect=/pages/profile/edit')
    }, 800)
    return
  }
  form.name = user.name || ''
  form.avatar = user.avatar || ''
  form.bio = user.bio || ''
  firstLoading.value = false
})

async function pickAvatar() {
  try {
    const urls = await pickAndUploadImages(1)
    if (urls[0]) form.avatar = urls[0]
  } catch (e) {
    // 不再统一写「上传失败」：uploadFile 已经把成因翻译成人话
    // （未登录 / 域名未配置 / 图片格式不支持 / 超时），原样透给用户才有解决方向。
    const msg = e instanceof Error && e.message ? e.message : '上传失败'
    uni.showToast({ title: msg, icon: 'none', duration: 3000 })
  }
}

// ===== 一键填入微信昵称与头像（仅小程序端能力；H5 走「从相册上传 + 手填」） =====
const wxNickName = ref('')

// 微信昵称一键填入 → 同步到昵称输入框
watch(wxNickName, (v) => {
  form.name = v
})

interface AvatarPickEvent {
  detail: { avatarUrl: string }
}

async function onPickWxAvatar(e: AvatarPickEvent) {
  const tempPath = e?.detail?.avatarUrl
  if (!tempPath) return
  try {
    // 复用 /api/v1/upload（与 pickAvatar 同链路）：临时路径先换正式 URL 再进表单
    const url = await uploadFile(tempPath)
    if (url) {
      form.avatar = url
      uni.showToast({ title: '头像已填入', icon: 'none' })
    }
  } catch (e) {
    const msg = e instanceof Error && e.message ? e.message : '头像上传失败，可稍后重试'
    uni.showToast({ title: msg, icon: 'none', duration: 3000 })
  }
}

async function save() {
  if (!form.name.trim()) {
    uni.showToast({ title: '请填写昵称', icon: 'none' })
    return
  }
  uni.showLoading({ title: '保存中...' })
  try {
    await updateUserProfile({ name: form.name.trim(), avatar: form.avatar.trim(), bio: form.bio.trim() })
    const user = userStore.user
    if (user) {
      user.name = form.name.trim()
      user.avatar = form.avatar.trim() || user.avatar
      user.bio = form.bio.trim() || user.bio
      uni.setStorageSync('user', JSON.stringify(userStore.user))
    }
    uni.showToast({ title: '保存成功', icon: 'success' })
    setTimeout(() => uni.navigateBack(), 800)
  } catch (e) {
    const code = e instanceof ApiError ? e.status : 0
    uni.showToast({ title: code === 400 ? '参数有误' : '保存失败', icon: 'none' })
  } finally {
    uni.hideLoading()
  }
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
  padding: $spacing-md;
  box-sizing: border-box;
}

.avatar-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 48rpx 0 40rpx;
}

.avatar-preview {
  width: 180rpx;
  height: 180rpx;
  border-radius: 50%;
  border: 4rpx solid $neon-purple;
  box-shadow: 0 0 24rpx $neon-purple-glow;
}

.avatar-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  background: $dark-bg-secondary;
  border-style: dashed;
  border-color: $dark-border;
  box-shadow: none;
}

.avatar-empty-mark {
  font-size: 56rpx;
  color: $dark-text-tertiary;
}

.avatar-hint {
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  margin-top: $spacing-md;
}

// ===== 一键填入微信昵称与头像（仅小程序端） =====
.wx-fill-card {
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: 24rpx;
  padding: 32rpx;
  margin-bottom: $spacing-md;
  box-shadow: 0 4rpx 24rpx rgba(0, 0, 0, 0.25);
  display: flex;
  flex-direction: column;
  align-items: center;
}

.wx-fill-title {
  font-size: 26rpx;
  color: $dark-text-secondary;
  margin-bottom: 24rpx;
  align-self: flex-start;
}

.wx-avatar-btn {
  width: 320rpx;
  height: 80rpx;
  background: $dark-bg-secondary;
  border: 1rpx solid $neon-purple;
  border-radius: 999rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;

  &::after {
    border: none;
  }
}

.wx-avatar-btn-text {
  font-size: 26rpx;
  color: $neon-purple;
}

.wx-nick-input {
  width: 100%;
  margin-top: 24rpx;
}

.avatar-upload {
  margin-top: $spacing-md;
  padding: 14rpx 40rpx;
  font-size: 26rpx;
  color: $neon-purple;
  border: 1rpx solid $neon-purple;
  border-radius: 999rpx;
  box-shadow: 0 0 12rpx $neon-purple-glow;
}

.form-card {
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: 24rpx;
  padding: 40rpx 32rpx;
  box-shadow: 0 4rpx 24rpx rgba(0, 0, 0, 0.25);
}

.field {
  margin-bottom: 36rpx;
}

.field-label {
  display: block;
  font-size: 26rpx;
  color: $dark-text-secondary;
  margin-bottom: 16rpx;
}

.field-textarea {
  width: 100%;
  height: 180rpx;
  background: $dark-bg-secondary;
  border: 1rpx solid $dark-border;
  border-radius: 16rpx;
  padding: 20rpx 24rpx;
  font-size: 28rpx;
  color: $dark-text-primary;
  box-sizing: border-box;
}

.field-input {
  height: 88rpx;
  background: $dark-bg-secondary;
  border: 1rpx solid $dark-border;
  border-radius: 16rpx;
  padding: 0 24rpx;
  font-size: 28rpx;
  color: $dark-text-primary;
  box-sizing: border-box;
}

.field-hint {
  display: block;
  font-size: 22rpx;
  color: $dark-text-tertiary;
  margin-top: 12rpx;
}

// 摄影师账号昵称 ≠ 对外艺名。用低饱和橙底提示块把两者关系说清，
// 右侧「去改艺名」是跳 profile-edit 的行动点，flex-shrink:0 防长句把它挤没。
.name-hint {
  display: flex;
  align-items: center;
  margin-top: 12rpx;
  padding: 16rpx 20rpx;
  border-radius: 12rpx;
  background: rgba(255, 107, 0, 0.08);
  border: 2rpx solid rgba(255, 107, 0, 0.24);
}

.name-hint-text {
  flex: 1;
  min-width: 0;
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  line-height: 30rpx;
}

.name-hint-link {
  flex-shrink: 0;
  margin-left: 12rpx;
  font-size: $font-size-xs;
  color: #ff8a3d;

  &:active {
    opacity: 0.7;
  }
}

.btn-submit {
  margin-top: 48rpx;
  text-align: center;
  padding: 24rpx 0;
  background: $neon-gradient;
  color: #fff;
  border-radius: 44rpx;
  font-size: 30rpx;
  font-weight: 600;
  box-shadow: 0 0 24rpx $neon-purple-glow;

  &:active {
    transform: scale(0.97);
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}
/* ── 首屏深色骨架：切页不再露白（B 档）── */
.skeleton-page { min-height: 100vh; padding: $spacing-md; box-sizing: border-box; display: flex; flex-direction: column; gap: 24rpx; }
.sk-card { background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 20rpx; box-sizing: border-box; }
.sk-block { background: $surface-2; border-radius: 12rpx; }
.sk-line { height: 28rpx; background: $surface-2; border-radius: 8rpx; }
.sk-circle { background: $surface-2; border-radius: 50%; flex-shrink: 0; }
.sk-row { display: flex; align-items: center; gap: 24rpx; }
.sk-pad { padding: 28rpx; display: flex; flex-direction: column; gap: 24rpx; }
.sk-avatar-120 { width: 120rpx; height: 120rpx; }
.sk-avatar-88 { width: 88rpx; height: 88rpx; }
.sk-hero { height: 160rpx; }
.sk-grid { display: flex; flex-wrap: wrap; gap: 16rpx; }
.sk-grid-item { width: calc(50% - 8rpx); height: 320rpx; border-radius: 16rpx; }
.sk-w-25 { width: 25%; } .sk-w-30 { width: 30%; } .sk-w-40 { width: 40%; }
.sk-w-60 { width: 60%; } .sk-w-80 { width: 80%; } .sk-w-full { width: 100%; }
</style>
