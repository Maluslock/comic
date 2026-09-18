<template>
  <view class="page">
    <view class="header">
      <text class="back" @click="goBack">‹</text>
      <text class="header-title">{{ title }}</text>
    </view>

    <view class="content">
      <view class="doc-card">
        <text v-for="(para, i) in paragraphs" :key="i" class="para">{{ para }}</text>
      </view>
      <text class="version">版本 1.0.0 · © 2024 米拉漫展</text>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'

type DocType = 'privacy' | 'terms' | 'about' | 'help'

const DOCS: Record<DocType, { title: string; paragraphs: string[] }> = {
  privacy: {
    title: '隐私政策',
    paragraphs: [
      '米拉漫展仅在提供约拍服务所必需的范围内收集信息，包括手机号、昵称与头像。',
      '手机号仅用于账号登录与身份识别，不会用于任何营销用途，也不会出售或共享给第三方。',
      '你的预约记录、收藏、关注与聊天内容仅对你本人及交易相关方可见。',
      '你可以随时在「我的 - 设置」中退出登录，或联系客服申请删除账号数据。'
    ]
  },
  terms: {
    title: '用户协议',
    paragraphs: [
      '使用米拉漫展即表示你同意遵守本协议及平台发布的相关规则。',
      '约拍双方应本着诚信原则履行预约：下单后请按约定时间地点赴约，如需变更请尽早沟通。',
      '禁止发布违法违规、侵犯他人肖像权或著作权的作品与言论，违规内容将被下架。',
      '平台仅提供信息撮合服务，实际拍摄服务由摄影师本人提供，请双方自行确认服务细节。'
    ]
  },
  about: {
    title: '关于米拉漫展',
    paragraphs: [
      '米拉漫展是一款面向漫展场景的约拍平台，帮助 coser 与摄影师高效对接。',
      '你可以浏览漫展日历、关注感兴趣的展会、按标签搜索摄影师，并在线完成预约与沟通。',
      '摄影师可自助开通身份、发布作品、管理接单，并申请官方认证获得黄V标识。'
    ]
  },
  help: {
    title: '帮助与反馈',
    paragraphs: [
      '如何预约：进入摄影师主页，选择服务与时段后提交预约，等待摄影师确认即可。',
      '订单状态：待确认 → 待完成 → 已完成；双方均可在订单页查看当前进度。',
      '如何沟通：在订单页或摄影师主页点击「联系摄影师」即可开启聊天。',
      '遇到问题：请通过客服邮箱 support@mira-comic.example 反馈，我们会尽快处理。'
    ]
  }
}

const type = ref<DocType>('about')
const doc = computed(() => DOCS[type.value] || DOCS.about)
const title = computed(() => doc.value.title)
const paragraphs = computed(() => doc.value.paragraphs)

onLoad((options) => {
  const t = options?.type as DocType | undefined
  if (t && t in DOCS) type.value = t
})

function goBack() {
  uni.navigateBack()
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
}

.header {
  display: flex;
  align-items: center;
  height: 88rpx;
  padding: 0 $spacing-md;
  border-bottom: 1rpx solid $dark-border;
}

.back {
  font-size: 44rpx;
  color: $dark-text-primary;
  width: 60rpx;
}

.header-title {
  font-size: $font-size-lg;
  font-weight: 600;
  color: $dark-text-primary;
}

.content {
  padding: $spacing-lg $spacing-md;
}

.doc-card {
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-md;
  padding: $spacing-lg;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.3);
}

.para {
  display: block;
  font-size: $font-size-sm;
  line-height: 1.9;
  color: $dark-text-secondary;
  margin-bottom: $spacing-md;
}

.version {
  display: block;
  margin-top: $spacing-lg;
  text-align: center;
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
}
</style>
