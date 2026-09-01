<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { Menu, Wallet } from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'
import S2VIconButton from '@/shared/ui/S2VIconButton.vue'
import S2VLogo from '@/shared/ui/S2VLogo.vue'

const props = defineProps<{ surface: 'client' | 'admin'; onMenu?: () => void }>()
const router = useRouter()
const app = useAppStore()
const isClient = computed(() => props.surface === 'client')
</script>

<template>
  <header class="sticky top-0 z-40 flex h-full items-center justify-between border-b border-s2v-line/80 bg-s2v-surface/95 px-4 shadow-xs backdrop-blur-md lg:px-6">
    <div class="flex min-w-0 items-center gap-3">
      <S2VIconButton class="md:hidden" label="打开菜单" @click="onMenu?.()"><Menu class="h-4 w-4" /></S2VIconButton>
      <RouterLink :to="isClient ? '/app/overview' : '/admin/overview'" class="flex min-w-0 items-center">
        <S2VLogo size="md" />
      </RouterLink>
    </div>
    <div class="flex items-center gap-3">
      <button v-if="isClient" type="button" class="s2v-mono hidden items-center gap-1.5 rounded-lg border border-s2v-primary/25 bg-s2v-primary-soft/80 px-3 py-1 text-xs font-bold text-s2v-primary-hover transition hover:bg-s2v-primary-soft md:flex" @click="router.push('/app/wallet')">
        <Wallet class="h-3.5 w-3.5" />¥ {{ app.balance.toFixed(2) }}
      </button>
      <button type="button" class="flex items-center gap-2.5 border-l border-s2v-line pl-2" :aria-label="isClient ? '打开个人资料' : '管理员账户'" @click="isClient ? router.push('/app/profile') : undefined">
        <span :class="['grid h-7 w-7 place-items-center rounded-full text-xs font-bold text-white', isClient ? 'bg-slate-900' : 'bg-emerald-900']">{{ isClient ? '李' : '管' }}</span>
        <strong class="hidden text-xs font-semibold text-s2v-ink sm:block">{{ isClient ? '李维工作室' : '超级管理员' }}</strong>
      </button>
    </div>
  </header>
</template>
