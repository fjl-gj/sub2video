<script setup lang="ts">
import { Bell, ChevronRight, KeyRound, LogOut, Mail, ShieldCheck, UserRound } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const router = useRouter()
const rows = [
  { title: '账号信息', desc: '昵称、头像和个人介绍', icon: UserRound },
  { title: '登录与安全', desc: '密码、邮箱验证和登录设备', icon: ShieldCheck },
  { title: '通知设置', desc: '任务完成、余额和工单提醒', icon: Bell },
]
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader badge="Account" title="个人资料">
      <template #description>管理你的账号信息和平台通知。</template>
      <template #actions>
        <S2VButton variant="ghost" @click="router.push('/auth/login')"><LogOut class="h-3.5 w-3.5" />退出登录</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="grid gap-5 lg:grid-cols-[minmax(0,.8fr)_minmax(0,1.2fr)]">
      <S2VCard variant="padded">
        <div class="flex items-center gap-3">
          <span class="s2v-avatar s2v-avatar-lg">林</span>
          <div>
            <h2 class="text-base font-bold">林默</h2>
            <p class="mt-1 text-xs text-s2v-muted">linmo@example.com</p>
            <S2VBadge class="mt-2" tone="success">专业创作者</S2VBadge>
          </div>
        </div>
        <div class="mt-5 grid gap-3 border-t border-s2v-line pt-4 text-xs">
          <div class="flex items-center gap-2 text-s2v-muted"><Mail class="h-3.5 w-3.5" />邮箱已验证</div>
          <div class="flex items-center gap-2 text-s2v-muted"><KeyRound class="h-3.5 w-3.5" />上次登录：今天 09:42</div>
        </div>
      </S2VCard>

      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">账户设置</h2>
            <p class="mt-1 text-xs text-s2v-muted">只保留影响创作体验的设置。</p>
          </div>
        </div>
        <div class="divide-y divide-s2v-line/70 px-4">
          <button v-for="row in rows" :key="row.title" class="flex w-full items-center gap-3 py-4 text-left" type="button" @click="router.push('/auth/verify-email')">
            <span class="s2v-icon-well"><component :is="row.icon" class="h-4 w-4" /></span>
            <span class="min-w-0 flex-1">
              <strong class="block text-xs">{{ row.title }}</strong>
              <span class="mt-1 block text-[11px] text-s2v-muted">{{ row.desc }}</span>
            </span>
            <ChevronRight class="h-4 w-4 text-s2v-muted" />
          </button>
        </div>
      </S2VCard>
    </div>
  </section>
</template>
