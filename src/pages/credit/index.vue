<template>
  <view class="page">
    <!-- 信用概览 -->
    <view class="score-card">
      <view class="score-head">
        <text class="score-label">我的信用分</text>
        <text class="credit-pill" :class="`tone-${tone}`">{{ levelText }}</text>
      </view>
      <view class="score-value-row">
        <text class="score-value">{{ summary.score }}</text>
        <text class="score-unit">分</text>
      </view>
      <view v-if="frozenText" class="frozen-hint">{{ frozenText }}</view>
      <view class="score-stats">
        <view class="stat-item">
          <text class="stat-num">{{ summary.successCount }}</text>
          <text class="stat-name">成功订单</text>
        </view>
        <view class="stat-item">
          <text class="stat-num">{{ summary.cancelCount }}</text>
          <text class="stat-name">取消预约</text>
        </view>
        <view class="stat-item">
          <text class="stat-num">{{ summary.noshowCount }}</text>
          <text class="stat-name">爽约次数</text>
        </view>
        <view class="stat-item">
          <text class="stat-num">{{ summary.disputeLostCount }}</text>
          <text class="stat-name">争议败诉</text>
        </view>
      </view>
    </view>

    <!-- 规则说明 -->
    <view class="section">
      <view class="section-title"><view class="title-bar" /><text>信用分规则</text></view>
      <view class="rule-list">
        <view class="rule-item"><text class="rule-dot good" /><text class="rule-text">订单正常完成 +2 分（30 日内最多 +4）</text></view>
        <view class="rule-item"><text class="rule-dot bad" /><text class="rule-text">争议裁决败诉 −10 分</text></view>
        <view class="rule-item"><text class="rule-dot bad" /><text class="rule-text">爽约（未到场）−15 分</text></view>
        <view class="rule-item"><text class="rule-dot bad" /><text class="rule-text">24 小时内取消预约 −3 分</text></view>
        <view class="rule-item"><text class="rule-dot bad" /><text class="rule-text">超时未交付 −8 分</text></view>
        <view class="rule-item"><text class="rule-dot warn" /><text class="rule-text">低于 60 分冻结接单 7 天，低于 40 分隐藏排序</text></view>
      </view>
    </view>

    <!-- 变动明细 -->
    <view class="section">
      <view class="section-title">
        <view class="title-bar" />
        <text>变动明细</text>
        <text v-if="total > 0" class="section-extra">共 {{ total }} 条</text>
      </view>

      <ListSkeleton v-if="loading" :rows="3" variant="row" />

      <view v-else-if="events.length === 0" class="empty-state">
        <text class="empty-icon">◇</text>
        <text class="empty-text">暂无信用分变动记录</text>
        <text class="empty-sub">完成订单、按时履约都会记录在这里</text>
      </view>

      <view v-else class="timeline">
        <view v-for="(ev, index) in events" :key="ev.id" class="timeline-node">
          <view class="timeline-rail">
            <view class="timeline-dot" :class="{ plus: ev.plus, minus: !ev.plus }" />
            <view v-if="index < events.length - 1" class="timeline-line" />
          </view>
          <view class="timeline-body">
            <view class="timeline-head">
              <text class="timeline-title">{{ ev.title }}</text>
              <text class="timeline-delta" :class="{ plus: ev.plus, minus: !ev.plus }">{{ ev.deltaText }}</text>
            </view>
            <text class="timeline-time">{{ ev.time }}</text>
            <text v-if="ev.bookingId" class="timeline-link" @click="goBooking(ev.bookingId)">关联订单 #{{ ev.bookingId }} →</text>
          </view>
        </view>
      </view>

      <view v-if="events.length > 0 && events.length < total" class="load-more" @click="loadMore">
        {{ loadingMore ? '加载中…' : '加载更多' }}
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import ListSkeleton from '@/components/ListSkeleton.vue'
import { getMyCredit, getMyCreditEvents } from '@/api'
import type { CreditSummary, CreditEventItem } from '@/types'
import {
  creditLevelText,
  creditTone,
  toCreditEventView,
  frozenHint,
  type CreditEventView
} from '@/utils/credit'

const PAGE_SIZE = 20

const summary = ref<CreditSummary>({
  score: 100,
  level: 'good',
  successCount: 0,
  cancelCount: 0,
  noshowCount: 0,
  disputeLostCount: 0,
  frozenUntil: null,
  updatedAt: null
})

const events = ref<CreditEventView[]>([])
const total = ref(0)
const loading = ref(false)
const loadingMore = ref(false)

const tone = computed(() => creditTone(summary.value.level, summary.value.score))
const levelText = computed(() => creditLevelText(summary.value.level, summary.value.score))
const frozenText = computed(() => frozenHint(summary.value.frozenUntil))

async function loadSummary() {
  try {
    summary.value = await getMyCredit()
  } catch {
    // 保留默认 100 分，页面仍可展示
  }
}

async function loadEvents(reset: boolean) {
  if (reset) {
    loading.value = true
  } else {
    loadingMore.value = true
  }
  try {
    const res = await getMyCreditEvents({
      limit: PAGE_SIZE,
      offset: reset ? 0 : events.value.length
    })
    const list: CreditEventItem[] = res?.list || []
    const views = list.map(toCreditEventView)
    events.value = reset ? views : events.value.concat(views)
    total.value = typeof res?.total === 'number' ? res.total : events.value.length
  } catch {
    if (reset) {
      events.value = []
      total.value = 0
    }
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function loadMore() {
  if (loadingMore.value || events.value.length >= total.value) return
  loadEvents(false)
}

function goBooking(bookingId: number) {
  uni.navigateTo({ url: `/pages/order/detail?id=${bookingId}` })
}

onLoad(() => {
  loadSummary()
  loadEvents(true)
})

onShow(() => {
  // 从订单详情返回时刷新（分数可能已变化）
  loadSummary()
})
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
  padding: 24rpx 24rpx 60rpx;
}

.score-card {
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: 24rpx;
  padding: 32rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.2);
}

.score-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.score-label {
  font-size: $font-size-sm;
  color: $dark-text-secondary;
}

.credit-pill {
  font-size: 22rpx;
  padding: 4rpx 14rpx;
  border-radius: 999rpx;
  border: 1rpx solid $dark-border;
  background: rgba(255, 255, 255, 0.04);

  &.tone-good {
    color: $success-color;
    border-color: rgba(34, 197, 94, 0.4);
  }

  &.tone-fair {
    color: $warning-color;
    border-color: rgba(250, 204, 21, 0.4);
  }

  &.tone-poor {
    color: $error-color;
    border-color: rgba(239, 68, 68, 0.4);
  }
}

.score-value-row {
  display: flex;
  align-items: baseline;
  margin-top: $spacing-sm;
}

.score-value {
  font-size: 72rpx;
  font-weight: 700;
  color: $dark-text-primary;
  line-height: 1.1;
}

.score-unit {
  font-size: $font-size-sm;
  color: $dark-text-tertiary;
  margin-left: 8rpx;
}

.frozen-hint {
  margin-top: $spacing-xs;
  font-size: $font-size-xs;
  color: $warning-color;
}

.score-stats {
  display: flex;
  margin-top: $spacing-md;
  padding-top: $spacing-md;
  border-top: 1rpx solid $dark-border;
}

.stat-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.stat-num {
  font-size: $font-size-lg;
  font-weight: 600;
  color: $dark-text-primary;
}

.stat-name {
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  margin-top: 4rpx;
}

.section {
  margin-top: $spacing-lg;
}

.section-title {
  display: flex;
  align-items: center;
  margin-bottom: $spacing-sm;
}

.title-bar {
  width: 6rpx;
  height: 28rpx;
  border-radius: 3rpx;
  background: $neon-gradient;
  margin-right: 12rpx;
}

.section-title text {
  font-size: $font-size-md;
  font-weight: 600;
  color: $dark-text-primary;
}

.section-extra {
  margin-left: auto;
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
}

.rule-list {
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: 20rpx;
  padding: 24rpx;
}

.rule-item {
  display: flex;
  align-items: flex-start;
  margin-bottom: 14rpx;

  &:last-child {
    margin-bottom: 0;
  }
}

.rule-dot {
  width: 10rpx;
  height: 10rpx;
  border-radius: 50%;
  margin-top: 10rpx;
  margin-right: 14rpx;
  flex-shrink: 0;

  &.good {
    background: $success-color;
  }

  &.bad {
    background: $error-color;
  }

  &.warn {
    background: $warning-color;
  }
}

.rule-text {
  font-size: $font-size-xs;
  color: $dark-text-secondary;
  line-height: 1.6;
}

.loading-text {
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  text-align: center;
  padding: 40rpx 0;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 60rpx 0;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: 20rpx;
}

.empty-icon {
  font-size: 64rpx;
  color: $dark-text-tertiary;
}

.empty-text {
  font-size: $font-size-sm;
  color: $dark-text-secondary;
  margin-top: $spacing-sm;
}

.empty-sub {
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  margin-top: 8rpx;
}

.timeline {
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: 20rpx;
  padding: 24rpx;
}

.timeline-node {
  display: flex;
}

.timeline-rail {
  width: 28rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  flex-shrink: 0;
}

.timeline-dot {
  width: 16rpx;
  height: 16rpx;
  border-radius: 50%;
  margin-top: 8rpx;

  &.plus {
    background: $success-color;
  }

  &.minus {
    background: $error-color;
  }
}

.timeline-line {
  flex: 1;
  width: 2rpx;
  background: $dark-border;
  margin: 6rpx 0;
}

.timeline-body {
  flex: 1;
  padding-bottom: 24rpx;
  padding-left: 16rpx;
}

.timeline-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.timeline-title {
  font-size: $font-size-sm;
  color: $dark-text-primary;
}

.timeline-delta {
  font-size: $font-size-sm;
  font-weight: 600;

  &.plus {
    color: $success-color;
  }

  &.minus {
    color: $error-color;
  }
}

.timeline-time {
  display: block;
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  margin-top: 4rpx;
}

.timeline-link {
  display: block;
  font-size: $font-size-xs;
  color: $neon-cyan;
  margin-top: 6rpx;
}

.load-more {
  text-align: center;
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  padding: 24rpx 0;
}
</style>
