<script setup lang="ts">
import { h, onMounted, ref } from 'vue';
import type { DataTableColumns, FormInst, FormRules, SelectOption } from 'naive-ui';
import { NButton, NImage, NSwitch, NTag } from 'naive-ui';
import { createBanner, fetchBanners, setBannerStatus, updateBanner } from '@/service/api/admin';
import { getAdminApiErrorMessage } from '@/service/request/admin';

const loading = ref(false);
const list = ref<Api.Admin.AdminBanner[]>([]);

const LINK_TYPE_NAMES: Record<string, string> = {
  event: '活动'
};

const columns: DataTableColumns<Api.Admin.AdminBanner> = [
  {
    title: '缩略图',
    key: 'imageUrl',
    width: 140,
    render(row) {
      return h(NImage, {
        width: 120,
        height: 70,
        objectFit: 'cover',
        src: row.imageUrl,
        fallbackSrc: 'https://picsum.photos/seed/banner-fallback/120/70',
        style: 'border-radius: 4px;'
      });
    }
  },
  {
    title: '标题',
    key: 'title',
    minWidth: 160,
    render(row) {
      return h('span', null, row.title || '—');
    }
  },
  {
    title: '类型',
    key: 'linkType',
    width: 90,
    render(row) {
      return h(
        NTag,
        { size: 'small', type: row.linkType === 'event' ? 'info' : 'default', round: true, bordered: false },
        { default: () => LINK_TYPE_NAMES[row.linkType] ?? (row.linkType || '—') }
      );
    }
  },
  {
    title: '关联ID',
    key: 'linkId',
    width: 90
  },
  {
    title: '排序',
    key: 'sortOrder',
    width: 80
  },
  {
    title: '上线',
    key: 'isActive',
    width: 90,
    render(row) {
      return h(NSwitch, {
        size: 'small',
        value: row.isActive,
        onUpdateValue: (value: boolean) => handleToggleActive(row, value)
      });
    }
  },
  {
    title: '创建时间',
    key: 'createdAt',
    width: 160,
    render(row) {
      return h('span', null, formatTime(row.createdAt));
    }
  },
  {
    title: '操作',
    key: 'actions',
    width: 90,
    fixed: 'right',
    render(row) {
      return h(
        NButton,
        { size: 'small', type: 'primary', secondary: true, onClick: () => openEdit(row) },
        { default: () => '编辑' }
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

async function loadList() {
  loading.value = true;

  try {
    list.value = await fetchBanners();
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    loading.value = false;
  }
}

async function handleToggleActive(row: Api.Admin.AdminBanner, value: boolean) {
  try {
    await setBannerStatus(row.id, value);

    window.$message?.success(value ? '已上线' : '已下线');

    await loadList();
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  }
}

const drawerShow = ref(false);
const drawerLoading = ref(false);
const editingId = ref<number | null>(null);

const linkTypeOptions: SelectOption[] = [{ label: '活动', value: 'event' }];

const formRef = ref<FormInst | null>(null);

const formModel = ref({
  imageUrl: '',
  title: '',
  linkType: 'event',
  linkId: null as number | null,
  sortOrder: 0 as number | null,
  isActive: true
});

const rules: FormRules = {
  title: [{ required: true, message: '请输入标题', trigger: 'blur' }]
};

function openCreate() {
  editingId.value = null;
  formModel.value = {
    imageUrl: '',
    title: '',
    linkType: 'event',
    linkId: null,
    sortOrder: 0,
    isActive: true
  };
  drawerShow.value = true;
}

function openEdit(row: Api.Admin.AdminBanner) {
  editingId.value = row.id;
  formModel.value = {
    imageUrl: row.imageUrl,
    title: row.title,
    linkType: row.linkType || 'event',
    linkId: row.linkId ?? null,
    sortOrder: row.sortOrder ?? 0,
    isActive: row.isActive
  };
  drawerShow.value = true;
}

async function handleSave() {
  await formRef.value?.validate();

  drawerLoading.value = true;

  try {
    const payload: Partial<Api.Admin.AdminBanner> = {
      imageUrl: formModel.value.imageUrl.trim(),
      title: formModel.value.title.trim(),
      linkType: formModel.value.linkType,
      linkId: formModel.value.linkId ?? 0,
      sortOrder: formModel.value.sortOrder ?? 0
    };

    if (editingId.value === null) {
      await createBanner(payload);

      window.$message?.success('新增成功');
    } else {
      // the update endpoint takes isActive as a plain bool — always send it so editing never silently deactivates
      payload.isActive = formModel.value.isActive;

      await updateBanner(editingId.value, payload);

      window.$message?.success('保存成功');
    }

    drawerShow.value = false;

    await loadList();
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
    <NCard :bordered="false" class="card-wrapper" title="轮播图管理">
    <template #header-extra>
      <NButton type="primary" @click="openCreate">新增 Banner</NButton>
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

  <NDrawer v-model:show="drawerShow" width="480">
    <NDrawerContent :title="editingId === null ? '新增 Banner' : '编辑 Banner'" closable>
      <NForm ref="formRef" :model="formModel" :rules="rules" label-placement="top">
        <NFormItem label="图片 URL" path="imageUrl">
          <NInput v-model:value="formModel.imageUrl" placeholder="图片 URL" clearable />
        </NFormItem>
        <NFormItem label="标题" path="title">
          <NInput v-model:value="formModel.title" placeholder="请输入标题" clearable />
        </NFormItem>
        <NFormItem label="跳转类型" path="linkType">
          <NSelect v-model:value="formModel.linkType" :options="linkTypeOptions" />
        </NFormItem>
        <NFormItem label="关联ID" path="linkId">
          <NInputNumber v-model:value="formModel.linkId" :min="0" class="w-full" placeholder="关联活动ID" />
        </NFormItem>
        <NFormItem label="排序" path="sortOrder">
          <NInputNumber v-model:value="formModel.sortOrder" :min="0" class="w-full" placeholder="数字越小越靠前" />
        </NFormItem>
        <NFormItem v-if="editingId !== null" label="上线状态" path="isActive">
          <NSwitch v-model:value="formModel.isActive" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="drawerShow = false">取消</NButton>
          <NButton type="primary" :loading="drawerLoading" @click="handleSave">保存</NButton>
        </NSpace>
      </template>
    </NDrawerContent>
  </NDrawer>
  </div>
</template>

<style scoped></style>
