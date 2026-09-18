<template>
  <view class="page">
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
      <picker mode="date" :value="selectedDate" @change="onDateChange">
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
        <text class="total-label">合计</text>
        <text class="total-price">¥{{ selectedService?.price || 0 }}</text>
      </view>
      <view 
        class="btn-primary" 
        :class="{ disabled: !canSubmit }"
        @click="submitBooking"
      >
        提交预约
      </view>
    </view>

  </view>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import ServiceCard from '@/components/ServiceCard.vue'
import { apiGet, apiPost, ApiError } from '@/api/client'
import { useUserStore } from '@/stores/user'
import type { Service } from '@/types'

const ALL_SLOTS = ['09:00', '10:00', '11:00', '13:00', '14:00', '15:00', '16:00', '17:00']

const userStore = useUserStore()

const services = ref<Service[]>([])
const selectedService = ref<Service | null>(null)
const photographerId = ref('1')
const photographerName = ref('摄影师')
const photographerAvatar = ref('')
const photographerLocation = ref('北京')
const selectedDate = ref('')
const selectedTime = ref('')
const timeSlots = ref<string[]>([])
const disabledTimes = ref<string[]>([])
const remark = ref('')

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

  const today = new Date()
  selectedDate.value = today.toISOString().split('T')[0]
  loadTimeSlots(selectedDate.value)
})

async function loadServices() {
  if (!photographerId.value) return
  try {
    const res = await apiGet<any>(`/v1/photographers/${photographerId.value}`)
    services.value = (res.services || []).map((s: any) => ({
      id: String(s.id), name: s.name, price: s.price,
      description: s.description, duration: s.duration,
    }))
    photographerName.value = res.name || '摄影师'
    photographerAvatar.value = res.avatar || ''
    photographerLocation.value = res.location || '北京'
    if (!selectedService.value && services.value.length) {
      selectedService.value = services.value[0]
    }
  } catch {
    uni.showToast({ title: '加载服务失败', icon: 'none' })
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

function isTimeDisabled(time: string): boolean {
  return disabledTimes.value.includes(time)
}

function selectTime(time: string) {
  if (isTimeDisabled(time)) return
  selectedTime.value = time
}

async function submitBooking() {
  if (!userStore.isLoggedIn) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => uni.navigateTo({ url: '/pages/login/index?redirect=/pages/booking/index' }), 800)
    return
  }
  if (!canSubmit.value) {
    uni.showToast({ title: '请完善预约信息', icon: 'none' })
    return
  }
  uni.showLoading({ title: '提交中...' })
  try {
    await apiPost<{ id: number }>('/v1/bookings', {
      photographerId: Number(photographerId.value),
      coserId: Number(userStore.user?.id),
      serviceId: Number(selectedService.value!.id),
      date: selectedDate.value,
      time: selectedTime.value,
      remarks: remark.value,
    })
    uni.showToast({ title: '预约成功', icon: 'success' })
    setTimeout(() => uni.navigateTo({ url: '/pages/order/list' }), 800)
  } catch (e) {
    uni.hideLoading()
    const code = e instanceof ApiError ? e.status : 0
    uni.showToast({ title: code === 409 ? '该时段已被预约' : '预约失败', icon: 'none' })
    return
  } finally {
    uni.hideLoading()
  }
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
  padding-bottom: 140rpx;
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
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
}

.time-slot {
  padding: $spacing-sm $spacing-lg;
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

.footer {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  align-items: center;
  padding: $spacing-md;
  padding-bottom: calc(#{$spacing-md} + env(safe-area-inset-bottom));
  background: $dark-bg-secondary;
  border-top: 1rpx solid $dark-border;
  box-shadow: 0 -2rpx 20rpx rgba(0, 0, 0, 0.3);
}

.total {
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

.btn-primary {
  flex: 1;
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
</style>
