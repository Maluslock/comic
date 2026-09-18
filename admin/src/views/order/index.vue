<script setup lang="ts">
import { h, onMounted, reactive, ref } from 'vue';
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import { NButton, NSpace, NTag } from 'naive-ui';
import { fetchAdminOrders, updateAdminOrderStatus, exportAdminCsv } from '@/service/api/admin';
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
    width: 190,
    fixed: 'right',
    render(row) {
      const isPending = row.status === 'pending';
      const isConfirmed = row.status === 'confirmed';
      const disabled = !isPending && !isConfirmed;

      if (disabled) {
        return h('span', { class: 'text-13px text-[var(--n-text-color-3)]' }, '—');
      }

      return h(NSpace, null, {
        default: () => [
          isPending
            ? h(
                NButton,
                {
                  size: 'small',
                  type: 'primary',
                  secondary: true,
                  onClick: () => handleUpdateStatus(row, 'confirmed')
                },
                { default: () => '确认' }
              )
            : h(
                NButton,
                {
                  size: 'small',
                  type: 'primary',
                  secondary: true,
                  onClick: () => handleUpdateStatus(row, 'completed')
                },
                { default: () => '完成' }
              ),
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
        ]
      });
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

    window.$message?.success(`订单 #${row.id} 已${status === 'cancelled' ? '取消' : status === 'confirmed' ? '确认' : '完成'}`);

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

onMounted(loadList);
</script>

<template>
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
      :scroll-x="1100"
      :pagination="pagination"
      @update:page="handlePageChange"
      @update:page-size="handlePageSizeChange"
    />
  </NCard>
</template>

<style scoped></style>
