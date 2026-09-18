<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue';
import type { DataTableColumns, FormInst, FormRules, PaginationProps, SelectOption } from 'naive-ui';
import { NButton, NTag } from 'naive-ui';
import { deleteNotification, fetchAdminUsers, fetchNotifications, publishNotification } from '@/service/api/admin';
import { getAdminApiErrorMessage } from '@/service/request/admin';

const loading = ref(false);
const list = ref<Api.Admin.NotificationItem[]>([]);
const total = ref(0);

const page = ref(1);
const pageSize = ref(10);

const tablePagination = computed<PaginationProps>(() => ({
  page: page.value,
  pageSize: pageSize.value,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  itemCount: total.value,
  prefix: ({ itemCount }) => `共 ${itemCount ?? 0} 条`
}));

const TYPE_NAMES: Record<string, string> = {
  success: '成功',
  info: '提示',
  warning: '警告'
};

const TYPE_TAG_TYPES: Record<string, 'success' | 'info' | 'warning' | 'default'> = {
  success: 'success',
  info: 'info',
  warning: 'warning'
};

const columns: DataTableColumns<Api.Admin.NotificationItem> = [
  { title: 'ID', key: 'id', width: 80 },
  {
    title: '标题',
    key: 'title',
    minWidth: 180
  },
  {
    title: '类型',
    key: 'type',
    width: 90,
    render(row) {
      return h(
        NTag,
        { size: 'small', type: TYPE_TAG_TYPES[row.type] ?? 'default', round: true, bordered: false },
        { default: () => TYPE_NAMES[row.type] ?? row.type }
      );
    }
  },
  {
    title: '目标',
    key: 'targetType',
    width: 140,
    render(row) {
      if (row.targetType === 'all') {
        return h(NTag, { size: 'small', type: 'info', round: true, bordered: false }, { default: () => '全员' });
      }

      return h(
        NTag,
        { size: 'small', type: 'warning', round: true, bordered: false },
        { default: () => `指定用户 #${row.userId ?? '—'}` }
      );
    }
  },
  {
    title: '内容',
    key: 'content',
    minWidth: 240,
    ellipsis: { tooltip: true }
  },
  {
    title: '发布时间',
    key: 'createdAt',
    width: 160,
    render(row) {
      return h('span', null, formatTime(row.createdAt));
    }
  },
  {
    title: '操作',
    key: 'actions',
    width: 100,
    fixed: 'right',
    render(row) {
      return h(
        NButton,
        { size: 'small', type: 'error', secondary: true, onClick: () => handleRecall(row) },
        { default: () => '撤回' }
      );
    }
  }
];

function formatTime(value: string) {
  if (!value) {
    return '—';
  }

  return value.replace('T', ' ').slice(0, 16);
}

async function handleRecall(row: Api.Admin.NotificationItem) {
  window.$dialog?.warning({
    title: '撤回通知',
    content: `确定撤回「${row.title}」吗？撤回后用户将不再看到该通知`,
    positiveText: '撤回',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteNotification(row.id);
        window.$message?.success('已撤回');
        await loadList();
      } catch (error) {
        window.$message?.error(getAdminApiErrorMessage(error));
      }
    }
  });
}

async function loadList() {
  loading.value = true;

  try {
    const { list: rows, total: count } = await fetchNotifications({ page: page.value, pageSize: pageSize.value });

    list.value = rows;
    total.value = count;
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    loading.value = false;
  }
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

const typeOptions: SelectOption[] = [
  { label: '成功', value: 'success' },
  { label: '提示', value: 'info' },
  { label: '警告', value: 'warning' }
];

const targetOptions: SelectOption[] = [
  { label: '全员通知', value: 'all' },
  { label: '指定用户', value: 'single' }
];

const userOptions = ref<SelectOption[]>([]);
const userLoading = ref(false);

async function loadUserOptions() {
  userLoading.value = true;

  try {
    const { list: users } = await fetchAdminUsers({ pageSize: 20 });

    userOptions.value = users.map(user => ({
      label: user.name ? `${user.name} (${user.phone})` : user.phone,
      value: user.id
    }));
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    userLoading.value = false;
  }
}

const formRef = ref<FormInst | null>(null);
const publishing = ref(false);

const formModel = ref({
  type: 'info' as Api.Admin.NotificationType,
  title: '',
  content: '',
  targetType: 'all' as Api.Admin.NotificationTargetType,
  userId: null as number | null
});

const rules: FormRules = {
  type: [{ required: true, message: '请选择类型', trigger: 'change' }],
  title: [{ required: true, message: '请输入标题', trigger: 'blur' }],
  content: [{ required: true, message: '请输入内容', trigger: 'blur' }],
  userId: [
    {
      required: true,
      trigger: 'change',
      validator: (_rule, value: number | null) => {
        if (formModel.value.targetType === 'single' && !value) {
          return new Error('请选择目标用户');
        }

        return true;
      }
    }
  ]
};

async function handlePublish() {
  await formRef.value?.validate();

  publishing.value = true;

  try {
    const payload: Api.Admin.PublishNotificationRequest = {
      type: formModel.value.type,
      title: formModel.value.title.trim(),
      content: formModel.value.content.trim(),
      targetType: formModel.value.targetType
    };

    if (formModel.value.targetType === 'single') {
      payload.userId = formModel.value.userId ?? undefined;
    }

    await publishNotification(payload);

    window.$message?.success('发布成功');

    formModel.value.title = '';
    formModel.value.content = '';

    page.value = 1;

    await loadList();
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    publishing.value = false;
  }
}

onMounted(() => {
  loadList();
  loadUserOptions();
});
</script>

<template>
  <div>
    <NCard :bordered="false" class="card-wrapper mb-16px" title="发布通知">
      <NForm ref="formRef" :model="formModel" :rules="rules" label-placement="left" label-width="72">
        <NFormItem label="类型" path="type">
          <NSelect v-model:value="formModel.type" :options="typeOptions" class="w-220px" />
        </NFormItem>
        <NFormItem label="标题" path="title">
          <NInput v-model:value="formModel.title" placeholder="请输入标题" clearable class="w-full max-w-560px" />
        </NFormItem>
        <NFormItem label="内容" path="content">
          <NInput
            v-model:value="formModel.content"
            type="textarea"
            placeholder="请输入通知内容"
            :rows="4"
            class="w-full max-w-560px"
          />
        </NFormItem>
        <NFormItem label="目标" path="targetType">
          <NRadioGroup v-model:value="formModel.targetType">
            <NRadio v-for="opt in targetOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </NRadio>
          </NRadioGroup>
        </NFormItem>
        <NFormItem v-if="formModel.targetType === 'single'" label="用户" path="userId">
          <NSelect
            v-model:value="formModel.userId"
            :options="userOptions"
            :loading="userLoading"
            filterable
            clearable
            placeholder="选择目标用户（前 20 位用户）"
            class="w-full max-w-360px"
          />
        </NFormItem>
        <NFormItem>
          <NButton type="primary" :loading="publishing" @click="handlePublish">发布</NButton>
        </NFormItem>
      </NForm>
    </NCard>

    <NCard :bordered="false" class="card-wrapper" title="通知历史">
      <NDataTable
        remote
        :columns="columns"
        :data="list"
        :loading="loading"
        :bordered="false"
        :single-line="false"
        :scroll-x="1000"
        :pagination="tablePagination"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </NCard>
  </div>
</template>

<style scoped></style>
