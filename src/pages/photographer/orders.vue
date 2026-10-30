<template>
  <view class="page" :style="{ paddingTop: (statusBarHeight + 44) + 'px' }">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />
    <view class="nav-bar" :style="{ paddingTop: statusBarHeight + 'px' }">
      <text class="back-arrow" @click="goBack">‹</text>
      <text class="nav-title">接单管理</text>
    </view>

    <view v-if="!notifyOn" class="notify-tip">
      <text class="notify-tip-text">开启接单提醒，客户下单时微信通知你</text>
      <view class="notify-tip-btn" @click="onEnableNotify">开启</view>
    </view>

    <view class="body">
      <view v-if="!userStore.user?.photographerId" class="empty-state">
        <text class="empty-icon">◈</text>
        <text class="empty-text">请先开通摄影师身份</text>
        <view class="btn-primary" @click="goActivate">去开通</view>
      </view>
      <view v-else-if="loading" class="empty-hint">加载中...</view>
      <view v-else-if="orders.length === 0" class="empty-state">
        <text class="empty-icon">◇</text>
        <text class="empty-text">暂无收到预约</text>
      </view>
      <view v-else class="order-list">
        <view v-for="order in orders" :key="order.id" class="order-card">
          <view class="order-header">
            <text class="order-date">{{ order.date }} {{ order.time }}</text>
            <view class="order-tags">
              <text v-if="subStatusText(order)" class="sub-status" :class="`sub-${subTone(order)}`">
                {{ subStatusText(order) }}
              </text>
              <text class="order-status" :class="`status-${order.status}`">{{ getStatusText(order.status) }}</text>
            </view>
          </view>

          <view class="order-content">
            <image :src="order.coserAvatar || ''" class="coser-avatar" mode="aspectFill" />
            <view class="order-info">
              <text class="coser-name">{{ order.coserName || `Coser${order.coserId}` }}</text>
              <text class="coser-phone">{{ renderCoserPhone(order) }}</text>
              <text class="service-name">{{ order.serviceName || '服务' }} · {{ renderOrderPrice(order) }}</text>
            </view>
          </view>

          <view v-if="order.remarks" class="order-remark">备注：{{ order.remarks }}</view>

          <view class="order-footer">
            <view v-if="order.status === 'pending'" class="btn-outline danger" @click="rejectOrder(order.id)">拒绝</view>
            <view v-if="order.status === 'pending' && !isNegotiablePending(order)" class="btn-primary" @click="updateStatus(order.id, 'confirmed')">确认接单</view>
            <view v-if="canQuote(order)" class="btn-primary" @click="quoteOrder(order.id)">报价</view>
            <view v-if="canCheckin(order)" class="btn-primary" @click="doCheckin(order)">核销</view>
            <view v-if="canDeliver(order)" class="btn-primary" @click="doDeliver(order)">提交交付</view>
            <view v-if="canMarkNoShow(order)" class="btn-outline danger" @click="doMarkNoShow(order)">标记爽约</view>
            <view v-if="order.status === 'confirmed' && !order.subStatus" class="btn-primary" @click="updateStatus(order.id, 'completed')">完成拍摄</view>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { TMPL, requestSubscribe, markSubscribed, isSubscribed } from '@/utils/subscribe'
import { apiGet, apiPut, ApiError } from '@/api/client'
import { useUserStore } from '@/stores/user'
import type { OrderPriceFields, EscrowBookingFields } from '@/types'
import { renderOrderPrice, toPriceMode, toPriceStatus, promptQuote } from '@/utils/quote'
import { escrowState } from '@/utils/escrow'
import { checkinBooking, deliverBooking, markNoShow, pickAndUploadImages } from '@/api'

interface PhotographerOrder extends OrderPriceFields, EscrowBookingFields {
  id: number
  photographerId: number
  coserId: number
  serviceId: number
  date: string
  time: string
  status: string
  remarks?: string
  serviceName?: string
  coserName?: string
  coserAvatar?: string
  coserPhone?: string
}

const statusBarHeight = ref(44)
const loading = ref(true)
const orders = ref<PhotographerOrder[]>([])
const userStore = useUserStore()

const notifyOn = ref(isSubscribed(TMPL.newBooking))

async function onEnableNotify() {
  if (notifyOn.value) return
  const res = await requestSubscribe([TMPL.newBooking])
  if (res?.[TMPL.newBooking] === 'accept') {
    markSubscribed(TMPL.newBooking)
    notifyOn.value = true
    uni.showToast({ title: '已开启', icon: 'success' })
  } else {
    uni.showToast({ title: '未开启，可稍后再试', icon: 'none' })
  }
}

onShow(() => {
  const sysInfo = uni.getSystemInfoSync()
  statusBarHeight.value = sysInfo.statusBarHeight || 44
  loadOrders()
})

async function loadOrders() {
  if (!userStore.user?.photographerId) {
    orders.value = []
    loading.value = false
    return
  }
  try {
    const res = await apiGet<PhotographerOrder[]>(`/v1/bookings/photographer/${userStore.user.photographerId}`)
    orders.value = (res || []).map((o) => ({
      ...o,
      priceMode: toPriceMode(o.priceMode),
      priceStatus: toPriceStatus(o.priceStatus),
      quotePrice: o.quotePrice ?? null,
      subStatus: o.subStatus ?? null,
      checkinAt: o.checkinAt ?? null,
      deliveredAt: o.deliveredAt ?? null,
      acceptDeadline: o.acceptDeadline ?? null,
      deliveryEvidence: Array.isArray(o.deliveryEvidence) ? o.deliveryEvidence : [],
      noShowParty: o.noShowParty ?? null,
      cancelledBy: o.cancelledBy ?? null
    }))
  } catch {
    orders.value = []
  } finally {
    loading.value = false
  }
}

function isNegotiablePending(order: PhotographerOrder): boolean {
  return order.priceMode === 'negotiable' && (order.priceStatus === 'awaiting_quote' || order.priceStatus === 'quoted')
}

// 手机号是个人信息：摄影师在确认接单前没有联系对方的正当必要，后端据此在 SQL 层就遮罩了
// （confirmed/completed 才返回），这里只负责把「为什么看不到」讲清楚。接单后要赴约，必须能联系上。
const PHONE_VISIBLE_STATUSES = ['confirmed', 'completed']

function renderCoserPhone(order: PhotographerOrder): string {
  if (!PHONE_VISIBLE_STATUSES.includes(order.status)) {
    return '电话：确认接单后可见'
  }
  return `电话：${order.coserPhone || '未留电话'}`
}

function canQuote(order: PhotographerOrder): boolean {
  return order.priceMode === 'negotiable' && order.priceStatus === 'awaiting_quote'
}

/* ───────────────── 担保交易：核销 / 交付 / 爽约 ───────────────── */

/**
 * 可核销：已确认接单、尚未核销、且订单没有进入争议/交付。
 * 注意：后端确认接单走的是 UpdateBookingStatus（只改 status），
 * 所以「待核销」时 sub_status 可能还是空，也可能已经是 in_service，
 * 两种都要允许核销，否则按钮永远不出现。
 */
function canCheckin(order: PhotographerOrder): boolean {
  if (order.status !== 'confirmed' || order.checkinAt) return false
  return order.subStatus !== 'delivered' && order.subStatus !== 'disputed'
}

/** 可交付：已核销、尚未交付、且没有争议。 */
function canDeliver(order: PhotographerOrder): boolean {
  if (order.status !== 'confirmed' || !order.checkinAt) return false
  return order.subStatus !== 'delivered' && order.subStatus !== 'disputed'
}

/** 可标记爽约：已确认接单、尚未核销（coser 没到场），且还没标记过。 */
function canMarkNoShow(order: PhotographerOrder): boolean {
  return order.status === 'confirmed' && !order.checkinAt && !order.noShowParty && order.subStatus !== 'disputed'
}

/** 核销：输入 coser 的 6 位核销码。 */
function doCheckin(order: PhotographerOrder) {
  uni.showModal({
    title: '拍摄核销',
    editable: true,
    placeholderText: '请输入 coser 的 6 位核销码',
    success: async (res) => {
      if (!res.confirm) return
      const code = (res.content || '').trim()
      if (!/^[0-9a-zA-Z]{6}$/.test(code)) {
        uni.showToast({ title: '核销码为 6 位', icon: 'none' })
        return
      }
      uni.showLoading({ title: '核销中...' })
      try {
        await checkinBooking(order.id, code)
        uni.hideLoading()
        uni.showToast({ title: '核销成功', icon: 'success' })
        loadOrders()
      } catch (e) {
        uni.hideLoading()
        const code2 = e instanceof ApiError ? e.status : 0
        uni.showToast({
          title: code2 === 403 ? '核销码不正确' : code2 === 409 ? '当前状态不能核销' : '核销失败',
          icon: 'none'
        })
      }
    }
  })
}

/** 提交交付：上传成片凭证（图片）。 */
function doDeliver(order: PhotographerOrder) {
  uni.showLoading({ title: '提交中...' })
  pickAndUploadImages(3)
    .then(async (urls) => {
      if (!urls.length) {
        uni.hideLoading()
        return
      }
      await deliverBooking(order.id, urls)
      uni.hideLoading()
      uni.showToast({ title: '已提交交付', icon: 'success' })
      loadOrders()
    })
    .catch(() => {
      uni.hideLoading()
      uni.showToast({ title: '提交失败', icon: 'none' })
    })
}

/** 标记 coser 爽约。 */
function doMarkNoShow(order: PhotographerOrder) {
  uni.showModal({
    title: '标记爽约',
    content: '确认 coser 未按时到场？标记后对方信用分会被扣减。',
    success: async (res) => {
      if (!res.confirm) return
      uni.showLoading({ title: '提交中...' })
      try {
        await markNoShow(order.id, 'coser')
        uni.hideLoading()
        uni.showToast({ title: '已标记爽约', icon: 'success' })
        loadOrders()
      } catch (e) {
        uni.hideLoading()
        const code = e instanceof ApiError ? e.status : 0
        uni.showToast({
          title: code === 409 ? '当前状态不能标记' : code === 403 ? '无权操作该订单' : '标记失败',
          icon: 'none'
        })
      }
    }
  })
}

function quoteOrder(id: number) {
  promptQuote(id, loadOrders)
}

async function updateStatus(id: number, status: string) {
  uni.showLoading({ title: '提交中...' })
  try {
    await apiPut(`/v1/bookings/${id}/status`, { status, actorTag: 'photographer' })
    uni.hideLoading()
    uni.showToast({ title: '操作成功', icon: 'success' })
    loadOrders()
  } catch (e) {
    uni.hideLoading()
    const code = e instanceof ApiError ? e.status : 0
    uni.showToast({ title: code === 409 ? '操作不允许' : '操作失败', icon: 'none' })
  }
}

function rejectOrder(id: number) {
  uni.showModal({
    title: '拒绝预约',
    content: '确定要拒绝这个预约吗？',
    success: (res) => {
      if (res.confirm) updateStatus(id, 'cancelled')
    }
  })
}

function getStatusText(status: string): string {
  const map: Record<string, string> = {
    pending: '待确认',
    confirmed: '待完成',
    completed: '已完成',
    cancelled: '已取消'
  }
  return map[status] || status
}

/** 子状态小标签文案（待核销 / 拍摄中 / 待确认交付 / 争议中）。 */
function subStatusText(order: PhotographerOrder): string {
  return escrowState(order.status, order.subStatus, order.checkinAt).subText
}

/** 子状态小标签色调。 */
function subTone(order: PhotographerOrder): string {
  return escrowState(order.status, order.subStatus, order.checkinAt).tone
}

function goActivate() {
  uni.navigateTo({ url: '/pages/photographer/activate' })
}

function goBack() {
  uni.navigateBack()
}
</script>

<style lang="scss" scoped>
.notify-tip {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  margin: $spacing-md $spacing-md 0;
  padding: $spacing-sm $spacing-md;
  background: rgba(255, 107, 0, 0.08);
  border: 1rpx solid rgba(255, 107, 0, 0.24);
  border-radius: $border-radius-lg;
}

.notify-tip-text {
  flex: 1;
  font-size: $font-size-sm;
  color: $text-secondary;
}

.notify-tip-btn {
  padding: 8rpx 24rpx;
  font-size: $font-size-sm;
  color: #0c0d14;
  background: $brand;
  border-radius: 999rpx;
}

.page { min-height: 100vh; background: $dark-bg-primary; }
.status-bar { position: fixed; top: 0; left: 0; right: 0; z-index: 100; }
.nav-bar { position: fixed; top: 0; left: 0; right: 0; z-index: 99; height: 88rpx; display: flex; align-items: center; background: $dark-bg-primary; border-bottom: 1rpx solid $dark-border; }
.back-arrow { font-size: 48rpx; color: $dark-text-primary; padding: 0 24rpx; }
.nav-title { font-size: 34rpx; font-weight: 600; color: $dark-text-primary; }
.body { padding: 24rpx 32rpx 48rpx; }

.empty-state { display: flex; flex-direction: column; align-items: center; padding-top: 200rpx; }
.empty-icon { font-size: 100rpx; color: $neon-cyan; text-shadow: 0 0 24rpx $neon-cyan-glow; }
.empty-text { font-size: 28rpx; color: $dark-text-secondary; margin: 24rpx 0 32rpx; }
.empty-hint { text-align: center; padding-top: 200rpx; color: $dark-text-tertiary; }

.order-list { display: flex; flex-direction: column; gap: 24rpx; }
.order-card { background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 20rpx; padding: 28rpx; box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.2); }
.order-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20rpx; }
.order-date { font-size: 26rpx; color: $dark-text-secondary; }
.order-tags { display: flex; align-items: center; gap: 12rpx; }
.sub-status { font-size: 22rpx; padding: 4rpx 14rpx; border-radius: 999rpx; border: 1rpx solid $dark-border; background: rgba(255, 255, 255, 0.04);
  &.sub-cyan { color: $neon-cyan; border-color: rgba(6, 182, 212, 0.45); }
  &.sub-purple { color: $neon-purple-bright; border-color: rgba(255, 107, 0, 0.45); }
  &.sub-warning { color: $warning-color; border-color: rgba(250, 204, 21, 0.45); }
  &.sub-error { color: $error-color; border-color: rgba(244, 63, 94, 0.45); }
}
.order-status { font-size: 24rpx; font-weight: 500;
  &.status-pending { color: $neon-cyan; }
  &.status-confirmed { color: $neon-purple; }
  &.status-completed { color: $success-color; }
  &.status-cancelled { color: $dark-text-tertiary; }
}
.order-content { display: flex; align-items: center; }
.coser-avatar { width: 100rpx; height: 100rpx; border-radius: 50%; border: 2rpx solid $neon-purple-glow; background: $dark-bg-secondary; flex-shrink: 0; }
.order-info { flex: 1; margin-left: 20rpx; overflow: hidden; }
.coser-name { display: block; font-size: 30rpx; font-weight: 600; color: $dark-text-primary; }
.coser-phone { display: block; font-size: 24rpx; color: $dark-text-secondary; margin-top: 6rpx; }
.service-name { display: block; font-size: 24rpx; color: $neon-cyan; margin-top: 6rpx; }
.order-remark { font-size: 22rpx; color: $dark-text-tertiary; margin-top: 16rpx; padding-top: 16rpx; border-top: 1rpx solid $dark-border; }
.order-footer { display: flex; justify-content: flex-end; gap: 20rpx; margin-top: 24rpx; }

.btn-outline { padding: 14rpx 36rpx; border: 2rpx solid $dark-border; border-radius: 32rpx; font-size: 26rpx; color: $dark-text-secondary;
  &.danger { color: $error-color; border-color: rgba(239, 68, 68, 0.4); }
  &:active { transform: scale(0.97); background: $dark-bg-card-hover; }
}
.btn-primary { padding: 14rpx 36rpx; background: $neon-purple; color: #fff; border-radius: 32rpx; font-size: 26rpx; box-shadow: 0 0 16rpx $neon-purple-glow;
  &:active { transform: scale(0.97); box-shadow: 0 0 12rpx $neon-purple-glow; }
}
</style>
