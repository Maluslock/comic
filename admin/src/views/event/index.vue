<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue';
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import { NTag, NSwitch } from 'naive-ui';
import { fetchAdminEvents, updateAdminEventStatus } from '@/service/api/admin';
import { getAdminApiErrorMessage } from '@/service/request/admin';

const loading = ref(false);
const allList = ref<Api.Admin.EventItem[]>([]);
const onlyActive = ref(true);

const page = ref(1);
const pageSize = ref(10);

const filteredList = computed(() => {
  const rows = onlyActive.value ? allList.value.filter(item => !item.delFlag) : allList.value;

  return [...rows].sort((a, b) => {
    if (a.delFlag !== b.delFlag) {
      return a.delFlag ? 1 : -1;
    }

    return String(b.startDate).localeCompare(String(a.startDate));
  });
});

const totalCount = computed(() => filteredList.value.length);

/** the events api has no server-side pagination, so slice the filtered list client-side */
const pageRows = computed(() => {
  const start = (page.value - 1) * pageSize.value;

  return filteredList.value.slice(start, start + pageSize.value);
});

const tablePagination = computed<PaginationProps>(() => ({
  page: page.value,
  pageSize: pageSize.value,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  itemCount: totalCount.value,
  prefix: ({ itemCount }) => `共 ${itemCount ?? 0} 条`
}));

function handlePageChange(p: number) {
  page.value = p;
}

function handlePageSizeChange(ps: number) {
  pageSize.value = ps;
  page.value = 1;
}

function handleActiveFilterChange() {
  page.value = 1;
}

function todayStr() {
  const now = new Date();

  const month = String(now.getMonth() + 1).padStart(2, '0');
  const day = String(now.getDate()).padStart(2, '0');

  return `${now.getFullYear()}-${month}-${day}`;
}

function getStatusLabel(item: Api.Admin.EventItem) {
  const today = todayStr();
  const start = String(item.startDate || '');
  const end = String(item.endDate || '');

  if (end && end < today) {
    return '已结束';
  }

  if (start && start > today) {
    return '即将开始';
  }

  return '进行中';
}

const STATUS_TAG_TYPES: Record<string, 'default' | 'success' | 'info' | 'warning'> = {
  进行中: 'success',
  即将开始: 'info',
  已结束: 'default'
};

const columns: DataTableColumns<Api.Admin.EventItem> = [
  {
    title: '名称',
    key: 'name',
    minWidth: 240
  },
  {
    title: '城市',
    key: 'location',
    width: 100
  },
  {
    title: '场馆',
    key: 'venue',
    minWidth: 120,
    render(row) {
      return h('span', null, row.venue || '—');
    }
  },
  {
    title: '开始日期',
    key: 'startDate',
    width: 120
  },
  {
    title: '结束日期',
    key: 'endDate',
    width: 120
  },
  {
    title: '类型',
    key: 'typeName',
    width: 90
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render(row) {
      const label = getStatusLabel(row);

      return h(
        NTag,
        { size: 'small', type: STATUS_TAG_TYPES[label] ?? 'default', round: true, bordered: false },
        { default: () => label }
      );
    }
  },
  {
    title: '上架',
    key: 'delFlag',
    width: 80,
    fixed: 'right',
    render(row) {
      return h(NSwitch, {
        size: 'small',
        value: !row.delFlag,
        onUpdateValue: (value: boolean) => handleToggleShelf(row, value)
      });
    }
  }
];

async function loadList() {
  loading.value = true;

  try {
    allList.value = await fetchAdminEvents();
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    loading.value = false;
  }
}

async function handleToggleShelf(row: Api.Admin.EventItem, onShelf: boolean) {
  try {
    await updateAdminEventStatus(row.id, !onShelf);

    window.$message?.success(onShelf ? '已上架' : '已下架');

    row.delFlag = !onShelf;
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  }
}

onMounted(loadList);
</script>

<template>
  <NCard :bordered="false" class="card-wrapper" title="漫展管理">
    <template #header-extra>
      <div class="flex-y-center gap-8px">
        <span class="text-13px text-[var(--n-text-color-3)]">只看已上架</span>
        <NSwitch v-model:value="onlyActive" size="small" @update:value="handleActiveFilterChange" />
      </div>
    </template>
    <NDataTable
      remote
      :columns="columns"
      :data="pageRows"
      :loading="loading"
      :bordered="false"
      :single-line="false"
      :scroll-x="1100"
      :pagination="tablePagination"
      @update:page="handlePageChange"
      @update:page-size="handlePageSizeChange"
    />
  </NCard>
</template>

<style scoped></style>
