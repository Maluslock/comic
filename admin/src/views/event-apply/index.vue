<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue';
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import { NButton, NSpace, NTag, NTooltip } from 'naive-ui';
import { fetchEventApplications, reviewEventApplication } from '@/service/api/admin';
import { getAdminApiErrorMessage } from '@/service/request/admin';

const loading = ref(false);
const list = ref<Api.Admin.AdminEventApplication[]>([]);
const total = ref(0);

const statusTab = ref<Api.Admin.EventApplicationFilter>('pending');

const page = ref(1);
const pageSize = ref(10);

const tabs: { key: Api.Admin.EventApplicationFilter; label: string }[] = [
  { key: 'pending', label: '待审核' },
  { key: 'approved', label: '已通过' },
  { key: 'rejected', label: '未通过' },
  { key: 'all', label: '全部' }
];

const STATUS_NAMES: Record<Api.Admin.EventApplicationStatus, string> = {
  pending: '待审核',
  approved: '已通过',
  rejected: '未通过',
  withdrawn: '已撤回'
};

const STATUS_TAG_TYPES: Record<Api.Admin.EventApplicationStatus, 'default' | 'warning' | 'success' | 'error'> = {
  pending: 'warning',
  approved: 'success',
  rejected: 'error',
  withdrawn: 'default'
};

const tablePagination = computed<PaginationProps>(() => ({
  page: page.value,
  pageSize: pageSize.value,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  showQuickJumper: true,
  itemCount: total.value,
  prefix: ({ itemCount }) => `共 ${itemCount ?? 0} 条`
}));

const columns: DataTableColumns<Api.Admin.AdminEventApplication> = [
  {
    title: '漫展名',
    key: 'eventName',
    minWidth: 260,
    ellipsis: { tooltip: true },
    render(row) {
      return h('div', null, [
        h('div', null, row.eventName || '—'),
        h(
          'div',
          { class: 'text-12px text-[var(--n-text-color-3)]' },
          `#${row.eventId} · ${formatDateRange(row.startDate, row.endDate)}`
        )
      ]);
    }
  },
  {
    title: '摄影师名',
    key: 'photographerName',
    minWidth: 160,
    render(row) {
      return h('div', null, [
        h('div', null, row.photographerName || '—'),
        h('div', { class: 'text-12px text-[var(--n-text-color-3)]' }, row.userName || '—')
      ]);
    }
  },
  {
    title: '申请理由',
    key: 'message',
    minWidth: 240,
    ellipsis: { tooltip: true },
    render(row) {
      return h('span', null, row.message || '—');
    }
  },
  {
    title: '提交时间',
    key: 'appliedAt',
    width: 170,
    render(row) {
      return h('span', null, formatTime(row.appliedAt));
    }
  },
  {
    title: '状态',
    key: 'status',
    width: 120,
    render(row) {
      const tag = h(
        NTag,
        { size: 'small', type: STATUS_TAG_TYPES[row.status] ?? 'default', round: true, bordered: false },
        { default: () => STATUS_NAMES[row.status] ?? row.status }
      );

      if (!row.reviewReason) {
        return tag;
      }

      return h(NTooltip, null, {
        trigger: () => tag,
        default: () => `审核意见：${row.reviewReason}`
      });
    }
  },
  {
    title: '操作',
    key: 'actions',
    width: 180,
    fixed: 'right',
    render(row) {
      if (row.status !== 'pending') {
        return h('span', { class: 'text-[var(--n-text-color-3)]' }, '已处理');
      }

      return h(NSpace, null, {
        default: () => [
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
        ]
      });
    }
  }
];

function formatTime(value: string | null | undefined) {
  if (!value) {
    return '—';
  }

  return value.replace('T', ' ').slice(0, 16);
}

function formatDate(value: string | null | undefined) {
  if (!value) {
    return '';
  }

  return value.slice(0, 10);
}

function formatDateRange(start: string | null | undefined, end: string | null | undefined) {
  const from = formatDate(start);
  const to = formatDate(end);

  if (!from && !to) {
    return '场次时间待定';
  }

  if (!from) {
    return `至 ${to}`;
  }

  if (!to || from === to) {
    return from;
  }

  return `${from} ~ ${to}`;
}

async function loadList() {
  loading.value = true;

  try {
    const { list: rows, total: count } = await fetchEventApplications({
      status: statusTab.value === 'all' ? '' : statusTab.value,
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

function handleApprove(row: Api.Admin.AdminEventApplication) {
  window.$dialog?.warning({
    title: '通过入驻申请',
    content: `确认通过「${row.photographerName || row.userName || '该摄影师'}」入驻「${row.eventName || '该漫展'}」的申请？通过后该摄影师将出现在漫展详情的摄影师列表中。`,
    positiveText: '确认通过',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await reviewEventApplication(row.id, { status: 'approved' });

        window.$message?.success('已通过该入驻申请');

        await loadList();
      } catch (error) {
        window.$message?.error(mapReviewError(error));
      }
    }
  });
}

const rejectShow = ref(false);
const rejectLoading = ref(false);
const rejectTarget = ref<Api.Admin.AdminEventApplication | null>(null);
const rejectReason = ref('');

function openRejectModal(row: Api.Admin.AdminEventApplication) {
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
    await reviewEventApplication(rejectTarget.value.id, { status: 'rejected', reason });

    window.$message?.success('已驳回该入驻申请');

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

  if (message === 'application already reviewed') {
    return '该申请已被处理，请刷新列表';
  }

  if (message === 'application not found') {
    return '申请不存在（可能已被删除）';
  }

  if (message === 'reason is required when rejecting') {
    return '驳回必须填写理由';
  }

  if (message === 'status must be approved or rejected') {
    return '审核状态不合法';
  }

  return message;
}

onMounted(loadList);
</script>

<template>
  <div>
    <NCard :bordered="false" class="card-wrapper" title="漫展入驻审核">
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
        :pagination="tablePagination"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </NCard>

    <NModal v-model:show="rejectShow" preset="card" title="驳回入驻申请" class="w-520px">
      <template v-if="rejectTarget">
        <div class="text-13px text-[var(--n-text-color-3)] mb-4px">
          摄影师：{{ rejectTarget.photographerName || rejectTarget.userName || '—' }}
        </div>
        <div class="text-13px text-[var(--n-text-color-3)] mb-12px">漫展：{{ rejectTarget.eventName || '—' }}</div>
        <NInput
          v-model:value="rejectReason"
          type="textarea"
          placeholder="请填写驳回理由（必填，会展示给申请人）"
          :autosize="{ minRows: 3, maxRows: 6 }"
          maxlength="200"
          show-count
        />
      </template>
      <template #footer>
        <div class="flex justify-end gap-8px">
          <NButton @click="rejectShow = false">取消</NButton>
          <NButton
            type="error"
            :loading="rejectLoading"
            :disabled="!rejectReason.trim()"
            @click="submitReject"
          >
            确认驳回
          </NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped></style>
