<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue';
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import { NButton, NSpace, NTag } from 'naive-ui';
import {
  fetchCertApplicationDetail,
  fetchCertApplications,
  reviewCertApplication
} from '@/service/api/admin';
import { getAdminApiErrorMessage } from '@/service/request/admin';

const loading = ref(false);
const list = ref<Api.Admin.AdminCertApplication[]>([]);
const total = ref(0);

const statusTab = ref<'pending' | 'approved' | 'rejected'>('pending');

const page = ref(1);
const pageSize = ref(10);

const tabs = [
  { key: 'pending', label: '待审核' },
  { key: 'approved', label: '已通过' },
  { key: 'rejected', label: '已驳回' }
];

const STATUS_NAMES: Record<Api.Admin.CertApplicationStatus, string> = {
  pending: '待审核',
  approved: '已通过',
  rejected: '已驳回'
};

const STATUS_TAG_TYPES: Record<Api.Admin.CertApplicationStatus, 'default' | 'warning' | 'success' | 'error'> = {
  pending: 'warning',
  approved: 'success',
  rejected: 'error'
};

const tablePagination = computed<PaginationProps>(() => ({
  page: page.value,
  pageSize: pageSize.value,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  itemCount: total.value,
  prefix: ({ itemCount }) => `共 ${itemCount ?? 0} 条`
}));

const columns: DataTableColumns<Api.Admin.AdminCertApplication> = [
  {
    title: '申请ID',
    key: 'id',
    width: 90
  },
  {
    title: '申请人',
    key: 'userName',
    minWidth: 120,
    render(row) {
      return h('span', null, row.userName || '—');
    }
  },
  {
    title: '摄影师',
    key: 'photographerName',
    minWidth: 120,
    render(row) {
      return h('span', null, row.photographerName || '—');
    }
  },
  {
    title: '提交时间',
    key: 'createdAt',
    width: 170,
    render(row) {
      return h('span', null, formatTime(row.createdAt));
    }
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render(row) {
      return h(
        NTag,
        { size: 'small', type: STATUS_TAG_TYPES[row.status] ?? 'default', round: true, bordered: false },
        { default: () => STATUS_NAMES[row.status] ?? row.status }
      );
    }
  },
  {
    title: '操作',
    key: 'actions',
    width: 220,
    fixed: 'right',
    render(row) {
      const actions = [
        h(
          NButton,
          { size: 'small', type: 'primary', secondary: true, onClick: () => openDetail(row) },
          { default: () => '详情' }
        )
      ];

      if (row.status === 'pending') {
        actions.push(
          h(
            NButton,
            { size: 'small', type: 'success', secondary: true, onClick: () => handleApprove(row) },
            { default: () => '通过' }
          ),
          h(
            NButton,
            { size: 'small', type: 'error', secondary: true, onClick: () => openRejectModal(row) },
            { default: () => '驳回' }
          )
        );
      }

      return h(NSpace, null, { default: () => actions });
    }
  }
];

function formatTime(value: string | null | undefined) {
  if (!value) {
    return '—';
  }

  return value.replace('T', ' ').slice(0, 16);
}

async function loadList() {
  loading.value = true;

  try {
    const { list: rows, total: count } = await fetchCertApplications({
      status: statusTab.value,
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

function handleTabChange() {
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

function handleApprove(row: Api.Admin.AdminCertApplication) {
  window.$dialog?.warning({
    title: '通过认证',
    content: `确认通过「${row.photographerName || row.userName}」的认证申请？通过后该摄影师将获得认证标识。`,
    positiveText: '确认通过',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await reviewCertApplication(row.id, { action: 'approve' });

        window.$message?.success('已通过认证');

        await loadList();
      } catch (error) {
        window.$message?.error(mapReviewError(error));
      }
    }
  });
}

const rejectShow = ref(false);
const rejectLoading = ref(false);
const rejectTarget = ref<Api.Admin.AdminCertApplication | null>(null);
const rejectReason = ref('');

function openRejectModal(row: Api.Admin.AdminCertApplication) {
  rejectTarget.value = row;
  rejectReason.value = '';
  rejectShow.value = true;
}

async function submitReject() {
  const reason = rejectReason.value.trim();

  if (!reason) {
    window.$message?.warning('请填写驳回理由');

    return;
  }

  if (!rejectTarget.value) {
    return;
  }

  rejectLoading.value = true;

  try {
    await reviewCertApplication(rejectTarget.value.id, { action: 'reject', reason });

    window.$message?.success('已驳回该申请');

    rejectShow.value = false;

    await loadList();
  } catch (error) {
    window.$message?.error(mapReviewError(error));
  } finally {
    rejectLoading.value = false;
  }
}

function mapReviewError(error: unknown) {
  const message = getAdminApiErrorMessage(error);

  if (message === 'cert application already reviewed') {
    return '该申请已被处理';
  }

  return message;
}

const drawerShow = ref(false);
const drawerLoading = ref(false);
const detail = ref<Api.Admin.AdminCertApplication | null>(null);

async function openDetail(row: Api.Admin.AdminCertApplication) {
  drawerShow.value = true;
  drawerLoading.value = true;
  detail.value = null;

  try {
    detail.value = await fetchCertApplicationDetail(row.id);
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
    <NCard :bordered="false" class="card-wrapper" title="认证审核">
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
      :scroll-x="900"
      :pagination="tablePagination"
      @update:page="handlePageChange"
      @update:page-size="handlePageSizeChange"
    />
  </NCard>

  <NModal v-model:show="rejectShow" preset="card" title="驳回认证申请" class="w-520px">
    <template v-if="rejectTarget">
      <div class="text-13px text-[var(--n-text-color-3)] mb-12px">
        摄影师：{{ rejectTarget.photographerName || rejectTarget.userName }}
      </div>
      <NInput
        v-model:value="rejectReason"
        type="textarea"
        placeholder="请填写驳回理由（必填）"
        :autosize="{ minRows: 3, maxRows: 6 }"
        maxlength="200"
        show-count
      />
    </template>
    <template #footer>
      <div class="flex justify-end gap-8px">
        <NButton @click="rejectShow = false">取消</NButton>
        <NButton type="error" :loading="rejectLoading" @click="submitReject">确认驳回</NButton>
      </div>
    </template>
  </NModal>

  <NDrawer v-model:show="drawerShow" width="720">
    <NDrawerContent title="认证申请详情" closable>
      <NSpin :show="drawerLoading">
        <template v-if="detail">
          <NDescriptions :column="2" bordered size="small" class="mb-20px">
            <NDescriptionsItem label="申请ID">{{ detail.id }}</NDescriptionsItem>
            <NDescriptionsItem label="状态">
              <NTag
                size="small"
                :type="STATUS_TAG_TYPES[detail.status] ?? 'default'"
                round
                :bordered="false"
              >
                {{ STATUS_NAMES[detail.status] ?? detail.status }}
              </NTag>
            </NDescriptionsItem>
            <NDescriptionsItem label="申请人">{{ detail.userName || '—' }}</NDescriptionsItem>
            <NDescriptionsItem label="摄影师">{{ detail.photographerName || '—' }}</NDescriptionsItem>
            <NDescriptionsItem label="摄影师ID">{{ detail.photographerId }}</NDescriptionsItem>
            <NDescriptionsItem label="提交时间">{{ formatTime(detail.createdAt) }}</NDescriptionsItem>
            <NDescriptionsItem label="审核时间">{{ formatTime(detail.reviewedAt) }}</NDescriptionsItem>
            <NDescriptionsItem label="驳回理由" :span="2">
              {{ detail.reviewReason || '—' }}
            </NDescriptionsItem>
          </NDescriptions>

          <div class="text-14px font-600 mb-8px">申请说明</div>
          <div class="text-13px text-[var(--n-text-color-2)] mb-20px whitespace-pre-wrap">
            {{ detail.evidenceDesc || '—' }}
          </div>

          <div class="text-14px font-600 mb-8px">证明材料</div>
          <NImageGroup v-if="detail.evidenceImages && detail.evidenceImages.length > 0">
            <NImage
              v-for="(image, index) in detail.evidenceImages"
              :key="index"
              :src="image"
              :width="120"
              :height="120"
              object-fit="cover"
              class="mr-8px mb-8px"
            />
          </NImageGroup>
          <div v-else class="text-13px text-[var(--n-text-color-3)]">无证明材料</div>
        </template>
        <NEmpty v-else-if="!drawerLoading" description="加载失败" />
      </NSpin>
    </NDrawerContent>
  </NDrawer>
  </div>
</template>

<style scoped></style>
