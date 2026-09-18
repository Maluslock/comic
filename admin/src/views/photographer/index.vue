<script setup lang="ts">
import { h, onMounted, ref } from 'vue';
import type { DataTableColumns } from 'naive-ui';
import { NAvatar, NButton, NDescriptions, NDescriptionsItem, NDrawer, NDrawerContent, NTag } from 'naive-ui';
import { fetchAdminPhotographerDetail, fetchAdminPhotographers, updateAdminPhotographerCertified, exportAdminCsv } from '@/service/api/admin';
import { getAdminApiErrorMessage } from '@/service/request/admin';

const loading = ref(false);
const exporting = ref(false);
const list = ref<Api.Admin.Photographer[]>([]);

async function handleExport() {
  exporting.value = true;
  try {
    const params = filter.value === 'all' ? undefined : { certified: String(filter.value === 'certified') };
    await exportAdminCsv('photographers', params);
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    exporting.value = false;
  }
}
const filter = ref<'all' | 'certified' | 'uncertified'>('all');

const detailVisible = ref(false);
const detailLoading = ref(false);
const detail = ref<Api.Admin.PhotographerDetail | null>(null);

const filterOptions = [
  { label: '全部', value: 'all' },
  { label: '已认证', value: 'certified' },
  { label: '未认证', value: 'uncertified' }
];

const MODE_NAMES: Record<string, string> = {
  free: '互勉',
  pay: '收费',
  both: '互勉 · 收费'
};

const MODE_TAG_TYPES: Record<string, 'default' | 'info' | 'success' | 'warning' | 'error'> = {
  free: 'info',
  pay: 'warning',
  both: 'success'
};

const columns: DataTableColumns<Api.Admin.Photographer> = [
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
    minWidth: 120
  },
  {
    title: '城市',
    key: 'location',
    width: 100
  },
  {
    title: '评分',
    key: 'rating',
    width: 80,
    render(row) {
      return h('span', { class: 'font-600 text-amber-500' }, row.rating.toFixed(1));
    }
  },
  {
    title: '接单模式',
    key: 'mode',
    width: 120,
    render(row) {
      return h(
        NTag,
        { size: 'small', type: MODE_TAG_TYPES[row.mode] ?? 'default', bordered: false },
        { default: () => MODE_NAMES[row.mode] ?? row.mode }
      );
    }
  },
  {
    title: '接单数',
    key: 'orderCount',
    width: 90
  },
  {
    title: '认证状态',
    key: 'certified',
    width: 110,
    render(row) {
      return h(
        NTag,
        { size: 'small', type: row.certified ? 'warning' : 'default', round: true, bordered: false },
        { default: () => (row.certified ? '已认证' : '未认证') }
      );
    }
  },
  {
    title: '操作',
    key: 'actions',
    width: 180,
    fixed: 'right',
    render(row) {
      return h('div', { class: 'flex gap-8px' }, [
        h(NButton, { size: 'small', secondary: true, onClick: () => openDetail(row) }, { default: () => '详情' }),
        h(
          NButton,
          {
            size: 'small',
            type: row.certified ? 'error' : 'primary',
            secondary: true,
            onClick: () => handleToggleCertified(row)
          },
          { default: () => (row.certified ? '取消认证' : '通过认证') }
        )
      ]);
    }
  }
];

async function loadList() {
  loading.value = true;

  try {
    const certified = filter.value === 'all' ? undefined : filter.value === 'certified';

    list.value = await fetchAdminPhotographers(certified);
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    loading.value = false;
  }
}

async function handleToggleCertified(row: Api.Admin.Photographer) {
  const next = !row.certified;

  try {
    await updateAdminPhotographerCertified(row.id, next);

    window.$message?.success(next ? '已通过认证' : '已取消认证');

    if (filter.value !== 'all') {
      await loadList();
    } else {
      row.certified = next;
    }
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  }
}

function handleFilterChange() {
  loadList();
}

async function openDetail(row: Api.Admin.Photographer) {
  detailVisible.value = true;
  detailLoading.value = true;
  detail.value = null;

  try {
    detail.value = await fetchAdminPhotographerDetail(row.id);
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    detailLoading.value = false;
  }
}

onMounted(loadList);
</script>

<template>
  <div>
    <NCard :bordered="false" class="card-wrapper" title="摄影师审核">
      <template #header-extra>
        <div class="flex items-center gap-12px">
          <NRadioGroup v-model:value="filter" size="small" @update:value="handleFilterChange">
            <NRadioButton v-for="option in filterOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </NRadioButton>
          </NRadioGroup>
          <NButton size="small" secondary :loading="exporting" @click="handleExport">导出 CSV</NButton>
        </div>
      </template>
      <NDataTable
        :columns="columns"
        :data="list"
        :loading="loading"
        :bordered="false"
        :single-line="false"
        :scroll-x="1060"
      />
    </NCard>

    <NDrawer v-model:show="detailVisible" :width="440" placement="right">
      <NDrawerContent title="摄影师详情" closable>
        <NSpin :show="detailLoading">
          <NDescriptions v-if="detail" :column="1" label-placement="left" bordered size="small">
            <NDescriptionsItem label="ID">{{ detail.id }}</NDescriptionsItem>
            <NDescriptionsItem label="昵称">{{ detail.name }}</NDescriptionsItem>
            <NDescriptionsItem label="城市">{{ detail.location || '-' }}</NDescriptionsItem>
            <NDescriptionsItem label="评分">{{ detail.rating.toFixed(1) }}</NDescriptionsItem>
            <NDescriptionsItem label="接单模式">{{ MODE_NAMES[detail.mode] ?? detail.mode }}</NDescriptionsItem>
            <NDescriptionsItem label="接单数">{{ detail.orderCount }}</NDescriptionsItem>
            <NDescriptionsItem label="认证状态">{{ detail.certified ? '已认证' : '未认证' }}</NDescriptionsItem>
            <NDescriptionsItem label="手机号">{{ detail.userPhone || '-' }}</NDescriptionsItem>
            <NDescriptionsItem label="作品数">{{ detail.worksCount }}</NDescriptionsItem>
            <NDescriptionsItem label="评价数">{{ detail.reviewsCount }}</NDescriptionsItem>
            <NDescriptionsItem label="简介">{{ detail.description || '-' }}</NDescriptionsItem>
            <NDescriptionsItem label="互勉说明">{{ detail.mutualIntro || '-' }}</NDescriptionsItem>
          </NDescriptions>
        </NSpin>
      </NDrawerContent>
    </NDrawer>
  </div>
</template>

<style scoped></style>
