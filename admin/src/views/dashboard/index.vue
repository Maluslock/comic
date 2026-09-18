<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useEcharts } from '@/hooks/common/echarts';
import { fetchAdminDashboard } from '@/service/api/admin';
import { getAdminApiErrorMessage } from '@/service/request/admin';

const loading = ref(true);

const totals = reactive<Api.Admin.DashboardTotals>({
  users: 0,
  photographers: 0,
  certified: 0,
  orders: 0,
  pendingOrders: 0,
  events: 0
});

const lastActiveUsers = ref(0);
const lastNewUsers = ref(0);

const cards: { key: CardKey; label: string; icon: string; color: string; subtitle: () => string }[] = [
  {
    key: 'users',
    label: '用户总量',
    icon: 'mdi:account-group',
    color: '#6366f1',
    subtitle: () => `今日新增 +${lastNewUsers.value}`
  },
  {
    key: 'newUsers',
    label: '今日新增用户',
    icon: 'mdi:account-plus',
    color: '#06b6d4',
    subtitle: () => '注册趋势见下方折线图'
  },
  {
    key: 'activeUsers',
    label: '日活用户',
    icon: 'mdi:account-heart',
    color: '#10b981',
    subtitle: () => `今日活跃 ${lastActiveUsers.value} 人`
  },
  {
    key: 'photographers',
    label: '摄影师',
    icon: 'mdi:camera',
    color: '#a855f7',
    subtitle: () => `已认证 ${totals.certified} 人`
  },
  {
    key: 'orders',
    label: '订单总量',
    icon: 'mdi:clipboard-text',
    color: '#f59e0b',
    subtitle: () => `待确认 ${totals.pendingOrders} 单`
  },
  {
    key: 'events',
    label: '漫展数量',
    icon: 'mdi:calendar-star',
    color: '#ef4444',
    subtitle: () => '全站漫展活动'
  }
];

type CardKey = 'users' | 'newUsers' | 'activeUsers' | 'photographers' | 'orders' | 'events';

const cardValues = computed<Record<CardKey, number>>(() => ({
  users: totals.users,
  newUsers: lastNewUsers.value,
  activeUsers: lastActiveUsers.value,
  photographers: totals.photographers,
  orders: totals.orders,
  events: totals.events
}));

const { domRef: lineRef, updateOptions: updateLine } = useEcharts(() => ({
  tooltip: {
    trigger: 'axis',
    axisPointer: {
      type: 'cross',
      label: {
        backgroundColor: '#6a7985'
      }
    }
  },
  legend: {
    data: ['新增用户', '活跃用户'],
    top: '0'
  },
  grid: {
    left: '3%',
    right: '4%',
    bottom: '3%',
    top: '15%'
  },
  xAxis: {
    type: 'category',
    boundaryGap: false,
    data: [] as string[]
  },
  yAxis: {
    type: 'value'
  },
  series: [
    {
      color: '#6366f1',
      name: '新增用户',
      type: 'line',
      smooth: true,
      areaStyle: {
        color: {
          type: 'linear',
          x: 0,
          y: 0,
          x2: 0,
          y2: 1,
          colorStops: [
            {
              offset: 0.25,
              color: '#6366f1'
            },
            {
              offset: 1,
              color: '#fff'
            }
          ]
        }
      },
      emphasis: {
        focus: 'series'
      },
      data: [] as number[]
    },
    {
      color: '#06b6d4',
      name: '活跃用户',
      type: 'line',
      smooth: true,
      areaStyle: {
        color: {
          type: 'linear',
          x: 0,
          y: 0,
          x2: 0,
          y2: 1,
          colorStops: [
            {
              offset: 0.25,
              color: '#06b6d4'
            },
            {
              offset: 1,
              color: '#fff'
            }
          ]
        }
      },
      emphasis: {
        focus: 'series'
      },
      data: [] as number[]
    }
  ]
}));

const { domRef: pieRef, updateOptions: updatePie } = useEcharts(() => ({
  tooltip: {
    trigger: 'item'
  },
  legend: {
    bottom: '0'
  },
  series: [
    {
      name: '订单状态',
      type: 'pie',
      radius: ['45%', '70%'],
      avoidLabelOverlap: false,
      itemStyle: {
        borderRadius: 6,
        borderColor: '#fff',
        borderWidth: 2
      },
      label: {
        show: true,
        formatter: '{b}: {c}'
      },
      data: [] as { name: string; value: number; itemStyle: { color: string } }[]
    }
  ]
}));

const { domRef: barRef, updateOptions: updateBar } = useEcharts(() => ({
  tooltip: {
    trigger: 'axis',
    axisPointer: {
      type: 'shadow'
    }
  },
  grid: {
    left: '3%',
    right: '4%',
    bottom: '3%',
    top: '15%'
  },
  xAxis: {
    type: 'category',
    data: [] as string[]
  },
  yAxis: {
    type: 'value'
  },
  series: [
    {
      name: '订单量',
      type: 'bar',
      barMaxWidth: 32,
      itemStyle: {
        borderRadius: [4, 4, 0, 0],
        color: '#6366f1'
      },
      data: [] as number[]
    }
  ]
}));

const STATUS_COLORS: Record<keyof Api.Admin.OrdersByStatus, string> = {
  pending: '#f59e0b',
  confirmed: '#06b6d4',
  completed: '#10b981',
  cancelled: '#ef4444'
};

const STATUS_NAMES: Record<keyof Api.Admin.OrdersByStatus, string> = {
  pending: '待确认',
  confirmed: '已确认',
  completed: '已完成',
  cancelled: '已取消'
};

async function loadDashboard() {
  loading.value = true;

  try {
    const data = await fetchAdminDashboard();

    Object.assign(totals, data.totals);

    lastNewUsers.value = data.trends.newUsers.at(-1) ?? 0;
    lastActiveUsers.value = data.trends.activeUsers.at(-1) ?? 0;

    updateLine(opts => {
      opts.xAxis.data = data.trends.date;
      opts.series[0].data = data.trends.newUsers;
      opts.series[1].data = data.trends.activeUsers;

      return opts;
    });

    updatePie(opts => {
      opts.series[0].data = (Object.keys(data.ordersByStatus) as (keyof Api.Admin.OrdersByStatus)[]).map(key => ({
        name: STATUS_NAMES[key],
        value: data.ordersByStatus[key],
        itemStyle: { color: STATUS_COLORS[key] }
      }));

      return opts;
    });

    updateBar(opts => {
      opts.xAxis.data = data.trends.date;
      opts.series[0].data = data.trends.orders;

      return opts;
    });
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    loading.value = false;
  }
}

onMounted(loadDashboard);
</script>

<template>
  <NSpin :show="loading">
    <NSpace vertical :size="16">
      <div class="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-6 gap-16px">
        <div v-for="card in cards" :key="card.key">
          <NCard :bordered="false" class="card-wrapper">
            <div class="flex-y-center justify-between">
              <div>
                <div class="text-14px text-[var(--n-text-color-3)]">{{ card.label }}</div>
                <div class="mt-8px text-30px font-bold tabular-nums">{{ cardValues[card.key] }}</div>
              </div>
              <div
                class="size-52px rd-12px flex-center text-24px text-white"
                :style="{ backgroundColor: card.color }"
              >
                <Icon :icon="card.icon" />
              </div>
            </div>
            <div class="mt-12px text-12px text-[var(--n-text-color-3)]">{{ card.subtitle() }}</div>
          </NCard>
        </div>
      </div>
      <div class="grid grid-cols-1 lg:grid-cols-12 gap-16px">
        <div class="lg:col-span-7">
          <NCard :bordered="false" class="card-wrapper" title="用户增长趋势">
            <div ref="lineRef" class="h-360px overflow-hidden"></div>
          </NCard>
        </div>
        <div class="lg:col-span-5">
          <NCard :bordered="false" class="card-wrapper" title="订单状态分布">
            <div ref="pieRef" class="h-360px overflow-hidden"></div>
          </NCard>
        </div>
      </div>
      <NCard :bordered="false" class="card-wrapper" title="每日订单量">
        <div ref="barRef" class="h-300px overflow-hidden"></div>
      </NCard>
    </NSpace>
  </NSpin>
</template>

<style scoped></style>
