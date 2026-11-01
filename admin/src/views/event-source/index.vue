<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue';
import type { DataTableColumns } from 'naive-ui';
import { NButton, NTag, NTooltip } from 'naive-ui';
import type { SelectOption } from 'naive-ui';
import { fetchEventSources } from '@/service/api/admin';
import { getAdminApiErrorMessage } from '@/service/request/admin';

/**
 * admin「数据源状态」页。
 *
 * 一屏回答四个问题：
 *  1. 每个源在库里有多少条（totalItems / activeItems）
 *  2. 每个源最近一次跑成什么样（lastStatus / lastRun / lastItems）
 *  3. 最近一次抓到了多少（lastItems）
 *  4. 近 7 天成功率如何（successRate7d / runCount7d）
 * 下面是运行历史表格：谁、什么时候、结果、抓到/新增/更新/错误、耗时、说明。
 *
 * 数据完全来自后端一次 GET /event-sources：所有已知来源都由后端补齐，
 * 前端不自己拼零值，只负责渲染 + 本地按源过滤。
 */

const loading = ref(false);
const overview = ref<Api.Admin.AdminEventSourceOverview>({
  sources: [],
  runs: [],
  totals: {},
  generatedAt: ''
});

/** 表格筛选用的源，'' 表示全部 */
const sourceFilter = ref<string>('');

const SOURCE_NAMES: Record<string, string> = {
  allcpp: 'cpp 无差别同人站',
  bilibili: '哔哩哔哩会员购',
  nyato: '喵特',
  manual: '人工录入'
};

const STATUS_NAMES: Record<Api.Admin.EventSourceRunStatus, string> = {
  running: '运行中',
  ok: '成功',
  partial: '部分成功',
  failed: '失败',
  empty: '无新数据'
};

const STATUS_TAG_TYPES: Record<Api.Admin.EventSourceRunStatus, 'default' | 'info' | 'warning' | 'success' | 'error'> = {
  running: 'info',
  ok: 'success',
  partial: 'warning',
  failed: 'error',
  empty: 'default'
};

function sourceLabel(source: string) {
  return SOURCE_NAMES[source] ?? source;
}

function statusLabel(status: string) {
  if (!status) {
    return '未运行';
  }

  return STATUS_NAMES[status as Api.Admin.EventSourceRunStatus] ?? status;
}

function statusTagType(status: string): 'default' | 'info' | 'warning' | 'success' | 'error' {
  if (!status) {
    return 'default';
  }

  return STATUS_TAG_TYPES[status as Api.Admin.EventSourceRunStatus] ?? 'default';
}

function formatTime(value: string | null | undefined) {
  if (!value) {
    return '—';
  }

  return value.replace('T', ' ').slice(0, 19);
}

/** 后端给的是 0~1 的比率，这里显示成百分比；没跑过（分母 0）显示「—」 */
function formatRate(item: Api.Admin.AdminEventSourceStatus) {
  if (!item.runCount7d) {
    return '—';
  }

  return `${(item.successRate7d * 100).toFixed(1)}%`;
}

function formatDuration(ms: number | null | undefined) {
  if (ms === null || ms === undefined) {
    return '—';
  }

  if (ms < 1000) {
    return `${ms} ms`;
  }

  return `${(ms / 1000).toFixed(1)} s`;
}

/** 顶部四张卡：按后端返回的源顺序铺开，没跑过的源也在（lastStatus 为空串） */
const cards = computed(() => overview.value.sources);

const sourceOptions = computed<SelectOption[]>(() => [
  { label: '全部来源', value: '' },
  ...overview.value.sources.map(item => ({
    label: sourceLabel(item.source),
    value: item.source
  }))
]);

const runs = computed(() => {
  const all = overview.value.runs;

  if (!sourceFilter.value) {
    return all;
  }

  return all.filter(run => run.source === sourceFilter.value);
});

const columns: DataTableColumns<Api.Admin.AdminEventSourceRun> = [
  {
    title: '源',
    key: 'source',
    width: 170,
    render(row) {
      return h('div', null, [
        h('div', null, sourceLabel(row.source)),
        h('div', { class: 'text-12px text-[var(--n-text-color-3)]' }, row.source)
      ]);
    }
  },
  {
    title: '开始时间',
    key: 'startedAt',
    width: 190,
    render(row) {
      return h('span', null, formatTime(row.startedAt));
    }
  },
  {
    title: '状态',
    key: 'status',
    width: 110,
    render(row) {
      return h(
        NTag,
        { size: 'small', type: statusTagType(row.status), round: true, bordered: false },
        { default: () => statusLabel(row.status) }
      );
    }
  },
  {
    title: '抓到',
    key: 'items',
    width: 90,
    align: 'right',
    render(row) {
      return h('span', { class: 'tabular-nums' }, String(row.items ?? 0));
    }
  },
  {
    title: '新增',
    key: 'inserted',
    width: 90,
    align: 'right',
    render(row) {
      return h('span', { class: 'tabular-nums text-[var(--n-color-target)]' }, String(row.inserted ?? 0));
    }
  },
  {
    title: '更新',
    key: 'updated',
    width: 90,
    align: 'right',
    render(row) {
      return h('span', { class: 'tabular-nums' }, String(row.updated ?? 0));
    }
  },
  {
    title: '错误',
    key: 'errors',
    width: 90,
    align: 'right',
    render(row) {
      const count = row.errors ?? 0;

      return h(
        'span',
        { class: count > 0 ? 'tabular-nums text-[var(--n-color-error)]' : 'tabular-nums' },
        String(count)
      );
    }
  },
  {
    title: '耗时',
    key: 'durationMs',
    width: 110,
    align: 'right',
    render(row) {
      return h('span', { class: 'tabular-nums' }, formatDuration(row.durationMs));
    }
  },
  {
    title: '说明',
    key: 'message',
    minWidth: 240,
    ellipsis: { tooltip: true },
    render(row) {
      return h('span', null, row.message || '—');
    }
  }
];

async function load() {
  loading.value = true;

  try {
    overview.value = await fetchEventSources();
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    loading.value = false;
  }
}

onMounted(load);
</script>

<template>
  <div>
    <NSpace vertical :size="16">
      <div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-16px">
        <NCard v-for="card in cards" :key="card.source" :bordered="false" class="card-wrapper">
          <div class="flex-y-center justify-between">
            <div>
              <div class="text-14px text-[var(--n-text-color-3)]">{{ sourceLabel(card.source) }}</div>
              <div class="mt-8px text-30px font-bold tabular-nums">
                {{ card.totalItems }}
                <span class="text-13px font-normal text-[var(--n-text-color-3)]">条在库</span>
              </div>
            </div>
            <NTag :type="statusTagType(card.lastStatus)" size="small" round :bordered="false">
              {{ statusLabel(card.lastStatus) }}
            </NTag>
          </div>
          <div class="mt-12px text-12px text-[var(--n-text-color-3)] leading-20px">
            <div>上次抓取：{{ card.lastItems }} 条 · {{ formatTime(card.lastRun) }}</div>
            <div>近 7 天成功率：{{ formatRate(card) }}（{{ card.runCount7d }} 轮）</div>
            <div>已下架：{{ card.totalItems - card.activeItems }} 条</div>
          </div>
        </NCard>
      </div>

      <NCard :bordered="false" class="card-wrapper" title="运行历史">
        <template #header-extra>
          <NSpace align="center" :size="8">
            <NSelect
              v-model:value="sourceFilter"
              :options="sourceOptions"
              size="small"
              class="w-160px"
              placeholder="按源筛选"
            />
            <span class="text-12px text-[var(--n-text-color-3)]">
              更新于 {{ formatTime(overview.generatedAt) }}
            </span>
            <NButton size="small" :loading="loading" @click="load">刷新</NButton>
          </NSpace>
        </template>
        <NDataTable
          :columns="columns"
          :data="runs"
          :loading="loading"
          :bordered="false"
          :single-line="false"
          :scroll-x="1200"
          :pagination="{ pageSize: 20 }"
        />
      </NCard>
    </NSpace>
  </div>
</template>

<style scoped></style>
