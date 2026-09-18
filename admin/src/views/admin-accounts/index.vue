<script setup lang="ts">
import { h, onMounted, ref } from 'vue';
import type { DataTableColumns, FormInst, FormRules, SelectOption } from 'naive-ui';
import { NButton, NSpace, NTooltip, NTag } from 'naive-ui';
import { createAdmin, fetchAdmins, resetAdminPassword, setAdminStatus } from '@/service/api/admin';
import { getAdminApiErrorMessage } from '@/service/request/admin';

const PRIMARY_ADMIN_ID = 1;

const loading = ref(false);
const list = ref<Api.Admin.AdminAccount[]>([]);

const ROLE_NAMES: Record<Api.Admin.AdminRole, string> = {
  admin: '管理员',
  super: '超级管理员'
};

const columns: DataTableColumns<Api.Admin.AdminAccount> = [
  {
    title: 'ID',
    key: 'id',
    width: 80
  },
  {
    title: '用户名',
    key: 'username',
    minWidth: 140,
    render(row) {
      return h('span', null, row.username || '—');
    }
  },
  {
    title: '角色',
    key: 'role',
    width: 120,
    render(row) {
      return h(
        NTag,
        { size: 'small', type: row.role === 'super' ? 'warning' : 'info', round: true, bordered: false },
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
    width: 220,
    fixed: 'right',
    render(row) {
      return h(NSpace, null, {
        default: () => [
          h(
            NButton,
            { size: 'small', type: 'primary', secondary: true, onClick: () => openReset(row) },
            { default: () => '重置密码' }
          ),
          h(
            NTooltip,
            { disabled: row.id !== PRIMARY_ADMIN_ID, trigger: 'hover' },
            {
              trigger: () =>
                h(
                  NButton,
                  {
                    size: 'small',
                    type: row.status === 'active' ? 'error' : 'primary',
                    secondary: true,
                    disabled: row.id === PRIMARY_ADMIN_ID && row.status === 'active',
                    onClick: () => handleToggleStatus(row)
                  },
                  { default: () => (row.status === 'active' ? '禁用' : '启用') }
                ),
              default: () => '主管理员不可禁用'
            }
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
    list.value = await fetchAdmins();
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    loading.value = false;
  }
}

function handleToggleStatus(row: Api.Admin.AdminAccount) {
  const next: Api.Admin.AdminStatus = row.status === 'active' ? 'disabled' : 'active';
  const isDisable = next === 'disabled';

  window.$dialog?.warning({
    title: isDisable ? '禁用管理员' : '启用管理员',
    content: `${isDisable ? '禁用' : '启用'}管理员「${row.username}」？${isDisable ? '禁用后该账号将无法登录。' : ''}`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await setAdminStatus(row.id, next);

        window.$message?.success(isDisable ? '已禁用该管理员' : '已启用该管理员');

        await loadList();
      } catch (error) {
        window.$message?.error(mapStatusError(error));
      }
    }
  });
}

function mapStatusError(error: unknown) {
  const message = getAdminApiErrorMessage(error);

  if (message === 'cannot disable primary admin') {
    return '主管理员不可禁用';
  }

  return message;
}

const createShow = ref(false);
const createLoading = ref(false);
const createFormRef = ref<FormInst | null>(null);

const roleOptions: SelectOption[] = [
  { label: '管理员', value: 'admin' },
  { label: '超级管理员', value: 'super' }
];

const createModel = ref({
  username: '',
  password: '',
  role: 'admin' as Api.Admin.AdminRole
});

const createRules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码至少 6 位', trigger: 'blur' }
  ]
};

function openCreate() {
  createModel.value = { username: '', password: '', role: 'admin' };
  createShow.value = true;
}

async function handleCreate() {
  await createFormRef.value?.validate();

  createLoading.value = true;

  try {
    await createAdmin({
      username: createModel.value.username.trim(),
      password: createModel.value.password,
      role: createModel.value.role
    });

    window.$message?.success('新增成功');

    createShow.value = false;

    await loadList();
  } catch (error) {
    window.$message?.error(mapCreateError(error));
  } finally {
    createLoading.value = false;
  }
}

function mapCreateError(error: unknown) {
  const message = getAdminApiErrorMessage(error);

  if (message === 'username exists') {
    return '该用户名已存在';
  }

  return message;
}

const resetShow = ref(false);
const resetLoading = ref(false);
const resetTarget = ref<Api.Admin.AdminAccount | null>(null);
const resetFormRef = ref<FormInst | null>(null);

const resetModel = ref({
  password: ''
});

const resetRules: FormRules = {
  password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码至少 6 位', trigger: 'blur' }
  ]
};

function openReset(row: Api.Admin.AdminAccount) {
  resetTarget.value = row;
  resetModel.value = { password: '' };
  resetShow.value = true;
}

async function handleReset() {
  await resetFormRef.value?.validate();

  if (!resetTarget.value) {
    return;
  }

  resetLoading.value = true;

  try {
    await resetAdminPassword(resetTarget.value.id, resetModel.value.password);

    window.$message?.success('密码已重置，该账号的登录状态已失效');

    resetShow.value = false;

    await loadList();
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    resetLoading.value = false;
  }
}

onMounted(loadList);
</script>

<template>
  <div>
    <NCard :bordered="false" class="card-wrapper" title="管理员管理">
      <template #header-extra>
        <NButton type="primary" @click="openCreate">新建管理员</NButton>
      </template>
      <NDataTable
        :columns="columns"
        :data="list"
        :loading="loading"
        :bordered="false"
        :single-line="false"
        :scroll-x="900"
      />
    </NCard>

    <NModal v-model:show="createShow" preset="card" title="新建管理员" class="w-440px">
      <NForm ref="createFormRef" :model="createModel" :rules="createRules" label-placement="top">
        <NFormItem label="用户名" path="username">
          <NInput v-model:value="createModel.username" placeholder="请输入用户名" clearable />
        </NFormItem>
        <NFormItem label="密码" path="password">
          <NInput
            v-model:value="createModel.password"
            type="password"
            show-password-on="click"
            placeholder="至少 6 位"
          />
        </NFormItem>
        <NFormItem label="角色" path="role">
          <NSelect v-model:value="createModel.role" :options="roleOptions" />
        </NFormItem>
      </NForm>
      <template #footer>
        <div class="flex justify-end gap-8px">
          <NButton @click="createShow = false">取消</NButton>
          <NButton type="primary" :loading="createLoading" @click="handleCreate">创建</NButton>
        </div>
      </template>
    </NModal>

    <NModal v-model:show="resetShow" preset="card" title="重置密码" class="w-440px">
      <template v-if="resetTarget">
        <div class="text-13px text-[var(--n-text-color-3)] mb-12px">管理员：{{ resetTarget.username }}</div>
        <NForm ref="resetFormRef" :model="resetModel" :rules="resetRules" label-placement="top">
          <NFormItem label="新密码" path="password">
            <NInput
              v-model:value="resetModel.password"
              type="password"
              show-password-on="click"
              placeholder="至少 6 位"
            />
          </NFormItem>
        </NForm>
      </template>
      <template #footer>
        <div class="flex justify-end gap-8px">
          <NButton @click="resetShow = false">取消</NButton>
          <NButton type="primary" :loading="resetLoading" @click="handleReset">确认重置</NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped></style>
