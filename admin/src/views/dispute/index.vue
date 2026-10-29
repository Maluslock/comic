<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue';
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import { NButton, NSpace, NTag } from 'naive-ui';
import { fetchDisputeDetail, fetchDisputes, resolveDispute } from '@/service/api/admin';
import { getAdminApiErrorMessage } from '@/service/request/admin';

const loading = ref(false);
const list = ref<Api.Admin.Dispute[]>([]);
const total = ref(0);

const statusTab = ref<Api.Admin.DisputeFilter>('all');

const page = ref(1);
const pageSize = ref(10);

const tabs: { key: Api.Admin.DisputeFilter; label: string }[] = [
  { key: 'all', label: '全部' },
  { key: 'open', label: '待响应' },
  { key: 'responding', label: '响应中' },
  { key: 'escalated', label: '已升级' },
  { key: 'resolved', label: '已裁决' },
  { key: 'rejected', label: '已驳回' }
];

const STATUS_NAMES: Record<Api.Admin.DisputeStatus, string> = {
  open: '待响应',
  responding: '响应中',
  escalated: '已升级',
  resolved: '已裁决',
  closed: '已关闭'
};

const STATUS_TAG_TYPES: Record<Api.Admin.DisputeStatus, 'default' | 'warning' | 'success' | 'error' | 'info'> = {
  open: 'warning',
  responding: 'info',
  escalated: 'error',
  resolved: 'success',
  closed: 'default'
};

const TYPE_NAMES: Record<Api.Admin.DisputeType, string> = {
  no_show: '未到场',
  not_delivered: '未交付',
  quality: '质量问题',
  other: '其他'
};

const ROLE_NAMES: Record<'coser' | 'photographer', string> = {
  coser: 'Coser',
  photographer: '摄影师'
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

const columns: DataTableColumns<Api.Admin.Dispute> = [
  {
    title: '争议ID',
    key: 'id',
    width: 90
  },
  {
    title: '订单号',
    key: 'bookingId',
    width: 100,
    render(row) {
      return h('span', null, `#${row.bookingId}`);
    }
  },
  {
    title: '争议类型',
    key: 'type',
    width: 110,
    render(row) {
      return h('span', null, TYPE_NAMES[row.type] ?? row.type);
    }
  },
  {
    title: '申请方',
    key: 'openedByRole',
    width: 150,
    render(row) {
      return h(
        'span',
        null,
        `${ROLE_NAMES[row.openedByRole] ?? row.openedByRole} · ${row.openedByRole === 'coser' ? row.coserName : row.photographerName || '—'}`
      );
    }
  },
  {
    title: '双方',
    key: 'parties',
    minWidth: 170,
    render(row) {
      return h('span', null, `${row.coserName || '—'} / ${row.photographerName || '—'}`);
    }
  },
  {
    title: '理由',
    key: 'reason',
    minWidth: 200,
    ellipsis: { tooltip: true },
    render(row) {
      return h('span', null, row.reason || '—');
    }
  },
  {
    title: '响应截止',
    key: 'respondDeadline',
    width: 170,
    render(row) {
      return h('span', null, formatTime(row.respondDeadline));
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

      if (row.status === 'open' || row.status === 'responding' || row.status === 'escalated') {
        actions.push(
          h(
            NButton,
            { size: 'small', type: 'success', secondary: true, onClick: () => openResolve(row, 'approve') },
            { default: () => '支持申请方' }
          ),
          h(
            NButton,
            { size: 'small', type: 'error', secondary: true, onClick: () => openResolve(row, 'reject') },
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
    const { list: rows, total: count } = await fetchDisputes({
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

// ---------- detail drawer ----------
const drawerShow = ref(false);
const drawerLoading = ref(false);
const detail = ref<Api.Admin.DisputeDetail | null>(null);

async function openDetail(row: Api.Admin.Dispute) {
  drawerShow.value = true;
  drawerLoading.value = true;
  detail.value = null;

  try {
    detail.value = await fetchDisputeDetail(row.id);
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    drawerLoading.value = false;
  }
}

function parsePayload(payload: string): Record<string, unknown> {
  try {
    const parsed = JSON.parse(payload || '{}');
    return typeof parsed === 'object' && parsed !== null ? (parsed as Record<string, unknown>) : {};
  } catch {
    return {};
  }
}

function eventText(event: Api.Admin.DisputeEvent) {
  const data = parsePayload(event.payload);

  if (event.eventType === 'open') {
    return `发起争议：${String(data.reason ?? '')}`;
  }

  if (event.eventType === 'respond') {
    return `回应：${String(data.response ?? '')}`;
  }

  if (event.eventType === 'escalate') {
    return `升级争议：${String(data.reason ?? '')}`;
  }

  if (event.eventType === 'resolve') {
    return `裁决：${String(data.resolution ?? '')}`;
  }

  return event.eventType;
}

function evidenceOf(payload: string): string[] {
  const data = parsePayload(payload);
  const evidence = data.evidence;

  return Array.isArray(evidence) ? evidence.map(item => String(item)) : [];
}

// ---------- resolve modal ----------
const resolveShow = ref(false);
const resolveLoading = ref(false);
const resolveTarget = ref<Api.Admin.Dispute | null>(null);
const resolveAction = ref<Api.Admin.DisputeResolveAction>('approve');
const resolveReason = ref('');

const resolveActionOptions: { label: string; value: Api.Admin.DisputeResolveAction }[] = [
  { label: '支持申请方（申请方胜诉，被申请方违约 -10 分）', value: 'approve' },
  { label: '驳回申请（被申请方胜诉，申请方违约 -10 分）', value: 'reject' },
  { label: '升级处理（暂不裁决，等待平台介入）', value: 'escalate' }
];

function openResolve(row: Api.Admin.Dispute, action: Api.Admin.DisputeResolveAction) {
  resolveTarget.value = row;
  resolveAction.value = action;
  resolveReason.value = '';
  resolveShow.value = true;
}

async function submitResolve() {
  const reason = resolveReason.value.trim();

  if (resolveAction.value === 'reject' && !reason) {
    window.$message?.warning('驳回申请必须填写理由');

    return;
  }

  if (!resolveTarget.value) {
    return;
  }

  resolveLoading.value = true;

  try {
    await resolveDispute(resolveTarget.value.id, { action: resolveAction.value, reason });

    window.$message?.success(
      resolveAction.value === 'approve'
        ? '已裁决：支持申请方'
        : resolveAction.value === 'reject'
          ? '已驳回争议申请'
          : '争议已升级'
    );

    resolveShow.value = false;

    await loadList();

    if (drawerShow.value && detail.value) {
      await openDetail(detail.value);
    }
  } catch (error) {
    window.$message?.error(mapResolveError(error));
  } finally {
    resolveLoading.value = false;
  }
}

function mapResolveError(error: unknown) {
  const message = getAdminApiErrorMessage(error);

  if (message === 'invalid resolve request') {
    return '裁决参数不合法（驳回必须填写理由）';
  }

  if (message === 'invalid dispute state') {
    return '该争议当前状态不可裁决';
  }

  if (message === 'dispute not found') {
    return '争议不存在';
  }

  return message;
}

onMounted(loadList);
</script>

<template>
  <div>
    <NCard :bordered="false" class="card-wrapper" title="争议处理">
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
        :pagination="tablePagination"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </NCard>

    <NModal v-model:show="resolveShow" preset="card" title="争议裁决" class="w-560px">
      <template v-if="resolveTarget">
        <NDescriptions :column="2" bordered size="small" class="mb-16px">
          <NDescriptionsItem label="争议ID">{{ resolveTarget.id }}</NDescriptionsItem>
          <NDescriptionsItem label="订单号">#{{ resolveTarget.bookingId }}</NDescriptionsItem>
          <NDescriptionsItem label="争议类型">
            {{ TYPE_NAMES[resolveTarget.type] ?? resolveTarget.type }}
          </NDescriptionsItem>
          <NDescriptionsItem label="申请方">
            {{ ROLE_NAMES[resolveTarget.openedByRole] ?? resolveTarget.openedByRole }}
          </NDescriptionsItem>
        </NDescriptions>

        <NRadioGroup v-model:value="resolveAction" class="mb-12px">
          <NSpace vertical>
            <NRadio v-for="option in resolveActionOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </NRadio>
          </NSpace>
        </NRadioGroup>

        <NInput
          v-model:value="resolveReason"
          type="textarea"
          placeholder="裁决说明（驳回申请时必填）"
          :autosize="{ minRows: 3, maxRows: 6 }"
          maxlength="500"
          show-count
        />
      </template>
      <template #footer>
        <div class="flex justify-end gap-8px">
          <NButton @click="resolveShow = false">取消</NButton>
          <NButton type="primary" :loading="resolveLoading" @click="submitResolve">确认裁决</NButton>
        </div>
      </template>
    </NModal>

    <NDrawer v-model:show="drawerShow" width="760">
      <NDrawerContent title="争议详情" closable>
        <NSpin :show="drawerLoading">
          <template v-if="detail">
            <NDescriptions :column="2" bordered size="small" class="mb-20px">
              <NDescriptionsItem label="争议ID">{{ detail.id }}</NDescriptionsItem>
              <NDescriptionsItem label="订单号">#{{ detail.bookingId }}</NDescriptionsItem>
              <NDescriptionsItem label="状态">
                <NTag size="small" :type="STATUS_TAG_TYPES[detail.status] ?? 'default'" round :bordered="false">
                  {{ STATUS_NAMES[detail.status] ?? detail.status }}
                </NTag>
              </NDescriptionsItem>
              <NDescriptionsItem label="争议类型">
                {{ TYPE_NAMES[detail.type] ?? detail.type }}
              </NDescriptionsItem>
              <NDescriptionsItem label="申请方">
                {{ ROLE_NAMES[detail.openedByRole] ?? detail.openedByRole }}（{{ detail.openedBy }}）
              </NDescriptionsItem>
              <NDescriptionsItem label="订单状态">
                {{ detail.bookingStatus }} / {{ detail.bookingSubStatus || '—' }}
              </NDescriptionsItem>
              <NDescriptionsItem label="Coser">{{ detail.coserName || '—' }}</NDescriptionsItem>
              <NDescriptionsItem label="摄影师">{{ detail.photographerName || '—' }}</NDescriptionsItem>
              <NDescriptionsItem label="发起时间">{{ formatTime(detail.createdAt) }}</NDescriptionsItem>
              <NDescriptionsItem label="响应截止">{{ formatTime(detail.respondDeadline) }}</NDescriptionsItem>
              <NDescriptionsItem label="举证截止">{{ formatTime(detail.evidenceDeadline) }}</NDescriptionsItem>
              <NDescriptionsItem label="裁决时间">{{ formatTime(detail.reviewedAt) }}</NDescriptionsItem>
              <NDescriptionsItem label="裁决结果" :span="2">{{ detail.resolution || '—' }}</NDescriptionsItem>
            </NDescriptions>

            <div class="text-14px font-600 mb-8px">申请理由</div>
            <div class="text-13px text-[var(--n-text-color-2)] mb-20px whitespace-pre-wrap">
              {{ detail.reason || '—' }}
            </div>

            <div class="text-14px font-600 mb-8px">申请方举证</div>
            <NImageGroup v-if="detail.evidence && detail.evidence.length > 0">
              <NImage
                v-for="(image, index) in detail.evidence"
                :key="index"
                :src="image"
                :width="120"
                :height="120"
                object-fit="cover"
                class="mr-8px mb-8px"
              />
            </NImageGroup>
            <div v-else class="text-13px text-[var(--n-text-color-3)] mb-20px">无举证材料</div>

            <div class="text-14px font-600 mb-8px">订单交付证据</div>
            <NImageGroup v-if="detail.deliveryEvidence && detail.deliveryEvidence.length > 0">
              <NImage
                v-for="(image, index) in detail.deliveryEvidence"
                :key="index"
                :src="image"
                :width="120"
                :height="120"
                object-fit="cover"
                class="mr-8px mb-8px"
              />
            </NImageGroup>
            <div v-else class="text-13px text-[var(--n-text-color-3)] mb-20px">无交付证据</div>

            <div class="text-14px font-600 mb-8px">事件时间线</div>
            <NTimeline v-if="detail.events && detail.events.length > 0">
              <NTimelineItem
                v-for="event in detail.events"
                :key="event.id"
                :type="event.actorRole === 'admin' ? 'info' : 'default'"
                :title="eventText(event)"
              >
                <div class="text-12px text-[var(--n-text-color-3)]">
                  {{ formatTime(event.createdAt) }} · {{ event.actorRole || 'system' }}
                </div>
                <NImageGroup v-if="evidenceOf(event.payload).length > 0" class="mt-4px">
                  <NImage
                    v-for="(image, index) in evidenceOf(event.payload)"
                    :key="index"
                    :src="image"
                    :width="80"
                    :height="80"
                    object-fit="cover"
                    class="mr-8px mb-8px"
                  />
                </NImageGroup>
              </NTimelineItem>
            </NTimeline>
            <div v-else class="text-13px text-[var(--n-text-color-3)]">暂无事件</div>

            <div
              v-if="detail.status === 'open' || detail.status === 'responding' || detail.status === 'escalated'"
              class="mt-24px"
            >
              <NSpace>
                <NButton type="success" @click="openResolve(detail, 'approve')">支持申请方</NButton>
                <NButton type="error" @click="openResolve(detail, 'reject')">驳回申请</NButton>
                <NButton type="warning" secondary @click="openResolve(detail, 'escalate')">升级处理</NButton>
              </NSpace>
            </div>
          </template>
          <NEmpty v-else-if="!drawerLoading" description="加载失败" />
        </NSpin>
      </NDrawerContent>
    </NDrawer>
  </div>
</template>

<style scoped></style>
