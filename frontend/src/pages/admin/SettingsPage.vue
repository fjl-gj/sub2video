<script setup lang="ts">
import { Bell, LockKeyhole, Mail, Save, Send, Server } from 'lucide-vue-next'
import { onMounted, ref } from 'vue'
import {
  fetchAdminSettings,
  testAdminSMTP,
  updateAdminSettings,
  type AdminSettings,
} from '@/api/admin'
import { useAppStore } from '@/stores/app'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const app = useAppStore()
const settings = ref<AdminSettings>({
  smtpHost: '',
  smtpPort: '587',
  smtpUsername: '',
  smtpFrom: '',
  smtpFromName: '',
  smtpUseTls: true,
  smtpPasswordConfigured: false,
  emailVerifyEnabled: false,
  passwordResetEnabled: false,
  registrationEnabled: true,
  frontendUrl: '',
  inputRetentionDays: '7',
  resultRetentionDays: '30',
  notifyTask: true,
  auditEnabled: true,
})
const smtpPassword = ref('')
const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const error = ref('')

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    settings.value = await fetchAdminSettings()
  } catch (e: unknown) {
    error.value = apiErrorMessage(e, '加载系统设置失败')
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  error.value = ''
  try {
    const { smtpPasswordConfigured: _configured, ...payload } = settings.value
    const next = await updateAdminSettings({
      ...payload,
      ...(smtpPassword.value.trim() ? { smtpPassword: smtpPassword.value.trim() } : {}),
    })
    settings.value = next
    smtpPassword.value = ''
    app.notify('系统设置已保存')
  } catch (e: unknown) {
    error.value = apiErrorMessage(e, '保存系统设置失败')
  } finally {
    saving.value = false
  }
}

async function testSMTP() {
  testing.value = true
  error.value = ''
  try {
    await testAdminSMTP()
    app.notify('SMTP 连接测试成功')
  } catch (e: unknown) {
    error.value = apiErrorMessage(e, 'SMTP 连接测试失败')
  } finally {
    testing.value = false
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="系统设置">
      <template #description>配置邮件发送与注册邮箱验证。</template>
      <template #actions>
        <S2VButton variant="primary" :loading="saving" :disabled="loading" @click="save"><Save class="h-3.5 w-3.5" />保存设置</S2VButton>
      </template>
    </S2VPageHeader>

    <p v-if="error" class="mb-5 text-xs text-s2v-danger">{{ error }}</p>

    <div class="grid gap-5 lg:grid-cols-2">
      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">SMTP 邮件服务</h2>
            <p class="mt-1 text-xs text-s2v-muted">用于发送注册验证码和密码重置邮件。</p>
          </div>
          <Server class="h-4 w-4 text-s2v-primary" />
        </div>
        <div class="s2v-panel-body grid gap-4">
          <div class="grid gap-4 sm:grid-cols-2">
            <S2VInput v-model="settings.smtpHost" label="SMTP 主机" placeholder="smtp.example.com" :disabled="loading" />
            <S2VInput v-model="settings.smtpPort" label="端口" type="number" placeholder="587" :disabled="loading" />
          </div>
          <S2VInput v-model="settings.smtpUsername" label="用户名" placeholder="name@example.com" :disabled="loading" />
          <S2VInput v-model="smtpPassword" label="密码" type="password" :placeholder="settings.smtpPasswordConfigured ? '留空以保持现有密码' : 'SMTP 密码'" :disabled="loading">
            <template #prefix><LockKeyhole class="h-4 w-4 text-s2v-muted" /></template>
          </S2VInput>
          <div class="grid gap-4 sm:grid-cols-2">
            <S2VInput v-model="settings.smtpFrom" label="发件邮箱" placeholder="name@example.com" :disabled="loading" />
            <S2VInput v-model="settings.smtpFromName" label="发件人名称" placeholder="Sub2Video" :disabled="loading" />
          </div>
          <label class="flex items-center justify-between gap-3 text-xs">
            <span class="flex items-center gap-2"><LockKeyhole class="h-4 w-4 text-s2v-muted" />使用 TLS 加密连接</span>
            <input v-model="settings.smtpUseTls" type="checkbox" class="h-4 w-4 accent-s2v-primary" :disabled="loading" />
          </label>
          <S2VButton :loading="testing" :disabled="loading || testing" @click="testSMTP"><Send class="h-3.5 w-3.5" />测试 SMTP 连接</S2VButton>
        </div>
      </S2VCard>

      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">邮箱验证</h2>
            <p class="mt-1 text-xs text-s2v-muted">开启后，注册新账号前必须填写邮件验证码。</p>
          </div>
          <Mail class="h-4 w-4 text-s2v-primary" />
        </div>
        <div class="divide-y divide-s2v-line/70 px-4">
          <label class="flex items-center justify-between gap-3 py-4 text-xs">
            <span class="flex items-center gap-2">
              <Bell class="h-4 w-4 text-s2v-muted" />
              <span>
                <strong class="block">注册邮箱验证</strong>
                <span class="mt-1 block text-[11px] text-s2v-muted">新用户需验证邮箱后才能完成注册</span>
              </span>
            </span>
            <input v-model="settings.emailVerifyEnabled" type="checkbox" class="h-4 w-4 accent-s2v-primary" :disabled="loading" />
          </label>
        </div>
      </S2VCard>
    </div>
  </section>
</template>
