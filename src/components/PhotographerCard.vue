<template>
  <view
    class="photographer-card"
    :class="`variant-${variant}`"
    hover-class="photographer-card--pressed"
    :hover-stay-time="60"
    @click="goDetail"
  >
    <image v-if="showAvatar" :src="photographer.avatar" class="avatar" mode="aspectFill" lazy-load @error="avatarFailed = true" />
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
      <!-- 描述槽位：grid 变体固定两行高度，文案缺失时用占位节点补齐 -->
      <text v-if="photographer.description" class="desc">{{ photographer.description }}</text>
      <view v-else-if="variant === 'grid'" class="desc" />
      <!-- 标签槽位：grid 变体固定一行高度，无标签时用占位节点补齐 -->
      <view class="tags" v-if="photographer.tags.length">
        <MTag
          v-for="tag in photographer.tags.slice(0, variant === 'grid' ? 2 : 3)"
          :key="tag"
          variant="neutral"
        >{{ tag }}</MTag>
      </view>
      <view v-else-if="variant === 'grid'" class="tags" />
      <view class="footer">
        <view class="footer-left">
          <MPriceTag v-if="price.text" :text="price.text" :tone="price.tone" size="md" />
          <text v-if="!price.text" class="muted">价格面议</text>
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
import { computed, ref } from 'vue'
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
// 不再用后缀白名单判定：微信头像、OSS 带参数、SVG 都会被误杀成大单字。
// 交给 <image> 自己加载，失败再回退首字（mp-weixin 不支持 svg 时也走这里）。
const avatarFailed = ref(false)
const showAvatar = computed(() => !!props.photographer.avatar?.trim() && !avatarFailed.value)

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

/* 首页 / 详情页双列竖卡
 * 等高策略：内部槽位固定 + 统一最小高度（grid 行高由内容决定，跨排不等高）
 * 槽位明细（grid 变体，1rpx = 750/视口宽）：
 *   头像 96rpx + 间距 8rpx
 *   名称行 48rpx / 描述 6rpx+2 行×33.6rpx / 标签 8rpx+36rpx / 页脚 8rpx+74rpx
 *   = 内容约 400rpx，最小高度取 420rpx 留 20rpx 余量，保证内容不被裁、卡片等高
 */
$grid-card-min-height: 360rpx;

.variant-grid {
  flex-direction: column;
  align-items: flex-start;
  padding: $s-md;
  gap: $s-xs;
  height: 100%;
  min-height: $grid-card-min-height;
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

/* 姓名：允许收缩到可用宽度，超出用省略号截断。
 * flex 子项默认 min-width:auto（= min-content），名字过长时会把 grid 轨道撑成
 * 不等宽（1fr 的最小尺寸是 auto），导致同一排两列宽度不一致、卡片错位；
 * 这里显式 min-width:0 打断 min-content 下限，名字再长也只在本卡内省略。 */
.variant-grid .name {
  min-width: 0;
}

/* 描述：grid 变体固定两行高度。
 * display:-webkit-box + -webkit-box-orient:vertical + overflow:hidden 必须三件套齐写，
 * 只写 -webkit-line-clamp 在部分内核（Chromium 155 会把 -webkit-box 归一为 flow-root）
 * 与小程序端都不生效；这里补齐三件套并给 min-height 锁住两行槽位，
 * 保证「有描述 / 无描述 / 一行描述」的卡片占位一致。 */
.variant-grid .desc {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  line-height: 1.4;
  min-height: 2.8em;
  max-height: 2.8em;
  width: 100%;
}

/* 标签：grid 变体固定一行高度，无标签时模板里渲染空的 .tags 占位节点，
 * 两种情况占位一致，卡片高度不会被内容多寡撑开；标签横向不换行、溢出隐藏。 */
.variant-grid .tags {
  width: 100%;
  min-height: 36rpx;
  max-height: 36rpx;
  flex-wrap: nowrap;
  align-items: center;
  overflow: hidden;
}

.variant-grid .tags :deep(.m-tag) {
  flex-shrink: 0;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
