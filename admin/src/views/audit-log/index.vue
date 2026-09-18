<script setup lang="ts">
import { h, onMounted, reactive, ref } from 'vue';
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import { NTag } from 'naive-ui';
import { fetchAuditLogs } from '@/service/api/admin';
import { getAdminApiErrorMessage } from '@/service/request/admin';

const loading = ref(false);
const list = ref<Api.Admin.AuditLog[]>([]);

const pagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [20, 50, 100],
  prefix: ({ itemCount }) => `共 ${itemCount} 条`
});

const METHOD_TYPES: Record<string, 'default' | 'info' | 'success' | 'warning' | 'error'> = {
  POST: 'success',
  PUT: 'warning',
  DELETE: 'error'
};

const columns: DataTableColumns<Api.Admin.AuditLog> = [
  { title: 'ID', key: 'id', width: 80 },
  {
    title: '管理员',
    key: 'adminName',
    width: 130,
    render(row) {
      return row.adminName || `#${row.adminId}`;
    }
  },
  {
    title: '方法',
    key: 'method',
    width: 90,
    render(row) {
      return h(NTag, { size: 'small', type: METHOD_TYPES[row.method] ?? 'default', bordered: false }, { default: () => row.method });
    }
  },
  { title: '路径', key: 'path', minWidth: 320 },
  {
    title: '状态',
    key: 'status',
    width: 90,
    render(row) {
      return h(NTag, { size: 'small', type: row.status < 400 ? 'success' : 'error', bordered: false }, { default: () => String(row.status) });
    }
  },
  { title: '时间', key: 'createdAt', width: 190 }
];

async function loadList() {
  loading.value = true;

  try {
    const res = await fetchAuditLogs({ page: pagination.page, pageSize: pagination.pageSize });

    list.value = res.list ?? [];
    pagination.itemCount = res.total ?? 0;
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    loading.value = false;
  }
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
  <NCard :bordered="false" class="card-wrapper" title="操作日志">
    <NDataTable
      remote
      :columns="columns"
      :data="list"
      :loading="loading"
      :bordered="false"
      :single-line="false"
      :scroll-x="1000"
      :pagination="pagination"
      @update:page="handlePageChange"
      @update:page-size="handlePageSizeChange"
    />
  </NCard>
</template>

<style scoped></style>
