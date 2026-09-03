<script setup lang="ts">
import { Bell, KeyRound, LogOut, Mail, MonitorSmartphone, ShieldCheck, UserRound } from 'lucide-vue-next'
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VChoice from '@/shared/ui/S2VChoice.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const router = useRouter()
const app = useAppStore()
const auth = useAuthStore()
const profileName = computed(() => auth.user?.displayName || auth.user?.workspaceName || app.profile.name)
const profileEmail = computed(() => auth.user?.email || app.profile.email)

async function logout() {
  await auth.logout()
  await router.push('/auth/login')
}
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="个人资料">
      <template #description>管理账号信息、登录安全和设备，不占用创作主导航。</template>
      <template #actions>
        <S2VButton variant="ghost" @click="logout"><LogOut class="h-3.5 w-3.5" />退出登录</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="grid gap-5 lg:grid-cols-2">
      <S2VCard variant="padded">
        <div class="flex items-center gap-3">
          <span class="s2v-avatar s2v-avatar-lg">{{ profileName.slice(0, 1) || '—' }}</span>
          <div>
            <h2 class="text-base font-bold">{{ profileName || '未设置昵称' }}</h2>
            <p class="mt-1 text-xs text-s2v-muted">{{ profileEmail || '未设置邮箱' }}</p>
            <S2VBadge class="mt-2" tone="success">专业创作者</S2VBadge>
          </div>
        </div>
        <div class="mt-5 grid gap-3">
          <S2VInput v-model="app.profile.name" label="昵称" />
          <S2VInput v-model="app.profile.email" label="邮箱" type="email" />
          <S2VButton @click="app.saveProfile()">保存资料</S2VButton>
        </div>
      </S2VCard>

      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">账号安全</h2>
            <p class="mt-1 text-xs text-s2v-muted">密码、邮箱验证和登录保护。</p>
          </div>
          <ShieldCheck class="h-4 w-4 text-s2v-primary" />
        </div>
        <div class="s2v-panel-body grid gap-3">
          <div class="s2v-inset flex items-center justify-between gap-3">
            <span class="flex items-center gap-2 text-xs"><Mail class="h-3.5 w-3.5 text-s2v-muted" />邮箱已验证</span>
            <S2VButton size="sm" @click="router.push('/auth/verify-email')">重新验证</S2VButton>
          </div>
          <div class="s2v-inset flex items-center justify-between gap-3">
            <span class="flex items-center gap-2 text-xs"><KeyRound class="h-3.5 w-3.5 text-s2v-muted" />登录密码</span>
            <S2VButton size="sm" @click="router.push('/auth/reset-password')">修改密码</S2VButton>
          </div>
        </div>
      </S2VCard>

      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">登录设备</h2>
            <p class="mt-1 text-xs text-s2v-muted">下线不再使用的会话。</p>
          </div>
          <MonitorSmartphone class="h-4 w-4 text-s2v-muted" />
        </div>
        <div class="divide-y divide-s2v-line/70 px-4">
          <div v-for="device in app.devices" :key="device.id" class="flex items-center gap-3 py-4">
            <span class="s2v-icon-well"><UserRound class="h-4 w-4" /></span>
            <div class="min-w-0 flex-1">
              <strong class="block text-xs">{{ device.name }}</strong>
              <span class="mt-1 block text-[11px] text-s2v-muted">{{ device.meta }}</span>
            </div>
            <S2VBadge v-if="device.current" tone="success">当前</S2VBadge>
            <S2VButton v-else size="sm" @click="app.revokeDevice(device.id)">下线</S2VButton>
          </div>
        </div>
      </S2VCard>

      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">通知设置</h2>
            <p class="mt-1 text-xs text-s2v-muted">只保留影响创作和结算的提醒。</p>
          </div>
          <Bell class="h-4 w-4 text-s2v-muted" />
        </div>
        <div class="s2v-panel-body grid gap-2">
          <S2VChoice :selected="app.profile.notifyTask" @click="app.profile.notifyTask = !app.profile.notifyTask">任务完成通知</S2VChoice>
          <S2VChoice :selected="app.profile.notifyBalance" @click="app.profile.notifyBalance = !app.profile.notifyBalance">余额变动通知</S2VChoice>
          <S2VChoice :selected="app.profile.notifyTicket" @click="app.profile.notifyTicket = !app.profile.notifyTicket">工单回复通知</S2VChoice>
        </div>
      </S2VCard>
    </div>
  </section>
</template>
