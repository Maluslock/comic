<template>
  <view class="page">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }"></view>
    
    <view class="header">
      <view class="back-btn" @click="goBack">
        <text class="back-icon">‹</text>
      </view>
      <text class="header-title">订单详情</text>
      <view class="placeholder"></view>
    </view>

    <scroll-view scroll-y class="content">
      <view class="status-card" :class="order.status">
        <text class="status-icon">◆</text>
        <text class="status-text">{{ getStatusText(order.status) }}</text>
        <text v-if="subState.subText" class="status-sub" :class="`sub-${subState.tone}`">
          {{ subState.subText }}
        </text>
        <text class="order-id">订单号：{{ order.id }}</text>
      </view>

      <view v-if="showEscrowInfo" class="section">
        <view class="section-title">担保进度</view>
        <view class="service-detail">
          <view v-if="showCheckinCode" class="detail-row">
            <text class="detail-label">核销码</text>
            <text class="detail-value code">{{ order.checkinCode }}</text>
          </view>
          <view v-if="order.checkinAt" class="detail-row">
            <text class="detail-label">核销时间</text>
            <text class="detail-value">{{ formatDateTime(order.checkinAt) }}</text>
          </view>
          <view v-if="order.deliveredAt" class="detail-row">
            <text class="detail-label">交付时间</text>
            <text class="detail-value">{{ formatDateTime(order.deliveredAt) }}</text>
          </view>
          <view v-if="showAcceptDeadline" class="detail-row">
            <text class="detail-label">确认交付截止</text>
            <text class="detail-value">{{ formatDateTime(order.acceptDeadline) }}</text>
          </view>
          <view v-if="showAcceptDeadline" class="detail-row">
            <text class="detail-label">剩余时间</text>
            <text class="detail-value warn">{{ countdownText(order.acceptDeadline) }}</text>
          </view>
          <view v-if="order.noShowParty" class="detail-row">
            <text class="detail-label">爽约方</text>
            <text class="detail-value error">{{ noShowText }}</text>
          </view>
          <view v-if="order.cancelledBy" class="detail-row">
            <text class="detail-label">取消方</text>
            <text class="detail-value">{{ cancelledByText }}</text>
          </view>
        </view>
        <view v-if="deliveryEvidence.length" class="evidence">
          <text class="evidence-title">交付凭证</text>
          <view class="evidence-list">
            <image
              v-for="(img, idx) in deliveryEvidence"
              :key="idx"
              :src="img"
              class="evidence-img"
              mode="aspectFill"
              @click="previewEvidence(idx)"
            />
          </view>
        </view>
      </view>

      <view class="section">
        <view class="section-title">履约时间线</view>
        <view class="timeline">
          <view v-for="(node, idx) in timeline" :key="node.key" class="timeline-node">
            <view class="timeline-rail">
              <view class="timeline-dot" :class="{ done: node.done }"></view>
              <view v-if="idx < timeline.length - 1" class="timeline-line"></view>
            </view>
            <view class="timeline-body">
              <view class="timeline-head">
                <text class="timeline-title" :class="{ done: node.done }">{{ node.title }}</text>
                <text v-if="node.time" class="timeline-time">{{ node.time }}</text>
              </view>
              <text class="timeline-desc">{{ node.desc }}</text>
              <text v-if="node.actor" class="timeline-actor">{{ node.actor }}</text>
            </view>
          </view>
        </view>
      </view>

      <view v-if="dispute" class="section dispute-section">
        <view class="section-title">争议进展</view>
        <view class="dispute-head">
          <text class="dispute-type">{{ disputeTypeLabel }}</text>
          <text class="dispute-status" :class="`d-${dispute.status}`">{{ disputeStatusLabel }}</text>
        </view>
        <view class="dispute-reason">{{ dispute.reason }}</view>
        <view class="service-detail">
          <view class="detail-row">
            <text class="detail-label">申请方</text>
            <text class="detail-value">{{ dispute.openedByRole === 'coser' ? 'coser' : '摄影师' }}</text>
          </view>
          <view class="detail-row">
            <text class="detail-label">响应截止</text>
            <text class="detail-value">{{ formatDateTime(dispute.respondDeadline) || '—' }}</text>
          </view>
          <view v-if="dispute.status === 'open' || dispute.status === 'responding'" class="detail-row">
            <text class="detail-label">剩余时间</text>
            <text class="detail-value warn">{{ respondCountdown }}</text>
          </view>
          <view class="detail-row">
            <text class="detail-label">举证截止</text>
            <text class="detail-value">{{ formatDateTime(dispute.evidenceDeadline) || '—' }}</text>
          </view>
          <view v-if="dispute.resolution" class="detail-row">
            <text class="detail-label">裁决结果</text>
            <text class="detail-value">{{ dispute.resolution }}</text>
          </view>
        </view>
        <view v-if="dispute.status === 'open' || dispute.status === 'responding'" class="dispute-actions">
          <view v-if="canRespondDispute" class="btn-outline" @click="onRespondDispute">提交说明</view>
          <view v-if="canEscalateDispute" class="btn-outline danger" @click="onEscalateDispute">升级裁决</view>
        </view>
      </view>

      <view class="section">
        <view class="section-title">摄影师信息</view>
        <view class="photographer-info">
          <image :src="order.photographerAvatar" class="avatar" mode="aspectFill" />
          <view class="info">
            <text class="name">{{ order.photographerName }}</text>
            <text class="location">{{ order.location }}</text>
          </view>
          <view class="btn-outline" @click="goChat">联系</view>
        </view>
      </view>

      <view class="section">
        <view class="section-title">服务详情</view>
        <view class="service-detail">
          <view class="detail-row">
            <text class="detail-label">服务套餐</text>
            <text class="detail-value">{{ order.serviceName }}</text>
          </view>
          <view class="detail-row">
            <text class="detail-label">拍摄时长</text>
            <text class="detail-value">{{ formatDuration(order.duration) }}</text>
          </view>
          <view class="detail-row">
            <text class="detail-label">拍摄日期</text>
            <text class="detail-value">{{ order.date }}</text>
          </view>
          <view class="detail-row">
            <text class="detail-label">拍摄时间</text>
            <text class="detail-value">{{ order.time }}</text>
          </view>
          <view class="detail-row">
            <text class="detail-label">备注信息</text>
            <text class="detail-value">{{ order.remark || '无' }}</text>
          </view>
        </view>
      </view>

      <view class="section">
        <view class="section-title">费用明细</view>
        <view class="price-detail">
          <view class="price-row">
            <text class="price-label">{{ order.serviceName }}</text>
            <text class="price-value">{{ renderOrderPrice(order) }}</text>
          </view>
          <view class="price-row total">
            <text class="price-label">总计</text>
            <text class="price-value">{{ renderOrderPrice(order) }}</text>
          </view>
        </view>
      </view>

      <view class="section">
        <view class="section-title">下单时间</view>
        <text class="order-time">{{ order.createTime }}</text>
      </view>

      <view class="bottom-space"></view>
    </scroll-view>

    <view class="footer">
      <view 
        v-if="showRespondQuote" 
        class="btn-outline danger"
        @click="rejectQuote"
      >
        拒绝
      </view>
      <view 
        v-if="showRespondQuote" 
        class="btn-primary"
        @click="acceptQuote"
      >
        接受报价
      </view>
      <view 
        v-if="showQuote" 
        class="btn-primary"
        @click="quoteOrder"
      >
        报价
      </view>
      <view 
        v-if="(order.status === 'pending' || order.status === 'confirmed') && !showRespondQuote" 
        class="btn-outline"
        @click="cancelOrder"
      >
        取消预约
      </view>
      <view 
        v-if="order.status === 'confirmed' && order.subStatus === 'delivered'" 
        class="btn-primary"
        @click="acceptDelivery"
      >
        确认完成
      </view>
      <view 
        v-else-if="order.status === 'confirmed' && !order.subStatus" 
        class="btn-primary"
        @click="confirmOrder"
      >
        确认完成
      </view>
      <view 
        v-if="canOpenDispute" 
        class="btn-outline danger"
        @click="onOpenDispute"
      >
        发起争议
      </view>
      <view 
        v-if="order.status === 'completed'" 
        class="btn-primary"
        @click="goReview"
      >
        去评价
      </view>
      <view 
        v-if="order.status === 'pending' || order.status === 'confirmed'" 
        class="btn-primary"
        @click="goChat"
      >
        联系摄影师
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { apiGet, apiPost, apiPut, ApiError } from '@/api/client'
import { useUserStore } from '@/stores/user'
import type {
  OrderPriceFields,
  EscrowBookingFields,
  BookingSubStatus,
  OrderEventItem,
  DisputeItem,
  DisputeStatus,
  DisputeType
} from '@/types'
import { renderOrderPrice, toPriceMode, toPriceStatus, promptQuote, confirmRespondQuote } from '@/utils/quote'
import { formatDuration } from '@/utils/duration'
import { resolveImage } from '@/utils/mappers'
import {
  escrowState,
  buildTimeline,
  formatDateTime,
  countdownText,
  disputeStatusText,
  disputeTypeText
} from '@/utils/escrow'
import {
  acceptDelivery as acceptDeliveryApi,
  getOrderEvents,
  getBookingDispute,
  openDispute as openDisputeApi,
  respondDispute as respondDisputeApi,
  escalateDispute as escalateDisputeApi
} from '@/api'

interface OrderDetail extends OrderPriceFields, EscrowBookingFields {
  id: string
  coserId?: string
  photographerId?: string
  photographerUserId?: number
  photographerName: string
  photographerAvatar: string
  location: string
  serviceId?: string
  serviceName: string
  duration: number | null
  date: string
  time: string
  status: string
  remark: string
  createTime: string | number
}

const statusBarHeight = ref(44)

const defaultOrder: OrderDetail = {
  id: '',
  photographerName: '未知摄影师',
  photographerAvatar: '',
  location: '',
  serviceName: '未知服务',
  duration: null,
  date: '',
  time: '',
  remark: '',
  totalPrice: 0,
  priceMode: 'fixed',
  quotePrice: null,
  priceStatus: 'agreed',
  status: 'pending',
  createTime: '',
  subStatus: null,
  checkinCode: null,
  checkinAt: null,
  deliveredAt: null,
  acceptDeadline: null,
  deliveryEvidence: [],
  noShowParty: null,
  cancelledBy: null
}

const order = ref<OrderDetail>({ ...defaultOrder })
const userStore = useUserStore()

const bookingId = ref('')

/** 履约事件（时间线数据源）。 */
const events = ref<OrderEventItem[]>([])
/** 当前未关闭争议（无则 null）。 */
const dispute = ref<DisputeItem | null>(null)
/** 我在该订单中的角色（服务端判定）。 */
const myRole = ref<'coser' | 'photographer' | ''>('')
/** 倒计时刷新用的 tick（每 60 秒触发一次重算）。 */
const clock = ref(Date.now())

onLoad((options: Record<string, string> | undefined) => {
  bookingId.value = options?.id || ''
  if (!bookingId.value) {
    uni.showToast({ title: '订单不存在', icon: 'none' })
    return
  }
  loadOrder()
})

async function loadOrder() {
  let found: OrderDetail | undefined

  if (userStore.isLoggedIn && userStore.user) {
    try {
      const res = await apiGet<any[]>(`/v1/bookings/${userStore.user.id}`)
      const serverOrder = (res || []).find((b: any) => String(b.id) === bookingId.value)
      if (serverOrder) {
        found = {
          id: String(serverOrder.id),
          coserId: serverOrder.coserId === undefined || serverOrder.coserId === null ? '' : String(serverOrder.coserId),
          photographerId: String(serverOrder.photographerId),
          photographerUserId: serverOrder.photographerUserId,
          photographerName: serverOrder.photographerName || `摄影师${serverOrder.photographerId}`,
          photographerAvatar: serverOrder.photographerAvatar || '',
          location: serverOrder.location || '',
          serviceId: String(serverOrder.serviceId),
          serviceName: serverOrder.serviceName || `服务${serverOrder.serviceId}`,
          duration: serverOrder.serviceDuration ?? null,
          date: serverOrder.date,
          time: serverOrder.time,
          totalPrice: serverOrder.totalPrice || 0,
          priceMode: toPriceMode(serverOrder.priceMode),
          quotePrice: serverOrder.quotePrice ?? null,
          priceStatus: toPriceStatus(serverOrder.priceStatus),
          status: serverOrder.status || 'pending',
          remark: serverOrder.remarks || '',
          createTime: serverOrder.createdAt || Date.now(),
          subStatus: (serverOrder.subStatus || null) as BookingSubStatus | null,
          checkinCode: serverOrder.checkinCode ?? null,
          checkinAt: serverOrder.checkinAt ?? null,
          deliveredAt: serverOrder.deliveredAt ?? null,
          acceptDeadline: serverOrder.acceptDeadline ?? null,
          deliveryEvidence: Array.isArray(serverOrder.deliveryEvidence) ? serverOrder.deliveryEvidence : [],
          noShowParty: serverOrder.noShowParty ?? null,
          cancelledBy: serverOrder.cancelledBy ?? null
        }
      }
    } catch {
      // Ignore API errors, fall through to storage
    }
  }

  if (!found) {
    try {
      const storedJson = uni.getStorageSync('bookings') || '[]'
      const storedBookings: OrderDetail[] = JSON.parse(storedJson)
      found = storedBookings.find((b: OrderDetail) => String(b.id) === bookingId.value)
    } catch {
      // Ignore parse errors
    }
  }

  if (found) {
    order.value = found
    await loadEscrowInfo()
  } else {
    uni.showToast({ title: '订单不存在', icon: 'none' })
  }
}

/** 加载履约事件与争议（失败静默降级，时间线仍能显示主链路）。 */
async function loadEscrowInfo() {
  if (!bookingId.value) return
  try {
    const res = await getOrderEvents(bookingId.value)
    events.value = res?.list || []
  } catch {
    events.value = []
  }
  try {
    const res = await getBookingDispute(bookingId.value)
    dispute.value = res?.dispute || null
    myRole.value = res?.myRole || ''
  } catch {
    dispute.value = null
  }
}

onMounted(() => {
  const sysInfo = uni.getSystemInfoSync()
  statusBarHeight.value = sysInfo.statusBarHeight || 44
})

function getStatusText(status: string): string {
  const map: Record<string, string> = {
    pending: '待确认',
    confirmed: '待完成',
    completed: '已完成',
    cancelled: '已取消'
  }
  return map[status] || status
}

/* ───────────────── 担保交易：子状态 / 时间线 / 争议 ───────────────── */

/** 主状态之外的履约子状态（待核销 / 拍摄中 / 待确认交付 / 争议中）。 */
const subState = computed(() =>
  escrowState(order.value.status, order.value.subStatus, order.value.checkinAt)
)

/** 履约时间线。 */
const timeline = computed(() =>
  buildTimeline(events.value, myRole.value, {
    status: order.value.status,
    createdAt: order.value.createTime
  })
)

/** 交付凭证（摄影师上传的成片图）。 */
const deliveryEvidence = computed<string[]>(() =>
  (order.value.deliveryEvidence || []).map(item => resolveImage(item))
)

/** 是否显示担保进度区块。 */
const showEscrowInfo = computed(() => {
  const o = order.value
  if (o.status !== 'confirmed') return false
  return !!(
    o.subStatus ||
    o.checkinCode ||
    o.checkinAt ||
    o.deliveredAt ||
    o.acceptDeadline ||
    o.noShowParty ||
    o.cancelledBy ||
    deliveryEvidence.value.length
  )
})

/** 核销码只给 coser 看（后端摄影师视角恒为 null）。 */
const showCheckinCode = computed(() => !!order.value.checkinCode)

/** 待确认交付时才展示确认截止时间。 */
const showAcceptDeadline = computed(() => order.value.subStatus === 'delivered' && !!order.value.acceptDeadline)

const noShowText = computed(() => {
  const party = order.value.noShowParty
  if (party === 'coser') return 'coser'
  if (party === 'photographer') return '摄影师'
  if (party === 'both') return '双方'
  return ''
})

const cancelledByText = computed(() => {
  const by = order.value.cancelledBy
  if (by === 'coser') return 'coser'
  if (by === 'photographer') return '摄影师'
  if (by === 'system') return '系统'
  return ''
})

const disputeStatusLabel = computed(() => disputeStatusText(dispute.value?.status || ''))
const disputeTypeLabel = computed(() => disputeTypeText(dispute.value?.type || ''))

/** 争议响应倒计时。 */
const respondCountdown = computed(() => {
  void clock.value
  return countdownText(dispute.value?.respondDeadline)
})

/** 可以发起争议：confirmed 且处于可争议的子状态，且当前没有未关闭争议。 */
const canOpenDispute = computed(() => {
  const o = order.value
  if (o.status !== 'confirmed') return false
  if (dispute.value) return false
  return o.subStatus === 'in_service' || o.subStatus === 'delivered' || !o.subStatus
})

/** 可以响应争议：对方是申请方，且争议未超过响应截止时间。 */
const canRespondDispute = computed(() => {
  if (!dispute.value) return false
  if (myRole.value === dispute.value.openedByRole) return false
  return dispute.value.status === 'open' || dispute.value.status === 'responding'
})

/** 可以升级争议：争议未终结即可升级。 */
const canEscalateDispute = computed(() => {
  if (!dispute.value) return false
  return dispute.value.status === 'open' || dispute.value.status === 'responding'
})

const isOwnOrder = computed(() => {
  if (!userStore.user) return false
  // Stored/local rows carry no coserId but are still the user's own order.
  if (!order.value.coserId) return true
  return order.value.coserId === String(userStore.user.id)
})

const showRespondQuote = computed(() => (
  order.value.priceMode === 'negotiable' && order.value.priceStatus === 'quoted' && isOwnOrder.value
))

const showQuote = computed(() => (
  order.value.priceMode === 'negotiable' &&
  order.value.priceStatus === 'awaiting_quote' &&
  userStore.user?.photographerId != null &&
  order.value.photographerId === String(userStore.user.photographerId)
))

function acceptQuote() {
  if (!order.value.id) return
  confirmRespondQuote(order.value.id, true, loadOrder)
}

function rejectQuote() {
  if (!order.value.id) return
  confirmRespondQuote(order.value.id, false, loadOrder)
}

function quoteOrder() {
  if (!order.value.id) return
  promptQuote(order.value.id, loadOrder)
}

function goBack() {
  uni.navigateBack()
}

async function goChat() {
  if (!userStore.isLoggedIn || !userStore.user) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => uni.navigateTo({ url: '/pages/login/index?redirect=/pages/order/list' }), 800)
    return
  }
  if (!order.value.photographerId) {
    uni.showToast({ title: '订单信息缺失', icon: 'none' })
    return
  }
  try {
    const res = await apiPost<{ id: number }>('/v1/chat/sessions', {
      userId: Number(userStore.user.id),
      otherUserId: Number(order.value.photographerUserId || order.value.photographerId)
    })
    uni.navigateTo({
      url: `/pages/chat/index?sessionId=${res.id}&peerName=${encodeURIComponent(order.value.photographerName)}&peerAvatar=${encodeURIComponent(order.value.photographerAvatar || '')}`
    })
  } catch {
    uni.showToast({ title: '无法发起会话', icon: 'none' })
  }
}

async function updateStatus(newStatus: 'confirmed' | 'completed' | 'cancelled') {
  if (!order.value.id) return
  uni.showLoading({ title: '提交中...' })
  try {
    await apiPut(`/v1/bookings/${order.value.id}/status`, { status: newStatus })
    order.value = { ...order.value, status: newStatus }
    uni.showToast({ title: '操作成功', icon: 'success' })
  } catch (e) {
    const code = e instanceof ApiError ? e.status : 0
    uni.showToast({ title: code === 409 ? '操作不允许' : '操作失败', icon: 'none' })
  } finally {
    uni.hideLoading()
  }
}

function cancelOrder() {
  uni.showModal({
    title: '确认取消',
    content: '确定要取消这个预约吗？',
    success: (res) => { if (res.confirm) updateStatus('cancelled') }
  })
}

function confirmOrder() {
  uni.showModal({
    title: '确认完成',
    content: '确认拍摄已完成？',
    success: (res) => { if (res.confirm) updateStatus('completed') }
  })
}

/** 担保验收：确认收到摄影师交付的成片，订单转 completed。 */
function acceptDelivery() {
  if (!order.value.id) return
  uni.showModal({
    title: '确认交付',
    content: '确认已收到摄影师交付的成片？确认后订单将完成并结算。',
    success: async (res) => {
      if (!res.confirm) return
      uni.showLoading({ title: '提交中...' })
      try {
        await acceptDeliveryApi(order.value.id)
        uni.hideLoading()
        uni.showToast({ title: '已确认交付', icon: 'success' })
        await loadOrder()
      } catch (e) {
        uni.hideLoading()
        const code = e instanceof ApiError ? e.status : 0
        uni.showToast({ title: code === 409 ? '交付状态已变化，请刷新' : '操作失败', icon: 'none' })
      }
    }
  })
}

/** 争议类型选项（与后端 CHECK 约束一致）。 */
const DISPUTE_TYPES: { value: DisputeType; label: string }[] = [
  { value: 'no_show', label: '对方爽约' },
  { value: 'not_delivered', label: '未按约交付' },
  { value: 'quality', label: '成片质量' },
  { value: 'other', label: '其他' }
]

/** 发起争议：选类型 + 填理由。 */
function onOpenDispute() {
  if (!order.value.id) return
  uni.showActionSheet({
    itemList: DISPUTE_TYPES.map(t => t.label),
    success: (res) => {
      const picked = DISPUTE_TYPES[res.tapIndex]
      if (!picked) return
      uni.showModal({
        title: '发起争议',
        editable: true,
        placeholderText: `请说明${picked.label}的原因`,
        success: async (modalRes) => {
          if (!modalRes.confirm) return
          const reason = (modalRes.content || '').trim()
          if (!reason) {
            uni.showToast({ title: '请填写争议原因', icon: 'none' })
            return
          }
          uni.showLoading({ title: '提交中...' })
          try {
            await openDisputeApi(order.value.id, { type: picked.value, reason })
            uni.hideLoading()
            uni.showToast({ title: '争议已发起', icon: 'success' })
            await loadEscrowInfo()
          } catch (e) {
            uni.hideLoading()
            const code = e instanceof ApiError ? e.status : 0
            uni.showToast({
              title: code === 409 ? '该订单已有进行中的争议' : code === 403 ? '无权发起争议' : '发起失败',
              icon: 'none'
            })
          }
        }
      })
    }
  })
}

/** 响应争议：填写说明。 */
function onRespondDispute() {
  if (!dispute.value) return
  uni.showModal({
    title: '提交争议说明',
    editable: true,
    placeholderText: '请说明你的情况（必填）',
    success: async (res) => {
      if (!res.confirm) return
      const response = (res.content || '').trim()
      if (!response) {
        uni.showToast({ title: '请填写说明', icon: 'none' })
        return
      }
      uni.showLoading({ title: '提交中...' })
      try {
        await respondDisputeApi(dispute.value!.id, { response })
        uni.hideLoading()
        uni.showToast({ title: '已提交说明', icon: 'success' })
        await loadEscrowInfo()
      } catch (e) {
        uni.hideLoading()
        const code = e instanceof ApiError ? e.status : 0
        uni.showToast({
          title: code === 409 ? '争议状态已变化' : code === 403 ? '无权响应该争议' : '提交失败',
          icon: 'none'
        })
      }
    }
  })
}

/** 升级争议：等待平台裁决。 */
function onEscalateDispute() {
  if (!dispute.value) return
  uni.showModal({
    title: '升级裁决',
    editable: true,
    placeholderText: '补充说明（可留空）',
    success: async (res) => {
      if (!res.confirm) return
      const reason = (res.content || '').trim()
      uni.showLoading({ title: '提交中...' })
      try {
        await escalateDisputeApi(dispute.value!.id, reason || undefined)
        uni.hideLoading()
        uni.showToast({ title: '已升级', icon: 'success' })
        await loadEscrowInfo()
      } catch (e) {
        uni.hideLoading()
        const code = e instanceof ApiError ? e.status : 0
        uni.showToast({ title: code === 409 ? '争议状态已变化' : '升级失败', icon: 'none' })
      }
    }
  })
}

/** 预览交付凭证大图。 */
function previewEvidence(index: number) {
  const list = deliveryEvidence.value
  if (!list.length) return
  uni.previewImage({ current: index, urls: list })
}

function goReview() {
  uni.navigateTo({ url: `/pages/comment/index?photographerId=${order.value.photographerId}` })
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
  justify-content: space-between;
  padding: $spacing-sm $spacing-md;
  background: $dark-bg-secondary;
  border-bottom: 1rpx solid $dark-border;
}

.back-btn {
  width: 60rpx;
  height: 60rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.back-icon {
  font-size: 48rpx;
  color: $dark-text-primary;
}

.header-title {
  font-size: $font-size-lg;
  font-weight: 600;
  color: $dark-text-primary;
}

.placeholder {
  width: 60rpx;
}

.content {
  height: calc(100vh - 140rpx - env(safe-area-inset-top));
}

.status-card {
  padding: $spacing-lg;
  display: flex;
  flex-direction: column;
  align-items: center;

  &.confirmed {
    background: $neon-gradient;
  }

  &.pending {
    background: $neon-cyan;
  }

  &.completed {
    background: $success-color;
  }

  &.cancelled {
    background: $dark-bg-secondary;
    border-bottom: 1rpx solid $dark-border;
  }
}

.status-icon {
  font-size: 80rpx;
  margin-bottom: $spacing-sm;
  color: rgba(255, 255, 255, 0.9);
}

.status-text {
  font-size: $font-size-xl;
  font-weight: 600;
  color: #fff;
}

// 主状态卡背景是霓虹渐变（confirmed）或纯色（pending/completed），
// 子状态标签必须自己在任何底色上都可读：用深色实底 + 色调描边 + 色调文字。
.status-sub {
  font-size: $font-size-sm;
  font-weight: 600;
  margin-top: $spacing-xs;
  padding: 4rpx 20rpx;
  border-radius: $border-radius-full;
  background: rgba(10, 10, 26, 0.82);
  color: #fff;
  border: 1rpx solid rgba(255, 255, 255, 0.35);

  &.sub-cyan {
    color: $neon-cyan;
    border-color: $neon-cyan;
  }

  &.sub-purple {
    color: $neon-purple-bright;
    border-color: $neon-purple;
  }

  &.sub-warning {
    color: $warning-color;
    border-color: $warning-color;
  }

  &.sub-error {
    color: $error-color;
    border-color: $error-color;
  }
}

.order-id {
  font-size: $font-size-xs;
  color: rgba(255, 255, 255, 0.8);
  margin-top: $spacing-xs;
}

.section {
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-lg;
  margin: $spacing-md;
  padding: $spacing-md;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.2);
}

.section-title {
  font-size: $font-size-base;
  font-weight: 600;
  margin-bottom: $spacing-md;
  color: $dark-text-primary;
  padding-left: 20rpx;
  position: relative;

  &::before {
    content: '';
    position: absolute;
    left: 0;
    top: 50%;
    transform: translateY(-50%);
    width: 6rpx;
    height: 28rpx;
    background: $neon-gradient;
    border-radius: 3rpx;
  }
}

.photographer-info {
  display: flex;
  align-items: center;
}

.avatar {
  width: 120rpx;
  height: 120rpx;
  border-radius: $border-radius-md;
}

.info {
  flex: 1;
  margin-left: $spacing-md;
}

.name {
  font-size: $font-size-lg;
  font-weight: 500;
  display: block;
  color: $dark-text-primary;
}

.location {
  font-size: $font-size-sm;
  color: $dark-text-secondary;
  display: block;
  margin-top: 4rpx;
}

.btn-outline {
  padding: $spacing-sm $spacing-md;
  border: 2rpx solid $neon-purple;
  color: $neon-purple;
  border-radius: $border-radius-md;
  font-size: $font-size-sm;

  &.danger {
    border-color: rgba(239, 68, 68, 0.4);
    color: $error-color;
  }

  &:active {
    background: $dark-bg-card-hover;
    transform: scale(0.97);
    border-color: $neon-purple-glow;
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}

.service-detail {
  background: $dark-bg-secondary;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-md;
  padding: $spacing-sm;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  padding: $spacing-sm;

  &:not(:last-child) {
    border-bottom: 1rpx solid $dark-border;
  }
}

.detail-label {
  font-size: $font-size-sm;
  color: $dark-text-secondary;
}

.detail-value {
  font-size: $font-size-sm;
  color: $dark-text-primary;
}

.price-detail {
  background: $dark-bg-secondary;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-md;
  padding: $spacing-sm;
}

.price-row {
  display: flex;
  justify-content: space-between;
  padding: $spacing-sm;

  &.total {
    border-top: 1rpx solid $dark-border;
    margin-top: $spacing-sm;
    padding-top: $spacing-md;
  }
}

.price-label {
  font-size: $font-size-sm;
  color: $dark-text-secondary;
}

.price-value {
  font-size: $font-size-sm;
  color: $dark-text-primary;

  .total & {
    font-size: $font-size-lg;
    font-weight: 600;
    color: $neon-purple;
  }
}

.order-time {
  font-size: $font-size-sm;
  color: $dark-text-tertiary;
}

.bottom-space {
  height: 140rpx;
}

.footer {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  justify-content: flex-end;
  gap: $spacing-md;
  padding: $spacing-md;
  padding-bottom: calc(#{$spacing-md} + env(safe-area-inset-bottom));
  background: $dark-bg-secondary;
  border-top: 1rpx solid $dark-border;
  box-shadow: 0 -4rpx 16rpx rgba(0, 0, 0, 0.2);
}

.btn-primary {
  padding: $spacing-sm $spacing-xl;
  background: $neon-gradient;
  color: #fff;
  border-radius: $border-radius-lg;
  font-size: $font-size-base;

  &:active {
    transform: scale(0.97);
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}

/* ───────────────── 担保交易：进度 / 时间线 / 争议 ───────────────── */

.detail-value.code {
  font-family: monospace;
  font-size: $font-size-lg;
  font-weight: 600;
  letter-spacing: 4rpx;
  color: $neon-cyan;
}

.detail-value.warn {
  color: $warning-color;
}

.detail-value.error {
  color: $error-color;
}

.evidence {
  margin-top: $spacing-md;
}

.evidence-title {
  display: block;
  font-size: $font-size-sm;
  color: $dark-text-secondary;
  margin-bottom: $spacing-sm;
}

.evidence-list {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
}

.evidence-img {
  width: 180rpx;
  height: 180rpx;
  border-radius: $border-radius-md;
  border: 1rpx solid $dark-border;
  background: $dark-bg-secondary;
}

.timeline {
  display: flex;
  flex-direction: column;
}

.timeline-node {
  display: flex;
  gap: $spacing-sm;
}

.timeline-rail {
  width: 32rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  flex-shrink: 0;
}

.timeline-dot {
  width: 20rpx;
  height: 20rpx;
  border-radius: 50%;
  border: 2rpx solid $dark-border;
  background: $dark-bg-secondary;
  margin-top: 8rpx;

  &.done {
    border-color: $neon-purple;
    background: $neon-purple;
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}

.timeline-line {
  flex: 1;
  width: 2rpx;
  background: $dark-border;
  margin: 6rpx 0;
  min-height: 40rpx;
}

.timeline-body {
  flex: 1;
  padding-bottom: $spacing-md;
}

.timeline-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.timeline-title {
  font-size: $font-size-base;
  color: $dark-text-secondary;

  &.done {
    color: $dark-text-primary;
    font-weight: 600;
  }
}

.timeline-time {
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
}

.timeline-desc {
  display: block;
  font-size: $font-size-sm;
  color: $dark-text-tertiary;
  margin-top: 4rpx;
}

.timeline-actor {
  display: block;
  font-size: $font-size-xs;
  color: $neon-cyan;
  margin-top: 4rpx;
}

.dispute-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: $spacing-sm;
}

.dispute-type {
  font-size: $font-size-base;
  font-weight: 600;
  color: $dark-text-primary;
}

.dispute-status {
  font-size: $font-size-xs;
  padding: 4rpx 16rpx;
  border-radius: $border-radius-full;
  border: 1rpx solid $dark-border;
  color: $dark-text-secondary;

  &.d-open {
    color: $warning-color;
    border-color: $warning-color;
  }

  &.d-responding {
    color: $neon-cyan;
    border-color: $neon-cyan;
  }

  &.d-escalated {
    color: $error-color;
    border-color: $error-color;
  }

  &.d-resolved,
  &.d-closed {
    color: $success-color;
    border-color: $success-color;
  }
}

.dispute-reason {
  font-size: $font-size-sm;
  color: $dark-text-secondary;
  line-height: 1.6;
  margin-bottom: $spacing-md;
}

.dispute-actions {
  display: flex;
  gap: $spacing-md;
  margin-top: $spacing-md;
}
</style>
