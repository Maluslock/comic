<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue';
import type { DataTableColumns, FormInst, FormRules, PaginationProps, SelectOption } from 'naive-ui';
import { NAvatar, NButton, NImage, NSpace, NTag } from 'naive-ui';
import {
  createTag,
  deleteReview,
  deleteTag,
  fetchAdminWorks,
  fetchReviews,
  fetchTags,
  mergeTags,
  setWorkStatus,
  updateTag
} from '@/service/api/admin';
import { getAdminApiErrorMessage } from '@/service/request/admin';

const activeTab = ref<'works' | 'reviews' | 'tags'>('works');

const WORK_STATUS_NAMES: Record<string, string> = {
  active: '上架',
  down: '下架'
};

function formatTime(value: string) {
  if (!value) {
    return '—';
  }

  return value.replace('T', ' ').slice(0, 16);
}

/* ---------- 作品 tab ---------- */

const worksLoading = ref(false);
const worksList = ref<Api.Admin.AdminWork[]>([]);
const worksTotal = ref(0);
const worksPage = ref(1);
const worksPageSize = ref(10);
const worksStatus = ref<'all' | Api.Admin.WorkStatus>('all');

const worksStatusOptions: SelectOption[] = [
  { label: '全部', value: 'all' },
  { label: '上架', value: 'active' },
  { label: '下架', value: 'down' }
];

const worksPagination = computed<PaginationProps>(() => ({
  page: worksPage.value,
  pageSize: worksPageSize.value,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  showQuickJumper: true,
  itemCount: worksTotal.value,
  prefix: ({ itemCount }) => `共 ${itemCount ?? 0} 条`
}));

const worksColumns: DataTableColumns<Api.Admin.AdminWork> = [
  {
    title: '缩略图',
    key: 'thumbnail',
    width: 130,
    render(row) {
      return h(NImage, {
        width: 100,
        height: 70,
        objectFit: 'cover',
        src: row.images[0],
        fallbackSrc: 'https://picsum.photos/seed/work-fallback/100/70',
        style: 'border-radius: 4px;'
      });
    }
  },
  {
    title: '标题',
    key: 'title',
    minWidth: 200,
    render(row) {
      return h('span', null, row.title || '—');
    }
  },
  {
    title: '摄影师',
    key: 'photographerName',
    minWidth: 140,
    render(row) {
      return h('span', null, row.photographerName || '—');
    }
  },
  {
    title: '状态',
    key: 'status',
    width: 90,
    render(row) {
      return h(
        NTag,
        {
          size: 'small',
          type: row.status === 'active' ? 'success' : 'error',
          round: true,
          bordered: false
        },
        { default: () => WORK_STATUS_NAMES[row.status] ?? row.status }
      );
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
    width: 100,
    fixed: 'right',
    render(row) {
      const isActive = row.status === 'active';

      return h(
        NButton,
        {
          size: 'small',
          type: isActive ? 'error' : 'primary',
          secondary: true,
          onClick: () => handleToggleWork(row)
        },
        { default: () => (isActive ? '下架' : '恢复') }
      );
    }
  }
];

async function loadWorks() {
  worksLoading.value = true;

  try {
    const { list, total } = await fetchAdminWorks({
      status: worksStatus.value === 'all' ? undefined : worksStatus.value,
      page: worksPage.value,
      pageSize: worksPageSize.value
    });

    worksList.value = list;
    worksTotal.value = total;
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    worksLoading.value = false;
  }
}

function handleWorksFilterChange() {
  worksPage.value = 1;
  loadWorks();
}

function handleWorksPageChange(page: number) {
  worksPage.value = page;
  loadWorks();
}

function handleWorksPageSizeChange(pageSize: number) {
  worksPageSize.value = pageSize;
  worksPage.value = 1;
  loadWorks();
}

function handleToggleWork(row: Api.Admin.AdminWork) {
  const isActive = row.status === 'active';
  const next: Api.Admin.WorkStatus = isActive ? 'down' : 'active';

  window.$dialog?.warning({
    title: isActive ? '下架作品' : '恢复作品',
    content: `${isActive ? '下架' : '恢复'}作品「${row.title}」？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await setWorkStatus(row.id, next);

        window.$message?.success(isActive ? '已下架' : '已恢复');

        await loadWorks();
      } catch (error) {
        window.$message?.error(getAdminApiErrorMessage(error));
      }
    }
  });
}

/* ---------- 评论 tab ---------- */

const reviewsLoading = ref(false);
const reviewsList = ref<Api.Admin.AdminReview[]>([]);
const reviewsTotal = ref(0);
const reviewsPage = ref(1);
const reviewsPageSize = ref(10);
const reviewKeyword = ref('');

const reviewsPagination = computed<PaginationProps>(() => ({
  page: reviewsPage.value,
  pageSize: reviewsPageSize.value,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  showQuickJumper: true,
  itemCount: reviewsTotal.value,
  prefix: ({ itemCount }) => `共 ${itemCount ?? 0} 条`
}));

const reviewsColumns: DataTableColumns<Api.Admin.AdminReview> = [
  {
    title: '头像',
    key: 'avatar',
    width: 72,
    render(row) {
      return h(NAvatar, {
        round: true,
        size: 40,
        src: row.userAvatar,
        fallbackSrc: 'https://api.dicebear.com/7.x/avataaars/svg?seed=fallback'
      });
    }
  },
  {
    title: '用户名',
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
    title: '评分',
    key: 'rating',
    width: 80,
    render(row) {
      return h(
        NTag,
        { size: 'small', type: 'warning', round: true, bordered: false },
        { default: () => `${row.rating} 分` }
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
        { size: 'small', type: 'error', secondary: true, onClick: () => handleDeleteReview(row) },
        { default: () => '删除' }
      );
    }
  }
];

async function loadReviews() {
  reviewsLoading.value = true;

  try {
    const { list, total } = await fetchReviews({
      keyword: reviewKeyword.value.trim() || undefined,
      page: reviewsPage.value,
      pageSize: reviewsPageSize.value
    });

    reviewsList.value = list;
    reviewsTotal.value = total;
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    reviewsLoading.value = false;
  }
}

function handleReviewSearch() {
  reviewsPage.value = 1;
  loadReviews();
}

function handleReviewReset() {
  reviewKeyword.value = '';
  reviewsPage.value = 1;
  loadReviews();
}

function handleReviewsPageChange(page: number) {
  reviewsPage.value = page;
  loadReviews();
}

function handleReviewsPageSizeChange(pageSize: number) {
  reviewsPageSize.value = pageSize;
  reviewsPage.value = 1;
  loadReviews();
}

function handleDeleteReview(row: Api.Admin.AdminReview) {
  window.$dialog?.warning({
    title: '删除评论',
    content: `确定删除用户「${row.userName || '—'}」的评论？删除后不可恢复。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteReview(row.id);

        window.$message?.success('评论已删除');

        await loadReviews();
      } catch (error) {
        window.$message?.error(getAdminApiErrorMessage(error));
      }
    }
  });
}

/* ---------- 标签 tab ---------- */

const tagsLoading = ref(false);
const tagsList = ref<Api.Admin.AdminTag[]>([]);

const tagsColumns: DataTableColumns<Api.Admin.AdminTag> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '名称', key: 'name', minWidth: 160 },
  { title: '使用量', key: 'usageCount', width: 100 },
  {
    title: '操作',
    key: 'actions',
    width: 200,
    fixed: 'right',
    render(row) {
      return h(NSpace, null, {
        default: () => [
          h(
            NButton,
            { size: 'small', type: 'primary', secondary: true, onClick: () => openTagEdit(row) },
            { default: () => '编辑' }
          ),
          h(
            NButton,
            { size: 'small', type: 'warning', secondary: true, onClick: () => openTagMerge(row) },
            { default: () => '合并' }
          ),
          h(
            NButton,
            { size: 'small', type: 'error', secondary: true, onClick: () => handleDeleteTag(row) },
            { default: () => '删除' }
          )
        ]
      });
    }
  }
];

async function loadTags() {
  tagsLoading.value = true;

  try {
    tagsList.value = await fetchTags();
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    tagsLoading.value = false;
  }
}

const tagModalShow = ref(false);
const tagModalLoading = ref(false);
const tagEditingId = ref<number | null>(null);

const mergeModalShow = ref(false);
const mergeSourceTag = ref<Api.Admin.AdminTag | null>(null);
const mergeTargetId = ref<number | null>(null);
const mergeLoading = ref(false);

const mergeTargetOptions = computed<SelectOption[]>(() =>
  tagsList.value
    .filter(tag => tag.id !== mergeSourceTag.value?.id)
    .map(tag => ({ label: `${tag.name}（使用量 ${tag.usageCount}）`, value: tag.id }))
);

const tagFormRef = ref<FormInst | null>(null);

const tagFormModel = ref({ name: '' });

const tagRules: FormRules = {
  name: [{ required: true, message: '请输入标签名称', trigger: 'blur' }]
};

function openTagCreate() {
  tagEditingId.value = null;
  tagFormModel.value = { name: '' };
  tagModalShow.value = true;
}

function openTagEdit(row: Api.Admin.AdminTag) {
  tagEditingId.value = row.id;
  tagFormModel.value = { name: row.name };
  tagModalShow.value = true;
}

function openTagMerge(row: Api.Admin.AdminTag) {
  mergeSourceTag.value = row;
  mergeTargetId.value = null;
  mergeModalShow.value = true;
}

async function handleTagMerge() {
  const source = mergeSourceTag.value;
  const target = mergeTargetId.value;

  if (!source || target === null) {
    window.$message?.warning('请选择目标标签');
    return;
  }

  mergeLoading.value = true;

  try {
    await mergeTags(source.id, target);

    window.$message?.success('标签已合并');

    mergeModalShow.value = false;

    await loadTags();
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    mergeLoading.value = false;
  }
}

async function handleTagSave() {
  await tagFormRef.value?.validate();

  tagModalLoading.value = true;

  try {
    const payload = { name: tagFormModel.value.name.trim() };

    if (tagEditingId.value === null) {
      await createTag(payload);

      window.$message?.success('标签已新增');
    } else {
      await updateTag(tagEditingId.value, payload);

      window.$message?.success('标签已更新');
    }

    tagModalShow.value = false;

    await loadTags();
  } catch (error) {
    window.$message?.error(getAdminApiErrorMessage(error));
  } finally {
    tagModalLoading.value = false;
  }
}

function handleDeleteTag(row: Api.Admin.AdminTag) {
  window.$dialog?.warning({
    title: '删除标签',
    content: `确定删除标签「${row.name}」？若标签仍被作品使用将无法删除。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteTag(row.id);

        window.$message?.success('标签已删除');

        await loadTags();
      } catch (error) {
        const message = getAdminApiErrorMessage(error);

        window.$message?.error(message || '删除失败，标签可能仍在使用中');
      }
    }
  });
}

/* ---------- tab switch ---------- */

function handleTabChange(tab: 'works' | 'reviews' | 'tags') {
  worksPage.value = 1;
  reviewsPage.value = 1;

  if (tab === 'works') {
    loadWorks();
  } else if (tab === 'reviews') {
    loadReviews();
  } else {
    loadTags();
  }
}

onMounted(() => {
  loadWorks();
});
</script>

<template>
  <div>
    <NCard :bordered="false" class="card-wrapper" title="内容管理">
      <NTabs v-model:value="activeTab" type="line" animated @update:value="handleTabChange">
        <NTab name="works">作品</NTab>
        <NTab name="reviews">评论</NTab>
        <NTab name="tags">标签</NTab>
      </NTabs>
      <div v-if="activeTab === 'works'">
        <NSpace class="mb-16px" size="medium" align="center">
          <NSelect
            v-model:value="worksStatus"
            :options="worksStatusOptions"
            class="w-130px"
            @update:value="handleWorksFilterChange"
          />
        </NSpace>
        <NDataTable
          remote
          :columns="worksColumns"
          :data="worksList"
          :loading="worksLoading"
          :bordered="false"
          :single-line="false"
          :scroll-x="1100"
          :pagination="worksPagination"
          @update:page="handleWorksPageChange"
          @update:page-size="handleWorksPageSizeChange"
        />
      </div>
      <div v-if="activeTab === 'reviews'">
        <NSpace class="mb-16px" size="medium" align="center" wrap>
          <NInput
            v-model:value="reviewKeyword"
            placeholder="搜索用户名 / 内容"
            clearable
            class="w-220px"
            @keyup.enter="handleReviewSearch"
          />
          <NButton type="primary" @click="handleReviewSearch">查询</NButton>
          <NButton @click="handleReviewReset">重置</NButton>
        </NSpace>
        <NDataTable
          remote
          :columns="reviewsColumns"
          :data="reviewsList"
          :loading="reviewsLoading"
          :bordered="false"
          :single-line="false"
          :scroll-x="1100"
          :pagination="reviewsPagination"
          @update:page="handleReviewsPageChange"
          @update:page-size="handleReviewsPageSizeChange"
        />
      </div>
      <div v-if="activeTab === 'tags'">
        <NSpace class="mb-16px" size="medium" align="center">
          <NButton type="primary" @click="openTagCreate">新增标签</NButton>
        </NSpace>
        <NDataTable
          :columns="tagsColumns"
          :data="tagsList"
          :loading="tagsLoading"
          :bordered="false"
          :single-line="false"
          :scroll-x="600"
          :pagination="false"
        />
      </div>
    </NCard>

    <NModal v-model:show="tagModalShow">
      <NCard
        style="width: 440px"
        :bordered="false"
        size="huge"
        role="dialog"
        aria-modal="true"
        :title="tagEditingId === null ? '新增标签' : '编辑标签'"
      >
        <NForm ref="tagFormRef" :model="tagFormModel" :rules="tagRules" label-placement="top">
          <NFormItem label="标签名称" path="name">
            <NInput v-model:value="tagFormModel.name" placeholder="请输入标签名称" clearable />
          </NFormItem>
        </NForm>
        <template #footer>
          <NSpace justify="end">
            <NButton @click="tagModalShow = false">取消</NButton>
            <NButton type="primary" :loading="tagModalLoading" @click="handleTagSave">保存</NButton>
          </NSpace>
        </template>
      </NCard>
    </NModal>

    <NModal v-model:show="mergeModalShow">
      <NCard
        style="width: 440px"
        :bordered="false"
        size="huge"
        role="dialog"
        aria-modal="true"
        title="合并标签"
      >
        <NForm label-placement="top">
          <NFormItem label="源标签">
            <NInput :value="mergeSourceTag?.name" disabled />
          </NFormItem>
          <NFormItem label="合并到">
            <NSelect
              v-model:value="mergeTargetId"
              :options="mergeTargetOptions"
              placeholder="请选择目标标签"
              filterable
            />
          </NFormItem>
        </NForm>
        <template #footer>
          <NSpace justify="end">
            <NButton @click="mergeModalShow = false">取消</NButton>
            <NButton type="primary" :loading="mergeLoading" @click="handleTagMerge">合并</NButton>
          </NSpace>
        </template>
      </NCard>
    </NModal>
  </div>
</template>

<style scoped></style>
