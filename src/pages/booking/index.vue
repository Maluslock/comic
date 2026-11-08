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
        <view class="sk-card sk-pad"><view class="sk-row"><view class="sk-circle sk-avatar-88" /><view class="sk-line sk-w-40" /></view></view>
        <ListSkeleton :rows="4" variant="card" />
      </view>
    </template>
    <template v-else>
    <view class="ph-card">
      <image v-if="photographerAvatar" :src="photographerAvatar" class="ph-avatar" mode="aspectFill" />
      <view v-else class="ph-avatar ph-avatar--initial">{{ photographerName.slice(0, 1) }}</view>
      <view class="ph-info">
        <text class="ph-name">{{ photographerName }}</text>
        <text class="ph-meta">{{ photographerLocation }} · 摄影师</text>
      </view>
      <text class="ph-hint">预约 TA</text>
    </view>

    <view class="section">
      <view class="section-title">选择服务</view>
      <ServiceCard 
        v-for="service in services" 
        :key="service.id" 
        :service="service"
        :selected="selectedService?.id === service.id"
        @select="selectService"
      />
    </view>

    <view class="section">
      <view class="section-title">选择日期</view>
      <picker mode="date" :value="selectedDate" :start="minDate" @change="onDateChange">
        <view class="date-picker">
          <text class="date-value">{{ selectedDate || '请选择日期' }}</text>
          <text class="date-arrow">›</text>
        </view>
      </picker>
    </view>

    <view class="section">
      <view class="section-title">选择时间</view>
      <view class="time-slots">
        <view 
          v-for="time in timeSlots" 
          :key="time" 
          class="time-slot"
          :class="{ active: selectedTime === time, disabled: isTimeDisabled(time) }"
          @click="selectTime(time)"
        >
          {{ time }}
        </view>
      </view>
    </view>

    <!-- 按张计价：张数决定总价，必须让用户在下单前就选，否则后端只按 1 张占位、价格待议 -->
    <view v-if="isPerPhoto" class="section">
      <view class="section-title">拍摄张数</view>
      <view class="photo-count-row">
        <view class="stepper">
          <text
            class="stepper-btn"
            :class="{ disabled: photoCount <= 1 }"
            @click="stepPhotoCount(-1)"
          >−</text>
          <text class="stepper-value">{{ photoCount }}</text>
          <text
            class="stepper-btn"
            :class="{ disabled: photoCount >= 99 }"
            @click="stepPhotoCount(1)"
          >+</text>
        </view>
        <text class="photo-count-hint">¥{{ selectedService?.price }} / 张</text>
      </view>
    </view>

    <view class="section">
      <view class="section-title">备注信息</view>
      <textarea 
        class="remark-input" 
        v-model="remark"
        placeholder="请输入拍摄要求、角色信息等..."
        :maxlength="500"
      />
      <text class="remark-count">{{ remark.length }}/500</text>
    </view>

    <view class="bottom-space"></view>

    <view class="footer">
      <view class="total">
        <view class="total-row">
          <text class="total-label">合计</text>
          <text class="total-price">{{ totalPriceText }}</text>
        </view>
        <!-- 按张套餐把算式摊开：只给一个 ¥30 用户不知道是 3 张算出来的 -->
        <text v-if="isPerPhoto && selectedService?.price" class="quote-hint">
          ¥{{ selectedService?.price }} × {{ photoCount }} 张
        </text>
        <text v-else-if="selectedService?.price === null" class="quote-hint">面议套餐：提交后等待摄影师报价</text>
      </view>
      <view 
        class="btn-primary" 
        :class="{ disabled: !canSubmit }"
        @click="submitBooking"
      >
        提交预约
      </view>
    </view>

    </template>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import ServiceCard from '@/components/ServiceCard.vue'
import { apiGet, apiPost, ApiError } from '@/api/client'
import { useUserStore } from '@/stores/user'
import { TMPL, requestSubscribe, markSubscribed } from '@/utils/subscribe'
import { formatPrice, encodeRedirect, resolveImage } from '@/utils/mappers'
import { todayKey } from '@/utils/date'
import { navTo } from '@/utils/nav'
import type { Service } from '@/types'
import ListSkeleton from '@/components/ListSkeleton.vue'

/** 首屏深色骨架：数据就绪前置 true，消除切页白屏（B 档） */
const firstLoading = ref(true)

const ALL_SLOTS = ['09:00', '10:00', '11:00', '13:00', '14:00', '15:00', '16:00', '17:00']

const userStore = useUserStore()

const services = ref<Service[]>([])
const selectedService = ref<Service | null>(null)
const photographerId = ref('1')
const photographerName = ref('摄影师')
const photographerAvatar = ref('')
const photographerLocation = ref('北京')
const selectedDate = ref('')
// 日期选择器下限 = 今天（本地时区）。服务端也会拒过去日期，这里只是别让用户白选。
const minDate = todayKey()
const selectedTime = ref('')
const timeSlots = ref<string[]>([])
const disabledTimes = ref<string[]>([])
const remark = ref('')
// 按张计价的张数：默认 1 张。后端 photoCount 是 *int32，非按张套餐绝不下发这个字段。
const photoCount = ref(1)

const isPerPhoto = computed(() => selectedService.value?.priceUnit === 'per_photo')

// 面议（price === null）不能乘；按张只在单价有效时相乘；其它计价方式原样显示。
const totalPriceText = computed(() => {
  const svc = selectedService.value
  if (!svc) return ''
  if (svc.price === null) return '面议'
  if (isPerPhoto.value) return formatPrice((svc.price || 0) * photoCount.value)
  return formatPrice(svc.price)
})

function stepPhotoCount(delta: number) {
  const next = photoCount.value + delta
  if (next < 1 || next > 99) return
  photoCount.value = next
}

const canSubmit = computed(() => {
  return selectedService.value && selectedDate.value && selectedTime.value
})

onLoad((options: Record<string, string> | undefined) => {
  const routePid = options?.photographerId
  const storedPhotographerId = uni.getStorageSync('bookingPhotographerId')
  photographerId.value = routePid || storedPhotographerId || '1'
})

onMounted(() => {
  const storedService = uni.getStorageSync('selectedService')
  if (storedService) {
    try {
      selectedService.value = JSON.parse(storedService)
    } catch {}
  }

  loadServices()

  // 用本地日期，不能用 toISOString（UTC）：东八区 00:00–08:00 会得到「昨天」，
  // 默认就选中一个已过去的日期。若今天已没有可约时段，直接落到明天，
  // 避免用户选完日期才发现「每个时段都是灰的」。
  selectedDate.value = ALL_SLOTS.some((t) => !isPastSlotLabel(t)) ? todayKey() : tomorrowKey()
  loadTimeSlots(selectedDate.value)
})

async function loadServices() {
  if (!photographerId.value) { firstLoading.value = false; return }
  try {
    const res = await apiGet<any>(`/v1/photographers/${photographerId.value}`)
    services.value = (res.services || []).map((s: any) => ({
      id: String(s.id), name: s.name, price: s.price,
      description: s.description, duration: s.duration,
      // priceUnit / photoCount 必须带上：丢了它们 isPerPhoto 恒 false，
      // 按张套餐会被当成「整套 ¥10」直接下单。
      priceUnit: s.priceUnit || 'fixed',
      photoCount: typeof s.photoCount === 'number' ? s.photoCount : null,
    }))
    photographerName.value = res.name || '摄影师'
    // 后端 avatar 是 /static/remote/** 或 /static/uploads/** 相对路径，小程序端会被当包内资源读（404 →
    // 只剩 ph-avatar 的深色底）。补成绝对地址后再进 <image :src>。
    photographerAvatar.value = resolveImage(res.avatar || '')
    photographerLocation.value = res.location || '北京'
    // storage 里存的可能是旧版本页面写入的残缺对象（没有 priceUnit / photoCount）。
    // 只信任 id，其余字段一律以刚拉到的完整列表为准，否则 isPerPhoto 恒 false。
    if (selectedService.value) {
      const fresh = services.value.find((s) => String(s.id) === String(selectedService.value!.id))
      if (fresh) {
        selectedService.value = { ...selectedService.value, ...fresh }
      } else {
        // 套餐已被摄影师下线：清掉失效选择，让下面的兜底重新挑第一个。
        selectedService.value = null
      }
    }
    if (!selectedService.value && services.value.length) {
      selectedService.value = services.value[0]
    }
  } catch {
    uni.showToast({ title: '加载服务失败', icon: 'none' })
  } finally {
    firstLoading.value = false
  }
}

async function loadTimeSlots(date: string) {
  timeSlots.value = ALL_SLOTS
  disabledTimes.value = []
  if (!photographerId.value) return
  try {
    const res = await apiGet<{ occupied: string[] }>(`/v1/photographers/${photographerId.value}/timeslots`, { date })
    disabledTimes.value = res.occupied || []
  } catch {
    // 后端不可达时不展示禁用状态（不阻塞预约提交——后端仍会做冲突兜底）
  }
}

function selectService(service: Service) {
  selectedService.value = service
}

function onDateChange(e: { detail: { value: string } }) {
  selectedDate.value = e.detail.value
  selectedTime.value = ''
  loadTimeSlots(e.detail.value)
}

/** 当天已过去的时段（含当前这一小时）不可选，与后端 isPastSlot 口径一致。 */
function isPastSlotLabel(time: string): boolean {
  if (selectedDate.value !== todayKey()) return false
  const [h, m] = time.split(':').map(Number)
  const now = new Date()
  return h * 60 + m <= now.getHours() * 60 + now.getMinutes()
}

function tomorrowKey(): string {
  const d = new Date()
  d.setDate(d.getDate() + 1)
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}

function isTimeDisabled(time: string): boolean {
  return disabledTimes.value.includes(time) || isPastSlotLabel(time)
}

function selectTime(time: string) {
  if (isTimeDisabled(time)) return
  selectedTime.value = time
}

async function submitBooking() {
  if (!userStore.isLoggedIn) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    // 回跳必须带 photographerId：否则登录回来 onLoad 会退回默认的 1 号摄影师
    const back = encodeRedirect(`/pages/booking/index?photographerId=${photographerId.value}`)
    setTimeout(() => navTo(`/pages/login/index?redirect=${back}`), 800)
    return
  }
  if (!canSubmit.value) {
    uni.showToast({ title: '请完善预约信息', icon: 'none' })
    return
  }
  // 一次性订阅必须在用户点击里先要额度：这里一次拿「结果通知 + 交付通知」两张，
  // 之后摄影师确认、交付时服务端才有额度可发。用户拒绝也不影响下单。
  const sub = await requestSubscribe([TMPL.result, TMPL.delivered])
  if (sub?.[TMPL.result] === 'accept') markSubscribed(TMPL.result)
  if (sub?.[TMPL.delivered] === 'accept') markSubscribed(TMPL.delivered)

  uni.showLoading({ title: '提交中...' })
  try {
    await apiPost<{ id: number }>('/v1/bookings', {
      photographerId: Number(photographerId.value),
      coserId: Number(userStore.user?.id),
      serviceId: Number(selectedService.value!.id),
      date: selectedDate.value,
      time: selectedTime.value,
      remarks: remark.value,
      // 只有按张套餐才下发张数。后端字段是 *int32，传 0 / 空串都会 400；
      // 非按张套餐带上它会污染快照（fixed 套餐不该有 photo_count）。
      ...(isPerPhoto.value ? { photoCount: photoCount.value } : {}),
    })
    // 成功后清掉跨页面的临时选择，否则下一次给别人预约会预填这次的套餐
    uni.removeStorageSync('selectedService')
    uni.removeStorageSync('bookingPhotographerId')
    uni.hideLoading()
    uni.showToast({ title: '预约成功', icon: 'success' })
    setTimeout(() => navTo('/pages/order/list'), 800)
  } catch (e) {
    uni.hideLoading()
    const code = e instanceof ApiError ? e.status : 0
    // 后端已把原因写在 {"error": "…"} 里（如「不能预约已经过去的日期」），
    // 能拿到就直接展示，别统一糊成「预约失败」。
    const reason = e instanceof ApiError && e.message && e.message !== 'request failed' ? e.message : ''
    const fallback = code === 409 ? '该时段已被预约，换个时间吧' : '预约失败，请稍后再试'
    uni.showToast({ title: reason || fallback, icon: 'none', duration: 2500 })
    return
  }
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
  padding-bottom: 140rpx;
  &::before {
    content: '';
    position: fixed;
    top: var(--window-top, 0px);
    right: 0;
    bottom: 0;
    left: 0;
    background: $surface-page;
    z-index: -1;
    pointer-events: none;
  }
}

.ph-card {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin: $spacing-md $spacing-md 0;
  padding: $spacing-md;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-lg;
}

.ph-avatar {
  width: 88rpx;
  height: 88rpx;
  border-radius: 50%;
  background: $surface-2;
}

.ph-avatar--initial {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 36rpx;
  font-weight: 600;
  color: $brand;
  background: rgba(255, 107, 0, 0.12);
}

.ph-info {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.ph-name {
  font-size: $font-size-md;
  font-weight: 600;
  color: $dark-text-primary;
}

.ph-meta {
  font-size: 24rpx;
  color: $text-3;
}

.ph-hint {
  margin-left: auto;
  font-size: 24rpx;
  color: $text-3;
}

.section {
  background: $dark-bg-card;
  margin-top: $spacing-md;
  padding: $spacing-md;
  border: 1rpx solid $dark-border;
  box-shadow: 0 4rpx 20rpx rgba(0, 0, 0, 0.2);
}

.section-title {
  position: relative;
  padding-left: 20rpx;
  font-size: $font-size-lg;
  font-weight: 600;
  color: $dark-text-primary;
  margin-bottom: $spacing-md;

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

.date-picker {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: $spacing-md;
  background: $dark-bg-secondary;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-md;

  &:active {
    background: $dark-bg-card-hover;
    transform: scale(0.97);
    border-color: $neon-purple-glow;
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}

.date-value {
  font-size: $font-size-base;
  color: $dark-text-primary;
}

.date-arrow {
  font-size: $font-size-xl;
  color: $dark-text-tertiary;
}

.time-slots {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16rpx;
}

.time-slot {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  padding: 0 8rpx;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-md;
  font-size: $font-size-sm;
  color: $dark-text-secondary;

  &.active {
    background: $neon-purple;
    color: #fff;
    border-color: $neon-purple;
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }

  &.disabled {
    background: $dark-bg-card-hover;
    color: $dark-text-tertiary;
    opacity: 0.5;
  }
}

.remark-input {
  width: 100%;
  box-sizing: border-box;
  height: 200rpx;
  padding: $spacing-md;
  background: $dark-bg-secondary;
  color: $dark-text-primary;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-md;
  font-size: $font-size-base;

  &::placeholder {
    color: $dark-text-tertiary;
  }
}

.remark-count {
  display: block;
  text-align: right;
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  margin-top: $spacing-xs;
}

.bottom-space {
  height: $spacing-xl;
}

/* 按张计价的张数行：步进器左右各 44px 热区，中间是可读的大数字 */
.photo-count-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
}

.stepper {
  display: inline-flex;
  align-items: center;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-md;
  overflow: hidden;
}

.stepper-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  min-height: 88rpx;
  font-size: 40rpx;
  line-height: 1;
  color: $dark-text-primary;

  &.disabled {
    color: $dark-text-tertiary;
  }
}

.stepper-value {
  min-width: 96rpx;
  text-align: center;
  font-size: $font-size-lg;
  font-weight: 600;
  color: $dark-text-primary;
}

.photo-count-hint {
  font-size: $font-size-sm;
  color: $dark-text-secondary;
}

.footer {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  padding: $spacing-md;
  padding-bottom: calc(#{$spacing-md} + env(safe-area-inset-bottom));
  background: $dark-bg-secondary;
  border-top: 1rpx solid $dark-border;
  box-shadow: 0 -2rpx 20rpx rgba(0, 0, 0, 0.3);
}

.total {
  flex: 0 0 auto;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
}

.total-row {
  display: flex;
  align-items: baseline;
}

.total-label {
  font-size: $font-size-base;
  color: $dark-text-secondary;
}

.total-price {
  font-size: $font-size-xxl;
  font-weight: 600;
  color: $neon-purple;
  margin-left: $spacing-xs;
}

.quote-hint {
  font-size: $font-size-xs;
  color: $neon-cyan;
  margin-top: 4rpx;
}

.btn-primary {
  flex: 1;
  box-sizing: border-box;
  min-width: 0;
  background: $neon-gradient;
  color: #fff;
  border-radius: $border-radius-lg;
  padding: $spacing-md;
  text-align: center;
  font-size: $font-size-base;
  font-weight: 500;
  margin-left: $spacing-lg;
  box-shadow: 0 4rpx 16rpx $neon-purple-glow;

  &:active {
    transform: scale(0.97);
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }

  &.disabled {
    background: $dark-bg-card-hover;
    color: $dark-text-tertiary;
    box-shadow: none;
  }
}

.picker-trigger {
  display: none;
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
