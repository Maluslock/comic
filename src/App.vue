<script setup lang="ts">
import { onLaunch, onShow, onHide } from '@dcloudio/uni-app'
import { useUserStore } from '@/stores/user'
import { refreshMessageBadge } from '@/utils/badge'

const userStore = useUserStore()

onLaunch(() => {
  try {
    if (typeof uni.setLocale === 'function') uni.setLocale('zh-Hans')
  } catch (e) {
    // 部分平台不支持时忽略
  }
  userStore.init()
  console.log('App Launch')
})

onShow(() => {
  refreshMessageBadge()
})

onHide(() => {
  console.log('App Hide')
})
</script>

<style lang="scss">
@import 'uview-plus/index.scss';
@import '@/styles/global.scss';
</style>
