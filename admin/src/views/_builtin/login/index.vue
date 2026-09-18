<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import type { FormInst, FormRules } from 'naive-ui';
import { getPaletteColorByNumber, mixColor } from '@sa/color';
import { useAuthStore } from '@/store/modules/auth';
import { useThemeStore } from '@/store/modules/theme';
import { $t } from '@/locales';

const authStore = useAuthStore();
const themeStore = useThemeStore();

const formRef = ref<FormInst>();

const model = reactive({
  userName: '',
  password: ''
});

const rules: FormRules = {
  userName: [{ required: true, message: $t('form.userName.required'), trigger: 'blur' }],
  password: [{ required: true, message: $t('form.pwd.required'), trigger: 'blur' }]
};

async function handleSubmit() {
  try {
    await formRef.value?.validate();
  } catch {
    return;
  }

  authStore.login(model.userName, model.password);
}

const bgThemeColor = computed(() =>
  themeStore.darkMode ? getPaletteColorByNumber(themeStore.themeColor, 600) : themeStore.themeColor
);

const bgColor = computed(() => {
  const COLOR_WHITE = '#ffffff';

  const ratio = themeStore.darkMode ? 0.5 : 0.2;

  return mixColor(COLOR_WHITE, themeStore.themeColor, ratio);
});
</script>

<template>
  <div class="relative size-full flex-center overflow-hidden" :style="{ backgroundColor: bgColor }">
    <WaveBg :theme-color="bgThemeColor" />
    <NCard :bordered="false" class="relative z-4 w-auto rd-12px">
      <div class="w-400px lt-sm:w-300px">
        <header class="flex-y-center justify-between">
          <SystemLogo class="size-64px lt-sm:size-48px" />
          <h3 class="text-28px text-primary font-500 lt-sm:text-22px">{{ $t('system.title') }}</h3>
          <ThemeSchemaSwitch
            :theme-schema="themeStore.themeScheme"
            :show-tooltip="false"
            class="text-20px lt-sm:text-18px"
            @switch="themeStore.toggleThemeScheme"
          />
        </header>
        <main class="pt-24px">
          <h3 class="text-18px text-primary font-medium">管理后台登录</h3>
          <NForm ref="formRef" :model="model" :rules="rules" class="pt-24px" @keyup.enter="handleSubmit">
            <NFormItem path="userName" label="用户名">
              <NInput v-model:value="model.userName" :placeholder="$t('page.login.common.userNamePlaceholder')" clearable />
            </NFormItem>
            <NFormItem path="password" label="密码">
              <NInput
                v-model:value="model.password"
                type="password"
                show-password-on="click"
                :placeholder="$t('page.login.common.passwordPlaceholder')"
              />
            </NFormItem>
            <NAlert type="info" class="mb-16px" :show-icon="false">默认账号 admin / admin123</NAlert>
            <NButton type="primary" size="large" block :loading="authStore.loginLoading" @click="handleSubmit">
              登 录
            </NButton>
          </NForm>
        </main>
      </div>
    </NCard>
  </div>
</template>

<style scoped></style>
