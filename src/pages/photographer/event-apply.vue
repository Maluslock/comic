<template>
  <view class="page" :style="{ paddingTop: (statusBarHeight + 44) + 'px' }">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />
    <view class="nav-bar" :style="{ marginTop: statusBarHeight + 'px' }">
      <text class="back-arrow" @click="goBack">‹</text>
      <text class="nav-title">{{ isMine ? '我的漫展入驻' : '申请入驻' }}</text>
    </view>

    <view class="body">
      <!-- 首屏骨架：避免首帧直接落到空态/表单，视觉上像"没数据" -->
      <view v-if="loading" class="skeleton-wrap">
        <ListSkeleton :rows="4" variant="card" />
      </view>

      <!-- 未登录 -->
      <view v-else-if="!userStore.user" class="empty-state">
        <text class="empty-icon">◇</text>
        <text class="empty-text">登录后即可申请入驻漫展</text>
        <view class="btn-primary" @click="goLogin">去登录</view>
      </view>

      <!-- 非摄影师：入驻申请要求先开通摄影师身份（后端 403） -->
      <view v-else-if="notPhotographer" class="empty-state">
        <text class="empty-icon">◈</text>
        <text class="empty-text">成为摄影师后可申请入驻</text>
        <text class="empty-hint">开通后可向任意漫展提交入驻申请，审核通过即出现在漫展详情页</text>
        <view class="btn-primary" @click="goActivate">去开通摄影师</view>
      </view>

      <!-- 载入失败（漫展不存在 / 缺 id） -->
      <view v-else-if="loadError" class="empty-state">
        <text class="empty-icon">!</text>
        <text class="empty-text">{{ loadError }}</text>
        <view class="btn-outline" @click="goBack">返回</view>
      </view>

      <!-- ─────────── 我的漫展入驻：按状态分组 ─────────── -->
      <template v-else-if="isMine">
        <view v-if="groups.length === 0" class="empty-state">
          <text class="empty-icon">◇</text>
          <text class="empty-text">还没有申请过漫展入驻</text>
          <text class="empty-hint">在漫展详情页点「申请入驻本漫展」即可提交</text>
          <view class="btn-primary" @click="goEvents">去看看漫展</view>
        </view>

        <view v-else class="group-list">
          <view v-for="g in groups" :key="g.key" class="group">
            <view class="group-head">
              <text class="group-title">{{ g.label }}</text>
              <text class="group-count">{{ g.items.length }}</text>
            </view>

            <view
              v-for="item in g.items"
              :key="item.id"
              class="apply-card"
              :class="{ clickable: item.status === 'approved' }"
              @click="openEvent(item)"
            >
              <view class="apply-main">
                <image v-if="item.coverUrl" class="apply-cover" :src="resolveImage(item.coverUrl)" mode="aspectFill" />
                <view v-else class="apply-cover apply-cover--empty"><text>漫展</text></view>
                <view class="apply-info">
                  <view class="apply-head">
                    <text class="apply-name ellipsis">{{ item.eventName }}</text>
                    <text class="status-tag" :class="`tone-${item.status}`">{{ statusLabel(item.status) }}</text>
                  </view>
                  <text class="apply-meta">{{ formatRange(item.startDate, item.endDate) }}</text>
                  <text v-if="item.location || item.venue" class="apply-meta">
                    {{ [item.location, item.venue].filter(Boolean).join(' · ') }}
                  </text>
                  <text class="apply-time">申请于 {{ formatTime(item.appliedAt) }}</text>
                </view>
              </view>

              <view v-if="item.status === 'rejected' && item.reviewReason" class="apply-reason">
                驳回理由：{{ item.reviewReason }}
              </view>

              <view v-if="item.status === 'pending'" class="apply-footer">
                <view class="btn-outline danger" @click.stop="onWithdraw(item)">撤回申请</view>
              </view>
              <view v-else-if="item.status === 'approved'" class="apply-footer">
                <view class="btn-primary" @click.stop="openEvent(item)">查看漫展</view>
              </view>
              <view v-else-if="item.status === 'rejected'" class="apply-footer">
                <view class="btn-primary" @click.stop="reapply(item)">重新申请</view>
              </view>
              <view v-else-if="item.status === 'withdrawn'" class="apply-footer">
                <view class="btn-outline" @click.stop="reapply(item)">重新申请</view>
              </view>
            </view>
          </view>
        </view>
      </template>

      <!-- ─────────── 申请某漫展：漫展概要 + 表单 / 已申请状态 ─────────── -->
      <template v-else>
        <view v-if="event" class="event-card">
          <image v-if="event.cover" class="event-cover" :src="event.cover" mode="aspectFill" />
          <view class="event-info">
            <text class="event-name ellipsis-2">{{ event.name }}</text>
            <text class="event-meta">{{ formatRange(event.startDate, event.endDate) }}</text>
            <text v-if="event.location || event.venue" class="event-meta">
              {{ [event.location, event.venue].filter(Boolean).join(' · ') }}
            </text>
          </view>
        </view>

        <!-- 已提交且无需重填：显示状态，不重复给表单 -->
        <view v-if="application && !showForm" class="status-card" :class="`tone-${application.status}`">
          <text class="status-icon">{{ statusIcon(application.status) }}</text>
          <text class="status-title">{{ statusText(application.status) }}</text>
          <text class="status-meta">提交时间：{{ formatTime(application.appliedAt) }}</text>
          <text v-if="application.status === 'rejected'" class="status-reason">
            驳回理由：{{ application.reviewReason || '未填写' }}
          </text>
          <text v-else class="status-hint">{{ statusHint(application.status) }}</text>

          <view class="status-actions">
            <view v-if="application.status === 'pending'" class="btn-outline danger" @click="onWithdrawCurrent">
              撤回申请
            </view>
            <view v-if="application.status === 'rejected'" class="btn-primary" @click="showForm = true">
              重新申请
            </view>
            <view v-if="application.status === 'approved'" class="btn-primary" @click="openCurrentEvent">
              查看漫展详情
            </view>
          </view>
        </view>

        <!-- 表单：首次申请 / 已撤回 / 被驳回后点重新申请 -->
        <view v-if="showForm" class="form-card">
          <view class="form-header">
            <text class="form-title">我能提供什么</text>
            <text class="form-sub">写清设备、出图时效与样片链接，管理员审核更快</text>
          </view>

          <view class="field">
            <text class="field-label">申请说明</text>
            <textarea
              v-model="message"
              class="field-textarea"
              placeholder="例：佳能 R5 + 2 支闪光灯；场次当天拍完、7 天内出 30 张精修；样片 https://…"
              placeholder-style="color: #64748b"
              maxlength="500"
            />
            <text class="field-counter">{{ message.length }}/500</text>
          </view>

          <view v-if="eventEnded" class="form-tip form-tip--warn">该漫展已结束，无法再申请入驻</view>

          <view class="btn-submit" :class="{ disabled: !canSubmit }" @click="submit">
            {{ submitting ? '提交中...' : '提交申请' }}
          </view>
        </view>

        <view class="switch-link" @click="gotoMine">我的漫展入驻 ›</view>
      </template>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import ListSkeleton from '@/components/ListSkeleton.vue'
import { useStatusBar } from '@/composables/useStatusBar'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { apiGet, ApiError, NotFoundError } from '@/api/client'
import {
  applyEventPhotographer,
  getMyEventApplication,
  getMyEventApplications,
  withdrawEventApplication,
  type EventApplication,
  type MyEventApplication
} from '@/api/index'
import { useUserStore } from '@/stores/user'
import { mapEventItem, resolveImage, type EventDTO } from '@/utils/mappers'
import type { ComicEvent } from '@/types'
import { navTo } from '@/utils/nav'

const { statusBarHeight } = useStatusBar()
const userStore = useUserStore()

/** 路由：?id=<eventId> → 申请该漫展；?tab=mine → 我的入驻列表。 */
const eventId = ref('')
const isMine = ref(false)

const loading = ref(true)
const notPhotographer = ref(false)
const loadError = ref('')

const event = ref<ComicEvent | null>(null)
const application = ref<EventApplication | null>(null)
const list = ref<MyEventApplication[]>([])

const showForm = ref(false)
const message = ref('')
const submitting = ref(false)

const eventEnded = computed(() => event.value?.status === 'ended')

const canSubmit = computed(() => {
  if (submitting.value || eventEnded.value) return false
  return message.value.trim().length > 0
})

interface StatusGroup {
  key: string
  label: string
  items: MyEventApplication[]
}

const GROUP_ORDER: Array<{ key: string; label: string }> = [
  { key: 'pending', label: '待审核' },
  { key: 'approved', label: '已通过' },
  { key: 'rejected', label: '未通过' },
  { key: 'withdrawn', label: '已撤回' }
]

const groups = computed<StatusGroup[]>(() =>
  GROUP_ORDER.map((g) => ({
    key: g.key,
    label: g.label,
    items: list.value.filter((it) => it.status === g.key)
  })).filter((g) => g.items.length > 0)
)

onLoad((options: Record<string, string> | undefined) => {
  const id = options?.id || ''
  eventId.value = id
  isMine.value = options?.tab === 'mine' || !id
})

// onShow 而非 onMounted：从漫展详情/审核页返回时状态要立刻刷新（与 follow/list 一致）
onShow(() => {
  void load()
})

async function load() {
  if (!userStore.user) {
    loading.value = false
    return
  }
  loading.value = true
  notPhotographer.value = false
  loadError.value = ''

  try {
    if (isMine.value) {
      const res = await getMyEventApplications()
      list.value = res.list || []
    } else {
      if (!eventId.value) {
        loadError.value = '缺少漫展ID'
        return
      }
      await Promise.all([loadEvent(), loadApplication()])
    }
  } catch (e) {
    if (e instanceof ApiError && e.status === 403) {
      notPhotographer.value = true
    } else if (e instanceof NotFoundError) {
      loadError.value = '漫展不存在'
    } else if (!isMine.value && !event.value) {
      loadError.value = '加载失败，请稍后重试'
    }
  } finally {
    loading.value = false
  }
}

async function loadEvent() {
  const d = await apiGet<EventDTO>(`/v1/events/${eventId.value}`)
  event.value = mapEventItem(d)
}

async function loadApplication() {
  const res = await getMyEventApplication(eventId.value)
  application.value = res.application
  // 未申请 / 已撤回 → 直接给表单；pending/approved/rejected 先给状态卡
  const st = res.application?.status
  showForm.value = !res.application || st === 'withdrawn'
}

async function submit() {
  if (!canSubmit.value) {
    if (!message.value.trim()) uni.showToast({ title: '请填写你能提供什么', icon: 'none' })
    return
  }
  submitting.value = true
  uni.showLoading({ title: '提交中...' })
  try {
    await applyEventPhotographer(eventId.value, message.value.trim())
    uni.hideLoading()
    uni.showToast({ title: '已提交，等待审核', icon: 'success' })
    message.value = ''
    showForm.value = false
    await load()
  } catch (e) {
    uni.hideLoading()
    let title = '提交失败，请重试'
    if (e instanceof ApiError && e.status === 403) title = '仅摄影师可申请入驻'
    else if (e instanceof ApiError && e.status === 409) title = '已申请过本漫展'
    else if (e instanceof NotFoundError) title = '漫展不存在'
    uni.showToast({ title, icon: 'none' })
  } finally {
    submitting.value = false
  }
}

function onWithdrawCurrent() {
  if (!application.value) return
  doWithdraw(eventId.value)
}

function onWithdraw(item: MyEventApplication) {
  doWithdraw(String(item.eventId))
}

function doWithdraw(id: string) {
  uni.showModal({
    title: '撤回申请',
    content: '撤回后该漫展不再展示你的入驻信息，可重新申请。',
    success: async (res) => {
      if (!res.confirm) return
      uni.showLoading({ title: '撤回中...' })
      try {
        await withdrawEventApplication(id)
        uni.hideLoading()
        uni.showToast({ title: '已撤回', icon: 'none' })
      } catch (e) {
        uni.hideLoading()
        const code = e instanceof ApiError ? e.status : 0
        uni.showToast({ title: code === 404 ? '没有可撤回的申请' : '撤回失败', icon: 'none' })
      } finally {
        // 无论成败都按服务端真实状态重载：404 多半是后台刚处理过，别让页面停在旧的 pending
        await load()
      }
    }
  })
}

function reapply(item: MyEventApplication) {
  navTo(`/pages/photographer/event-apply?id=${item.eventId}`)
}

function openEvent(item: MyEventApplication) {
  if (item.status !== 'approved') return
  navTo(`/pages/event/detail?id=${item.eventId}`)
}

function openCurrentEvent() {
  if (!eventId.value) return
  navTo(`/pages/event/detail?id=${eventId.value}`)
}

function gotoMine() {
  navTo('/pages/photographer/event-apply?tab=mine')
}

function goActivate() {
  navTo('/pages/photographer/activate')
}

function goEvents() {
  uni.switchTab({ url: '/pages/index/index' })
}

function goLogin() {
  navTo('/pages/login/index?redirect=' + encodeURIComponent('/pages/photographer/event-apply'))
}

function goBack() {
  uni.navigateBack()
}

/* ───────────────── 文案与格式化 ───────────────── */

const STATUS_LABEL: Record<string, string> = {
  pending: '待审核',
  approved: '已通过',
  rejected: '未通过',
  withdrawn: '已撤回'
}

function statusLabel(status: string): string {
  return STATUS_LABEL[status] || status
}

function statusIcon(status: string): string {
  if (status === 'approved') return '★'
  if (status === 'rejected') return '⚠'
  if (status === 'withdrawn') return '—'
  return '◇'
}

function statusText(status: string): string {
  if (status === 'approved') return '已通过 · 已入驻'
  if (status === 'rejected') return '未通过'
  if (status === 'withdrawn') return '已撤回'
  return '待审核'
}

function statusHint(status: string): string {
  if (status === 'approved') return '你已出现在该漫展详情页的参展摄影师中'
  if (status === 'withdrawn') return '可重新提交申请'
  return '管理员审核通过后，你会在该漫展详情页展示'
}

function pad(n: number): string {
  return n < 10 ? `0${n}` : String(n)
}

function toDate(v?: string | number | null): Date | null {
  if (v === undefined || v === null || v === '') return null
  const d = new Date(v)
  return isNaN(d.getTime()) ? null : d
}

function formatDay(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function formatRange(start?: string | number | null, end?: string | number | null): string {
  const s = toDate(start)
  const e = toDate(end)
  if (!s) return ''
  if (!e) return formatDay(s)
  if (formatDay(s) === formatDay(e)) return formatDay(s)
  if (s.getFullYear() === e.getFullYear()) return `${formatDay(s)} 至 ${pad(e.getMonth() + 1)}-${pad(e.getDate())}`
  return `${formatDay(s)} 至 ${formatDay(e)}`
}

function formatTime(iso?: string | null): string {
  const d = toDate(iso)
  if (!d) return ''
  return `${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
}

.status-bar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  background: $dark-bg-primary;
  z-index: 100;
}

.nav-bar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 44px;
  display: flex;
  align-items: center;
  background: $dark-bg-primary;
  border-bottom: 1rpx solid $dark-border;
  z-index: 99;
}

.back-arrow {
  font-size: 48rpx;
  color: $dark-text-primary;
  padding: 0 24rpx;
  line-height: 1;
}

.nav-title {
  font-size: 34rpx;
  font-weight: 600;
  color: $dark-text-primary;
}

.body {
  padding: 24rpx 32rpx 48rpx;
}

.skeleton-wrap {
  margin-top: 8rpx;
}

/* ───── 空态 ───── */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 160rpx;
}

.empty-icon {
  font-size: 100rpx;
  color: $neon-cyan;
  text-shadow: 0 0 24rpx $neon-cyan-glow;
}

.empty-text {
  font-size: 28rpx;
  color: $dark-text-secondary;
  margin: 24rpx 0 8rpx;
}

.empty-hint {
  font-size: 22rpx;
  color: $dark-text-tertiary;
  text-align: center;
  line-height: 1.6;
  margin-bottom: 32rpx;
}

/* ───── 我的入驻：分组列表 ───── */
.group-list {
  display: flex;
  flex-direction: column;
  gap: 32rpx;
}

.group-head {
  display: flex;
  align-items: center;
  gap: 12rpx;
  margin-bottom: 16rpx;
}

.group-title {
  font-size: 28rpx;
  font-weight: 600;
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
    height: 26rpx;
    background: $neon-gradient;
    border-radius: 3rpx;
  }
}

.group-count {
  font-size: 22rpx;
  color: $dark-text-tertiary;
}

.apply-card {
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: 20rpx;
  padding: 24rpx;
  margin-bottom: 20rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.2);

  &.clickable:active {
    transform: scale(0.99);
    background: $dark-bg-card-hover;
  }
}

.apply-main {
  display: flex;
  align-items: flex-start;
}

.apply-cover {
  width: 160rpx;
  height: 120rpx;
  border-radius: 12rpx;
  background: $dark-bg-secondary;
  flex-shrink: 0;
}

.apply-cover--empty {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22rpx;
  color: $dark-text-tertiary;
  border: 1rpx solid $dark-border;
  box-sizing: border-box;
}

.apply-info {
  flex: 1;
  min-width: 0;
  margin-left: 20rpx;
}

.apply-head {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.apply-name {
  flex: 1;
  min-width: 0;
  font-size: 28rpx;
  font-weight: 600;
  color: $dark-text-primary;
}

.apply-meta {
  display: block;
  font-size: 22rpx;
  color: $dark-text-tertiary;
  margin-top: 6rpx;
}

.apply-time {
  display: block;
  font-size: 20rpx;
  color: $dark-text-tertiary;
  margin-top: 8rpx;
}

.apply-reason {
  margin-top: 16rpx;
  padding-top: 16rpx;
  border-top: 1rpx solid $dark-border;
  font-size: 22rpx;
  color: $error-color;
  line-height: 1.5;
}

.apply-footer {
  display: flex;
  justify-content: flex-end;
  gap: 20rpx;
  margin-top: 20rpx;
}

.status-tag {
  flex-shrink: 0;
  font-size: 22rpx;
  padding: 4rpx 14rpx;
  border-radius: 999rpx;
  border: 1rpx solid $dark-border;
  background: rgba(255, 255, 255, 0.04);
  color: $dark-text-secondary;

  &.tone-pending {
    color: $neon-cyan;
    border-color: rgba(6, 182, 212, 0.45);
  }

  &.tone-approved {
    color: $success-color;
    border-color: rgba(34, 197, 94, 0.45);
  }

  &.tone-rejected {
    color: $error-color;
    border-color: rgba(239, 68, 68, 0.45);
  }

  &.tone-withdrawn {
    color: $dark-text-tertiary;
  }
}

/* ───── 申请某漫展：漫展概要 ───── */
.event-card {
  display: flex;
  align-items: center;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: 20rpx;
  padding: 24rpx;
  margin-bottom: 24rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.2);
}

.event-cover {
  width: 180rpx;
  height: 130rpx;
  border-radius: 12rpx;
  background: $dark-bg-secondary;
  flex-shrink: 0;
}

.event-info {
  flex: 1;
  min-width: 0;
  margin-left: 20rpx;
}

.event-name {
  display: block;
  font-size: 30rpx;
  font-weight: 700;
  color: $dark-text-primary;
  line-height: 1.4;
}

.event-meta {
  display: block;
  font-size: 22rpx;
  color: $dark-text-secondary;
  margin-top: 8rpx;
}

/* ───── 已申请：状态卡 ───── */
.status-card {
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: 24rpx;
  padding: 56rpx 40rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  box-shadow: 0 4rpx 24rpx rgba(0, 0, 0, 0.25);

  .status-icon {
    font-size: 88rpx;
    margin-bottom: 20rpx;
    line-height: 1;
  }

  .status-title {
    font-size: 36rpx;
    font-weight: 700;
    margin-bottom: 16rpx;
  }

  .status-meta {
    font-size: 24rpx;
    color: $dark-text-tertiary;
    margin-bottom: 12rpx;
  }

  .status-hint {
    font-size: 24rpx;
    color: $dark-text-secondary;
    text-align: center;
    line-height: 1.6;
    margin-top: 8rpx;
  }

  .status-reason {
    font-size: 26rpx;
    color: $dark-text-secondary;
    text-align: center;
    line-height: 1.6;
    margin-top: 8rpx;
  }

  &.tone-pending {
    border-color: rgba(6, 182, 212, 0.4);

    .status-icon,
    .status-title {
      color: $neon-cyan;
      text-shadow: 0 0 16rpx $neon-cyan-glow;
    }
  }

  &.tone-approved {
    border-color: rgba(34, 197, 94, 0.4);

    .status-icon,
    .status-title {
      color: $success-color;
      text-shadow: 0 0 16rpx rgba(34, 197, 94, 0.4);
    }
  }

  &.tone-rejected {
    border-color: rgba(239, 68, 68, 0.4);

    .status-icon,
    .status-title {
      color: $error-color;
      text-shadow: 0 0 16rpx rgba(239, 68, 68, 0.4);
    }
  }

  &.tone-withdrawn {
    .status-icon,
    .status-title {
      color: $dark-text-tertiary;
    }
  }
}

.status-actions {
  display: flex;
  gap: 20rpx;
  margin-top: 40rpx;
}

/* ───── 申请表单 ───── */
.form-card {
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: 24rpx;
  padding: 40rpx 32rpx;
  box-shadow: 0 4rpx 24rpx rgba(0, 0, 0, 0.25);
}

.form-header {
  margin-bottom: 32rpx;
}

.form-title {
  display: block;
  font-size: 34rpx;
  font-weight: 700;
  color: $dark-text-primary;
  margin-bottom: 12rpx;
}

.form-sub {
  display: block;
  font-size: 24rpx;
  color: $dark-text-tertiary;
  line-height: 1.6;
}

.field {
  margin-bottom: 24rpx;
}

.field-label {
  display: block;
  font-size: 26rpx;
  color: $dark-text-secondary;
  margin-bottom: 16rpx;
}

.field-textarea {
  width: 100%;
  height: 260rpx;
  background: $dark-bg-secondary;
  border: 1rpx solid $dark-border;
  border-radius: 16rpx;
  padding: 20rpx 24rpx;
  font-size: 28rpx;
  color: $dark-text-primary;
  line-height: 1.6;
  box-sizing: border-box;
}

.field-counter {
  display: block;
  text-align: right;
  font-size: 20rpx;
  color: $dark-text-tertiary;
  margin-top: 8rpx;
}

.form-tip {
  font-size: 24rpx;
  color: $dark-text-secondary;
  margin-bottom: 16rpx;

  &--warn {
    color: $warning-color;
  }
}

.btn-submit {
  margin-top: 24rpx;
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

  &.disabled {
    opacity: 0.4;
    box-shadow: none;
  }
}

.switch-link {
  margin-top: 32rpx;
  text-align: center;
  font-size: 26rpx;
  color: $neon-cyan;
  padding: 16rpx 0;

  &:active {
    opacity: 0.7;
  }
}

/* ───── 按钮（与接单管理页同款） ───── */
.btn-primary {
  padding: 14rpx 36rpx;
  background: $neon-purple;
  color: #fff;
  border-radius: 32rpx;
  font-size: 26rpx;
  box-shadow: 0 0 16rpx $neon-purple-glow;

  &:active {
    transform: scale(0.97);
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}

.btn-outline {
  padding: 14rpx 36rpx;
  border: 2rpx solid $dark-border;
  border-radius: 32rpx;
  font-size: 26rpx;
  color: $dark-text-secondary;

  &.danger {
    color: $error-color;
    border-color: rgba(239, 68, 68, 0.4);
  }

  &:active {
    transform: scale(0.97);
    background: $dark-bg-card-hover;
  }
}
</style>
