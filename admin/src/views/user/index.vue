<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue';
import type { DataTableColumns, PaginationProps, SelectOption } from 'naive-ui';
import { NAvatar, NButton, NSpace, NTag } from 'naive-ui';
import { fetchAdminUserDetail, fetchAdminUsers, setUserStatus, exportAdminCsv } from '@/service/api/admin';
import { getAdminApiErrorMessage } from '@/service/request/admin';

const loading = ref(false);
const exporting = ref(false);
const list = ref<Api.Admin.AdminUser[]>([]);
const total = ref(0);

const searchForm = ref({
  keyword: '',
  role: 'all',
  status: 'all'
});

const page = ref(1);
const pageSize = ref(10);

async function handleExport() {
  exporting.value = true;
  try {
    const params: Record<string, string> = {};
    if (searchForm.value.keyword) params.keyword = searchForm.value.keyword;
    if (searchForm.value.role !== 'all') params.role = searchForm.value.role;
    if (searchForm.value.status !== 'all') params.status = searchForm.value.status;
    await exportAdminCsv('users', params);
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    exporting.value = false;
  }
}

const roleOptions: SelectOption[] = [
  { label: '全部', value: 'all' },
  { label: '摄影师', value: 'photographer' },
  { label: 'Coser', value: 'coser' }
];

const statusOptions: SelectOption[] = [
  { label: '全部', value: 'all' },
  { label: '启用', value: 'active' },
  { label: '禁用', value: 'disabled' }
];

const tablePagination = computed<PaginationProps>(() => ({
  page: page.value,
  pageSize: pageSize.value,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  showQuickJumper: true,
  itemCount: total.value,
  prefix: ({ itemCount }) => `共 ${itemCount ?? 0} 条`
}));

const ROLE_NAMES: Record<string, string> = {
  photographer: '摄影师',
  coser: 'Coser'
};

const columns: DataTableColumns<Api.Admin.AdminUser> = [
  {
    title: '头像',
    key: 'avatar',
    width: 72,
    render(row) {
      return h(NAvatar, {
        round: true,
        size: 40,
        src: row.avatar,
        fallbackSrc: 'https://api.dicebear.com/7.x/avataaars/svg?seed=fallback'
      });
    }
  },
  {
    title: '昵称',
    key: 'name',
    minWidth: 120,
    render(row) {
      return h('span', null, row.name || '—');
    }
  },
  {
    title: '手机号',
    key: 'phone',
    minWidth: 130
  },
  {
    title: '角色',
    key: 'role',
    width: 110,
    render(row) {
      return h(
        NTag,
        { size: 'small', type: row.role === 'photographer' ? 'warning' : 'info', round: true, bordered: false },
        { default: () => ROLE_NAMES[row.role] ?? row.role }
      );
    }
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render(row) {
      return h(
        NTag,
        { size: 'small', type: row.status === 'active' ? 'success' : 'error', round: true, bordered: false },
        { default: () => (row.status === 'active' ? '启用' : '禁用') }
      );
    }
  },
  {
    title: '注册时间',
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
      return h(NSpace, null, {
        default: () => [
          h(
            NButton,
            { size: 'small', type: 'primary', secondary: true, onClick: () => openDetail(row) },
            { default: () => '详情' }
          ),
          h(
            NButton,
            {
              size: 'small',
              type: row.status === 'active' ? 'error' : 'primary',
              secondary: true,
              onClick: () => handleToggleStatus(row)
            },
            { default: () => (row.status === 'active' ? '禁用' : '启用') }
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
    const { list: rows, total: count } = await fetchAdminUsers({
      keyword: searchForm.value.keyword.trim() || undefined,
      role: searchForm.value.role === 'all' ? undefined : searchForm.value.role,
      status: searchForm.value.status === 'all' ? undefined : searchForm.value.status,
      page: page.value,
      pageSize: pageSize.value
    });

    list.value = rows;
    total.value = count;
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    loading.value = false;
  }
}

function handleSearch() {
  page.value = 1;
  loadList();
}

function handleReset() {
  searchForm.value = { keyword: '', role: 'all', status: 'all' };
  page.value = 1;
  loadList();
}

function handlePageChange(p: number) {
  page.value = p;
  loadList();
}

function handlePageSizeChange(ps: number) {
  pageSize.value = ps;
  page.value = 1;
  loadList();
}

function handleToggleStatus(row: Api.Admin.AdminUser) {
  const next: Api.Admin.UserStatus = row.status === 'active' ? 'disabled' : 'active';
  const isDisable = next === 'disabled';

  window.$dialog?.warning({
    title: isDisable ? '禁用用户' : '启用用户',
    content: `${isDisable ? '禁用' : '启用'}用户「${row.name || row.phone}」(${row.phone})？${isDisable ? '禁用后该用户将无法登录。' : ''}`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await setUserStatus(row.id, next);

        window.$message?.success(isDisable ? '已禁用该用户' : '已启用该用户');

        await loadList();
      } catch (error) {
        window.$message?.error(getAdminApiErrorMessage(error));
      }
    }
  });
}

const drawerShow = ref(false);
const drawerLoading = ref(false);
const detail = ref<Api.Admin.AdminUserDetail | null>(null);

const statCards = computed(() => {
  const stats = detail.value?.stats;

  if (!stats) {
    return [];
  }

  return [
    { label: '预约订单', value: stats.bookingsCount },
    { label: '评价数', value: stats.reviewsCount },
    { label: '收藏数', value: stats.favoritesCount },
    { label: '关注漫展', value: stats.followsCount }
  ];
});

const BOOKING_STATUS_NAMES: Record<string, string> = {
  pending: '待确认',
  confirmed: '已确认',
  completed: '已完成',
  cancelled: '已取消'
};

const bookingColumns: DataTableColumns<Api.Admin.AdminUserBooking> = [
  { title: '订单ID', key: 'id', width: 90 },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render(row) {
      return h(
        NTag,
        { size: 'small', type: 'default', round: true, bordered: false },
        { default: () => BOOKING_STATUS_NAMES[row.status] ?? row.status }
      );
    }
  },
  { title: '拍摄日期', key: 'date', width: 120 },
  { title: '时间', key: 'time', width: 80 },
  {
    title: '价格',
    key: 'price',
    width: 100,
    render(row) {
      return h('span', { class: 'font-600 text-amber-500' }, `¥${row.price}`);
    }
  },
  { title: '摄影师', key: 'photographerName', minWidth: 120 },
  { title: '服务', key: 'serviceName', minWidth: 140 }
];

async function openDetail(row: Api.Admin.AdminUser) {
  drawerShow.value = true;
  drawerLoading.value = true;
  detail.value = null;

  try {
    detail.value = await fetchAdminUserDetail(row.id);
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
    <NCard :bordered="false" class="card-wrapper" title="用户管理">
    <NSpace class="mb-16px" size="medium" align="center" wrap>
      <NInput
        v-model:value="searchForm.keyword"
        placeholder="搜索昵称 / 手机号"
        clearable
        class="w-220px"
        @keyup.enter="handleSearch"
      />
      <NSelect
        v-model:value="searchForm.role"
        :options="roleOptions"
        class="w-130px"
        @update:value="handleSearch"
      />
      <NSelect
        v-model:value="searchForm.status"
        :options="statusOptions"
        class="w-130px"
        @update:value="handleSearch"
      />
      <NButton type="primary" @click="handleSearch">查询</NButton>
      <NButton @click="handleReset">重置</NButton>
      <NButton secondary :loading="exporting" @click="handleExport">导出 CSV</NButton>
    </NSpace>
    <NDataTable
      remote
      :columns="columns"
      :data="list"
      :loading="loading"
      :bordered="false"
      :single-line="false"
      :scroll-x="1100"
      :pagination="tablePagination"
      @update:page="handlePageChange"
      @update:page-size="handlePageSizeChange"
    />
  </NCard>

  <NDrawer v-model:show="drawerShow" width="720">
    <NDrawerContent title="用户详情" closable>
      <NSpin :show="drawerLoading">
        <template v-if="detail">
          <div class="flex-y-center gap-16px mb-20px">
            <NAvatar
              round
              :size="64"
              :src="detail.avatar"
              :fallback-src="'https://api.dicebear.com/7.x/avataaars/svg?seed=fallback'"
            />
            <div class="flex-1 min-w-0">
              <div class="text-18px font-600">{{ detail.name || '未命名用户' }}</div>
              <div class="text-13px text-[var(--n-text-color-3)] mt-4px">{{ detail.phone }}</div>
            </div>
            <NTag
              size="small"
              :type="detail.status === 'active' ? 'success' : 'error'"
              round
              :bordered="false"
            >
              {{ detail.status === 'active' ? '启用' : '禁用' }}
            </NTag>
          </div>

          <NDescriptions :column="2" bordered size="small" class="mb-20px">
            <NDescriptionsItem label="用户ID">{{ detail.id }}</NDescriptionsItem>
            <NDescriptionsItem label="角色">
              {{ ROLE_NAMES[detail.role] ?? detail.role }}
            </NDescriptionsItem>
            <NDescriptionsItem label="手机号">{{ detail.phone || '—' }}</NDescriptionsItem>
            <NDescriptionsItem label="注册时间">{{ formatTime(detail.createdAt) }}</NDescriptionsItem>
            <NDescriptionsItem label="摄影师ID">{{ detail.photographerId ?? '—' }}</NDescriptionsItem>
            <NDescriptionsItem label="昵称">{{ detail.name || '—' }}</NDescriptionsItem>
          </NDescriptions>

          <div class="grid grid-cols-4 gap-12px mb-20px">
            <div v-for="card in statCards" :key="card.label" class="stat-card">
              <div class="text-24px font-600">{{ card.value }}</div>
              <div class="text-12px text-[var(--n-text-color-3)]">{{ card.label }}</div>
            </div>
          </div>

          <div class="text-14px font-600 mb-8px">最近订单</div>
          <NDataTable
            :columns="bookingColumns"
            :data="detail.recentBookings"
            :bordered="false"
            :single-line="false"
            :scroll-x="760"
            size="small"
          />
        </template>
        <NEmpty v-else-if="!drawerLoading" description="加载失败" />
      </NSpin>
    </NDrawerContent>
  </NDrawer>
  </div>
</template>

<style scoped>
.stat-card {
  padding: 16px;
  border-radius: 8px;
  background: var(--n-color-embedded);
  text-align: center;
}
</style>
