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
      <view class="date-picker" @click="pickDate">
        <text class="date-value">{{ selectedDate || '请选择日期' }}</text>
        <text class="date-arrow">›</text>
      </view>
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

    <picker mode="date" :value="selectedDate" @change="onDateChange">
      <view class="picker-trigger"></view>
    </picker>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import ServiceCard from '@/components/ServiceCard.vue'
import { getServices, getTimeSlots } from '@/api/index'
import type { Service } from '@/types'

const services = ref<Service[]>([])
const selectedService = ref<Service | null>(null)
const selectedDate = ref('')
const selectedTime = ref('')
const timeSlots = ref<string[]>([])
const remark = ref('')
const disabledTimes = ref<string[]>([])

const canSubmit = computed(() => {
  return selectedService.value && selectedDate.value && selectedTime.value
})

onMounted(() => {
  loadServices()
  const today = new Date()
  selectedDate.value = today.toISOString().split('T')[0]
  loadTimeSlots(selectedDate.value)
  
  const storedService = uni.getStorageSync('selectedService')
  if (storedService) {
    try {
      selectedService.value = JSON.parse(storedService)
    } catch {}
  }
})

async function loadServices() {
  services.value = await getServices()
  if (!selectedService.value && services.value.length > 0) {
    selectedService.value = services.value[0]
  }
}

async function loadTimeSlots(date: string) {
  const slots = await getTimeSlots(date)
  timeSlots.value = slots
  disabledTimes.value = ['10:00', '15:00']
}

function selectService(service: Service) {
  selectedService.value = service
}

function pickDate() {
  const picker = document.querySelector('.picker-trigger') as HTMLElement
  picker?.click()
}

function onDateChange(e: any) {
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
  if (!canSubmit.value) {
    uni.showToast({ title: '请完善预约信息', icon: 'none' })
    return
  }

  uni.showLoading({ title: '提交中...' })
  try {
    await createBooking({
      photographerId: '1',
      coserId: 'user1',
      serviceId: selectedService.value!.id,
      date: selectedDate.value,
      time: selectedTime.value,
      remarks: remark.value
    })
    
    uni.hideLoading()
    uni.showToast({ title: '预约成功', icon: 'success' })
    
    setTimeout(() => {
      uni.navigateTo({ url: '/pages/order/list' })
    }, 1500)
  } catch (error) {
    uni.hideLoading()
    uni.showToast({ title: '预约失败', icon: 'none' })
  }
}

async function createBooking(data: {
  photographerId: string
  coserId: string
  serviceId: string
  date: string
  time: string
  remarks?: string
}) {
  return new Promise(resolve => setTimeout(resolve, 1000))
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $bg-page;
  padding-bottom: 140rpx;
}

.section {
  background: $bg-primary;
  margin-top: $spacing-md;
  padding: $spacing-md;
}

.section-title {
  font-size: $font-size-lg;
  font-weight: 600;
  margin-bottom: $spacing-md;
}

.date-picker {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: $spacing-md;
  background: $bg-secondary;
  border-radius: $border-radius-md;
}

.date-value {
  font-size: $font-size-base;
  color: $text-primary;
}

.date-arrow {
  font-size: $font-size-xl;
  color: $text-tertiary;
}

.time-slots {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
}

.time-slot {
  padding: $spacing-sm $spacing-lg;
  background: $bg-secondary;
  border-radius: $border-radius-md;
  font-size: $font-size-sm;
  color: $text-secondary;
  
  &.active {
    background: $primary-color;
    color: #fff;
  }
  
  &.disabled {
    background: #f0f0f0;
    color: #ccc;
  }
}

.remark-input {
  width: 100%;
  height: 200rpx;
  padding: $spacing-md;
  background: $bg-secondary;
  border-radius: $border-radius-md;
  font-size: $font-size-base;
}

.remark-count {
  display: block;
  text-align: right;
  font-size: $font-size-xs;
  color: $text-tertiary;
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
  background: $bg-primary;
  box-shadow: 0 -2rpx 10rpx rgba(0, 0, 0, 0.05);
}

.total {
  display: flex;
  align-items: baseline;
}

.total-label {
  font-size: $font-size-base;
  color: $text-secondary;
}

.total-price {
  font-size: $font-size-xxl;
  font-weight: 600;
  color: $primary-color;
  margin-left: $spacing-xs;
}

.btn-primary {
  flex: 1;
  background: $primary-color;
  color: #fff;
  border-radius: $border-radius-lg;
  padding: $spacing-md;
  text-align: center;
  font-size: $font-size-base;
  font-weight: 500;
  margin-left: $spacing-lg;
  
  &.disabled {
    background: #ccc;
  }
}

.picker-trigger {
  display: none;
}
</style>
