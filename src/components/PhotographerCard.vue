<template>
  <view
    class="photographer-card"
    :class="`variant-${variant}`"
    hover-class="photographer-card--pressed"
    :hover-stay-time="60"
    @click="goDetail"
  >
    <image v-if="showAvatar" :src="photographer.avatar" class="avatar" mode="aspectFill" />
    <view v-else class="avatar avatar-fallback"><text class="avatar-initial">{{ initial }}</text></view>
    <view class="info">
      <view class="header">
        <text class="name">{{ photographer.name }}</text>
        <view class="rating">
          <template v-if="hasRating">
            <text class="star">★</text>
            <text class="score">{{ photographer.rating }}</text>
          </template>
          <text v-else class="no-rating">暂无评分</text>
        </view>
      </view>
      <text v-if="photographer.description" class="desc">{{ photographer.description }}</text>
      <view class="tags" v-if="photographer.tags.length">
        <MTag
          v-for="tag in photographer.tags.slice(0, variant === 'grid' ? 2 : 3)"
          :key="tag"
          variant="neutral"
        >{{ tag }}</MTag>
      </view>
      <view class="footer">
        <view class="footer-left">
          <MPriceTag v-if="price.text" :text="price.text" :tone="price.tone" size="md" />
          <text v-if="!price.text" class="muted">暂未设置</text>
        </view>
        <view class="footer-right">
          <text class="meta">{{ photographer.location }}</text>
          <text v-if="variant === 'row'" class="meta">{{ photographer.orderCount }}单已接</text>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Photographer } from '@/types'
import { describePrice } from '@/utils/price'
import MPriceTag from '@/components/MPriceTag.vue'
import MTag from '@/components/MTag.vue'

const props = withDefaults(defineProps<{
  photographer: Photographer
  /** row：列表横卡（默认）；grid：首页双列竖卡 */
  variant?: 'row' | 'grid'
}>(), {
  variant: 'row'
})

const initial = computed(() => (props.photographer.name || '?').slice(0, 1))
const showAvatar = computed(() => !!props.photographer.avatar && /\.(jpe?g|png|webp)$/i.test(props.photographer.avatar))

// 评分按真实评价重算：没有评价时后端给 rating=0 / reviewCount=0，
// 显示「★ 0」会让人以为被打 0 分，故无评价时改文案。
const hasRating = computed(() => (props.photographer.reviewCount ?? 0) > 0)

// 价格取最低可成交价；接口没返回汇总时 describePrice 给空串，这里不渲染。
const price = computed(() => describePrice(props.photographer))

function goDetail() {
  uni.navigateTo({
    url: `/pages/photographer/detail?id=${props.photographer.id}`
  })
}
</script>

<style lang="scss" scoped>
.photographer-card {
  display: flex;
  background: $surface-1;
  border: 1rpx solid $border-subtle;
  border-radius: $r-lg;
  padding: $s-sm;
  transition: transform $motion-fast, background $motion-fast;
}

.photographer-card--pressed {
  background: $surface-2;
  transform: scale(0.985);
}

.avatar.avatar-fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, rgba(255, 107, 0, 0.22), rgba(255, 162, 77, 0.08));
  border-color: rgba(255, 107, 0, 0.35);
}

.avatar-initial {
  font-size: 40rpx;
  font-weight: 900;
  color: #FFB27A;
  line-height: 1;
}

.avatar {
  width: 104rpx;
  height: 104rpx;
  border-radius: $r-full;
  flex-shrink: 0;
  border: 2rpx solid rgba(255, 255, 255, 0.1);
  background: $surface-2;
}

.info {
  flex: 1;
  margin-left: $s-sm;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: $s-xs;
}

.name {
  font-size: $t-xl;
  font-weight: 600;
  color: $text-1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.rating {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.star {
  color: $state-warning;
  font-size: $t-sm;
}

.score {
  color: $state-warning;
  font-size: $t-xs;
  margin-left: 2rpx;
}

.no-rating {
  color: $text-3;
  font-size: $t-xs;
}

.desc {
  font-size: $t-sm;
  color: $text-2;
  margin-top: 6rpx;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 1;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.tags {
  display: flex;
  flex-wrap: wrap;
  gap: $s-2xs;
  margin-top: $s-xs;
}

.footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: $s-xs;
  margin-top: auto;
  padding-top: $s-xs;
}

.footer-left,
.footer-right {
  display: flex;
  align-items: center;
  gap: $s-xs;
  min-width: 0;
}

.meta {
  font-size: $t-xs;
  color: $text-3;
  white-space: nowrap;
}

.muted {
  font-size: $t-sm;
  color: $text-3;
  white-space: nowrap;
}

/* 首页 / 详情页双列竖卡 */
.variant-grid {
  flex-direction: column;
  align-items: flex-start;
  padding: $s-md;
  gap: $s-xs;
}

.variant-grid .avatar {
  width: 96rpx;
  height: 96rpx;
}

.variant-grid .info {
  margin-left: 0;
  margin-top: 0;
  width: 100%;
}

.variant-grid .header {
  width: 100%;
}

.variant-grid .desc {
  -webkit-line-clamp: 2;
}

.variant-grid .tags {
  min-height: 36rpx;
}

.variant-grid .footer {
  width: 100%;
  flex-direction: column;
  align-items: flex-start;
  gap: 2rpx;
}

.variant-grid .footer-left,
.variant-grid .footer-right {
  width: 100%;
  justify-content: flex-start;
}

.variant-grid .footer-right {
  gap: $s-2xs;
}
</style>
