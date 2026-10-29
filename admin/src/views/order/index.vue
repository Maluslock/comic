<script setup lang="ts">
import { h, onMounted, reactive, ref } from 'vue';
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import { NButton, NSpace, NTag } from 'naive-ui';
import {
  fetchAdminOrders,
  fetchDisputeDetail,
  fetchDisputes,
  updateAdminOrderStatus,
  exportAdminCsv
} from '@/service/api/admin';
import { getAdminApiErrorMessage } from '@/service/request/admin';

const loading = ref(false);
const exporting = ref(false);
const list = ref<Api.Admin.Order[]>([]);

async function handleExport() {
  exporting.value = true;
  try {
    await exportAdminCsv('orders', statusTab.value === 'all' ? undefined : { status: statusTab.value });
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    exporting.value = false;
  }
}

const statusTab = ref<'all' | Api.Admin.OrderStatus>('all');

const pagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  showQuickJumper: true,
  prefix: ({ itemCount }) => `共 ${itemCount} 条`
});

const tabs = [
  { key: 'all', label: '全部' },
  { key: 'pending', label: '待确认' },
  { key: 'confirmed', label: '已确认' },
  { key: 'completed', label: '已完成' },
  { key: 'cancelled', label: '已取消' }
];

const STATUS_NAMES: Record<Api.Admin.OrderStatus, string> = {
  pending: '待确认',
  confirmed: '已确认',
  completed: '已完成',
  cancelled: '已取消'
};

const STATUS_TAG_TYPES: Record<Api.Admin.OrderStatus, 'default' | 'warning' | 'info' | 'success' | 'error'> = {
  pending: 'warning',
  confirmed: 'info',
  completed: 'success',
  cancelled: 'default'
};

const SUB_STATUS_NAMES: Record<string, string> = {
  in_service: '服务中',
  delivered: '已交付',
  disputed: '争议中'
};

const DISPUTE_STATUS_NAMES: Record<string, string> = {
  open: '待响应',
  responding: '响应中',
  escalated: '已升级',
  resolved: '已裁决',
  closed: '已关闭'
};

const columns: DataTableColumns<Api.Admin.Order> = [
  {
    title: '订单ID',
    key: 'id',
    width: 90
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render(row) {
      return h(
        NTag,
        { size: 'small', type: STATUS_TAG_TYPES[row.status], round: true, bordered: false },
        { default: () => STATUS_NAMES[row.status] }
      );
    }
  },
  {
    title: '担保节点',
    key: 'subStatus',
    width: 100,
    render(row) {
      if (!row.subStatus) {
        return h('span', { class: 'text-13px text-[var(--n-text-color-3)]' }, '—');
      }

      return h(
        NTag,
        { size: 'small', type: row.subStatus === 'disputed' ? 'error' : 'info', round: true, bordered: false },
        { default: () => SUB_STATUS_NAMES[row.subStatus] ?? row.subStatus }
      );
    }
  },
  {
    title: '争议',
    key: 'disputeStatus',
    width: 100,
    render(row) {
      if (!row.disputeStatus) {
        return h('span', { class: 'text-13px text-[var(--n-text-color-3)]' }, '—');
      }

      return h(
        NTag,
        { size: 'small', type: 'warning', round: true, bordered: false },
        { default: () => DISPUTE_STATUS_NAMES[row.disputeStatus] ?? row.disputeStatus }
      );
    }
  },
  {
    title: '摄影师',
    key: 'photographerName',
    minWidth: 120
  },
  {
    title: '用户',
    key: 'coserName',
    minWidth: 120
  },
  {
    title: '拍摄日期',
    key: 'date',
    width: 120
  },
  {
    title: '时间',
    key: 'time',
    width: 80
  },
  {
    title: '价格',
    key: 'price',
    width: 100,
    render(row) {
      return h('span', { class: 'font-600 text-amber-500' }, `¥${row.price}`);
    }
  },
  {
    title: '创建时间',
    key: 'createdAt',
    width: 170,
    render(row) {
      return h('span', null, formatTime(row.createdAt));
    }
  },
  {
    title: '操作',
    key: 'actions',
    width: 240,
    fixed: 'right',
    render(row) {
      const isPending = row.status === 'pending';
      const isConfirmed = row.status === 'confirmed';

      const actions = [
        h(
          NButton,
          { size: 'small', type: 'primary', secondary: true, onClick: () => openDetail(row) },
          { default: () => '详情' }
        )
      ];

      if (row.disputeStatus) {
        actions.push(
          h(
            NButton,
            { size: 'small', type: 'warning', secondary: true, onClick: () => openDetail(row) },
            { default: () => '争议' }
          )
        );
      }

      if (isPending) {
        actions.push(
          h(
            NButton,
            {
              size: 'small',
              type: 'primary',
              secondary: true,
              onClick: () => handleUpdateStatus(row, 'confirmed')
            },
            { default: () => '确认' }
          )
        );
      } else if (isConfirmed) {
        actions.push(
          h(
            NButton,
            {
              size: 'small',
              type: 'primary',
              secondary: true,
              onClick: () => handleUpdateStatus(row, 'completed')
            },
            { default: () => '完成' }
          )
        );
      }

      if (isPending || isConfirmed) {
        actions.push(
          h(
            NButton,
            {
              size: 'small',
              type: 'error',
              secondary: true,
              onClick: () => handleUpdateStatus(row, 'cancelled')
            },
            { default: () => '取消' }
          )
        );
      }

      return h(NSpace, null, { default: () => actions });
    }
  }
];

function formatTime(value: string) {
  if (!value) {
    return '—';
  }

  return value.replace('T', ' ').slice(0, 16);
}

async function loadList() {
  loading.value = true;

  try {
    const { list: rows, total } = await fetchAdminOrders({
      status: statusTab.value === 'all' ? undefined : statusTab.value,
      page: pagination.page,
      pageSize: pagination.pageSize
    });

    list.value = rows;
    pagination.itemCount = total;
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    loading.value = false;
  }
}

async function handleUpdateStatus(row: Api.Admin.Order, status: Api.Admin.OrderStatus) {
  try {
    await updateAdminOrderStatus(row.id, status);

    window.$message?.success(
      `订单 #${row.id} 已${status === 'cancelled' ? '取消' : status === 'confirmed' ? '确认' : '完成'}`
    );

    await loadList();
  } catch (error) {
    const message = getAdminApiErrorMessage(error);

    window.$message?.error(message === 'invalid transition' ? '无效的状态流转' : message);
  }
}

function handleTabChange() {
  pagination.page = 1;
  loadList();
}

function handlePageChange(page: number) {
  pagination.page = page;
  loadList();
}

function handlePageSizeChange(pageSize: number) {
  pagination.pageSize = pageSize;
  pagination.page = 1;
  loadList();
}

// ---------- order detail drawer (with escrow dispute block) ----------
const drawerShow = ref(false);
const drawerLoading = ref(false);
const detailOrder = ref<Api.Admin.Order | null>(null);
const disputeList = ref<Api.Admin.Dispute[]>([]);
const disputeDetail = ref<Api.Admin.DisputeDetail | null>(null);

async function openDetail(row: Api.Admin.Order) {
  drawerShow.value = true;
  drawerLoading.value = true;
  detailOrder.value = row;
  disputeList.value = [];
  disputeDetail.value = null;

  try {
    const { list } = await fetchDisputes({ status: '', page: 1, pageSize: 50 });

    disputeList.value = list.filter(item => item.bookingId === row.id);
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    drawerLoading.value = false;
  }
}

async function openDisputeDetail(id: number) {
  drawerLoading.value = true;

  try {
    disputeDetail.value = await fetchDisputeDetail(id);
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    drawerLoading.value = false;
  }
}

onMounted(loadList);
</script>

<template>
  <div>
    <NCard :bordered="false" class="card-wrapper" title="订单管理">
      <template #header-extra>
        <NButton size="small" secondary :loading="exporting" @click="handleExport">导出 CSV</NButton>
      </template>
      <NTabs v-model:value="statusTab" type="line" animated @update:value="handleTabChange">
        <NTab v-for="tab in tabs" :key="tab.key" :name="tab.key">
          {{ tab.label }}
        </NTab>
      </NTabs>
      <NDataTable
        remote
        :columns="columns"
        :data="list"
        :loading="loading"
        :bordered="false"
        :single-line="false"
        :scroll-x="1400"
        :pagination="pagination"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </NCard>

    <NDrawer v-model:show="drawerShow" width="720">
      <NDrawerContent title="订单详情" closable>
        <NSpin :show="drawerLoading">
          <template v-if="detailOrder">
            <NDescriptions :column="2" bordered size="small" class="mb-20px">
              <NDescriptionsItem label="订单ID">{{ detailOrder.id }}</NDescriptionsItem>
              <NDescriptionsItem label="订单状态">
                <NTag size="small" :type="STATUS_TAG_TYPES[detailOrder.status]" round :bordered="false">
                  {{ STATUS_NAMES[detailOrder.status] }}
                </NTag>
              </NDescriptionsItem>
              <NDescriptionsItem label="担保节点">
                {{ SUB_STATUS_NAMES[detailOrder.subStatus] ?? '—' }}
              </NDescriptionsItem>
              <NDescriptionsItem label="争议状态">
                {{
                  detailOrder.disputeStatus
                    ? (DISPUTE_STATUS_NAMES[detailOrder.disputeStatus] ?? detailOrder.disputeStatus)
                    : '—'
                }}
              </NDescriptionsItem>
              <NDescriptionsItem label="摄影师">{{ detailOrder.photographerName || '—' }}</NDescriptionsItem>
              <NDescriptionsItem label="用户">{{ detailOrder.coserName || '—' }}</NDescriptionsItem>
              <NDescriptionsItem label="拍摄日期">{{ detailOrder.date }}</NDescriptionsItem>
              <NDescriptionsItem label="时间">{{ detailOrder.time }}</NDescriptionsItem>
              <NDescriptionsItem label="价格">¥{{ detailOrder.price }}</NDescriptionsItem>
              <NDescriptionsItem label="创建时间">{{ formatTime(detailOrder.createdAt) }}</NDescriptionsItem>
            </NDescriptions>

            <div class="text-14px font-600 mb-8px">争议</div>
            <div v-if="disputeList.length === 0" class="text-13px text-[var(--n-text-color-3)] mb-20px">
              该订单暂无争议记录
            </div>
            <div v-else class="mb-20px">
              <NCard
                v-for="item in disputeList"
                :key="item.id"
                size="small"
                :title="`争议 #${item.id} · ${item.openedByRole === 'coser' ? 'Coser' : '摄影师'}发起`"
                class="mb-12px"
              >
                <NDescriptions :column="2" size="small">
                  <NDescriptionsItem label="类型">
                    {{
                      { no_show: '未到场', not_delivered: '未交付', quality: '质量问题', other: '其他' }[item.type] ??
                      item.type
                    }}
                  </NDescriptionsItem>
                  <NDescriptionsItem label="状态">
                    <NTag size="small" type="warning" round :bordered="false">
                      {{ DISPUTE_STATUS_NAMES[item.status] ?? item.status }}
                    </NTag>
                  </NDescriptionsItem>
                  <NDescriptionsItem label="发起时间">{{ formatTime(item.createdAt) }}</NDescriptionsItem>
                  <NDescriptionsItem label="响应截止">{{ formatTime(item.respondDeadline) }}</NDescriptionsItem>
                  <NDescriptionsItem label="理由" :span="2">{{ item.reason || '—' }}</NDescriptionsItem>
                  <NDescriptionsItem v-if="item.resolution" label="裁决结果" :span="2">
                    {{ item.resolution }}
                  </NDescriptionsItem>
                </NDescriptions>

                <NImageGroup v-if="item.evidence && item.evidence.length > 0" class="mt-8px">
                  <NImage
                    v-for="(image, index) in item.evidence"
                    :key="index"
                    :src="image"
                    :width="100"
                    :height="100"
                    object-fit="cover"
                    class="mr-8px mb-8px"
                  />
                </NImageGroup>

                <div class="mt-8px">
                  <NButton size="small" secondary @click="openDisputeDetail(item.id)">查看争议详情</NButton>
                </div>
              </NCard>
            </div>

            <template v-if="disputeDetail">
              <div class="text-14px font-600 mb-8px">争议详情 #{{ disputeDetail.id }}</div>
              <NDescriptions :column="2" bordered size="small" class="mb-12px">
                <NDescriptionsItem label="申请方">
                  {{ disputeDetail.openedByRole === 'coser' ? 'Coser' : '摄影师' }}
                </NDescriptionsItem>
                <NDescriptionsItem label="状态">
                  {{ DISPUTE_STATUS_NAMES[disputeDetail.status] ?? disputeDetail.status }}
                </NDescriptionsItem>
                <NDescriptionsItem label="举证截止">{{ formatTime(disputeDetail.evidenceDeadline) }}</NDescriptionsItem>
                <NDescriptionsItem label="裁决时间">{{ formatTime(disputeDetail.reviewedAt ?? '') }}</NDescriptionsItem>
              </NDescriptions>
              <NTimeline v-if="disputeDetail.events && disputeDetail.events.length > 0">
                <NTimelineItem
                  v-for="event in disputeDetail.events"
                  :key="event.id"
                  :type="event.actorRole === 'admin' ? 'info' : 'default'"
                  :title="event.eventType"
                >
                  <div class="text-12px text-[var(--n-text-color-3)]">
                    {{ formatTime(event.createdAt) }} · {{ event.actorRole || 'system' }}
                  </div>
                </NTimelineItem>
              </NTimeline>
            </template>
          </template>
          <NEmpty v-else-if="!drawerLoading" description="加载失败" />
        </NSpin>
      </NDrawerContent>
    </NDrawer>
  </div>
</template>

<style scoped></style>
