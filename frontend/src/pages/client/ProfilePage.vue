<script setup lang="ts">
import { Bell, KeyRound, LogOut, Mail, MonitorSmartphone, ShieldCheck } from 'lucide-vue-next'
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const router = useRouter()
const auth = useAuthStore()
const profileName = computed(() => auth.user?.displayName || auth.user?.workspaceName || '')
const profileEmail = computed(() => auth.user?.email || '')

async function logout() {
  await auth.logout()
}
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="个人资料">
      <template #description>查看当前登录账号，资料编辑与设备管理暂未开放。</template>
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
          </div>
        </div>
        <div class="mt-5 grid gap-3">
          <S2VInput :model-value="profileName" label="昵称" readonly />
          <S2VInput :model-value="profileEmail" label="邮箱" type="email" readonly />
          <p class="text-xs text-s2v-muted">资料暂不支持在线修改。如需协助，请前往帮助与工单。</p>
          <S2VButton @click="router.push('/app/tickets')">帮助与工单</S2VButton>
        </div>
      </S2VCard>

      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">账号安全</h2>
            <p class="mt-1 text-xs text-s2v-muted">账号信息以实际登录记录为准。</p>
          </div>
          <ShieldCheck class="h-4 w-4 text-s2v-primary" />
        </div>
        <div class="s2v-panel-body grid gap-3">
          <div class="s2v-inset flex items-center justify-between gap-3">
            <span class="flex items-center gap-2 text-xs"><Mail class="h-3.5 w-3.5 text-s2v-muted" />邮箱验证状态暂不可查询</span>
          </div>
          <div class="s2v-inset flex items-center justify-between gap-3">
            <span class="flex items-center gap-2 text-xs"><KeyRound class="h-3.5 w-3.5 text-s2v-muted" />登录密码</span>
            <S2VButton size="sm" @click="router.push('/auth/forgot-password')">重置密码</S2VButton>
          </div>
        </div>
      </S2VCard>

      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">登录设备</h2>
            <p class="mt-1 text-xs text-s2v-muted">设备与会话管理暂未开放。</p>
          </div>
          <MonitorSmartphone class="h-4 w-4 text-s2v-muted" />
        </div>
        <div class="s2v-panel-body">
          <p class="text-xs text-s2v-muted">退出登录会结束当前浏览器的登录状态，此处暂不展示其他设备。</p>
        </div>
      </S2VCard>

      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">通知设置</h2>
            <p class="mt-1 text-xs text-s2v-muted">通知偏好设置暂未开放。</p>
          </div>
          <Bell class="h-4 w-4 text-s2v-muted" />
        </div>
        <div class="s2v-panel-body grid gap-2">
          <p class="text-xs text-s2v-muted">请在任务、钱包和帮助与工单页面查看最新状态。</p>
        </div>
      </S2VCard>
    </div>
  </section>
</template>
