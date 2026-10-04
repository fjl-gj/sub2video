<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { LogOut, Menu, UserRound, Wallet } from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { getAuthSession } from '@/shared/lib/auth-session'
import S2VIconButton from '@/shared/ui/S2VIconButton.vue'
import S2VLogo from '@/shared/ui/S2VLogo.vue'

const props = defineProps<{ surface: 'client' | 'admin'; onMenu?: () => void }>()
const router = useRouter()
const route = useRoute()
const app = useAppStore()
const auth = useAuthStore()
const isClient = computed(() => props.surface === 'client')
const balance = computed(() => auth.balance)
const label = computed(() => auth.displayName || (isClient.value ? '创作空间' : '管理员'))
const avatar = computed(() => (label.value ? label.value.slice(0, 1) : isClient.value ? '李' : '管'))

const dropdownOpen = ref(false)
const loggingOut = ref(false)
function toggleDropdown() { dropdownOpen.value = !dropdownOpen.value }
function closeDropdown() { dropdownOpen.value = false }
function goProfile() { closeDropdown(); router.push('/app/profile') }
async function doLogout() {
  if (loggingOut.value) return
  loggingOut.value = true
  closeDropdown()
  try {
    const pending = auth.logout()
    const endedSessionId = getAuthSession().id
    const revoked = await pending
    if (!revoked && getAuthSession().id === endedSessionId) app.notify('已退出本机，服务端会话注销暂未确认')
  } finally {
    loggingOut.value = false
  }
}
</script>

<template>
  <header class="s2v-header">
    <div class="flex min-w-0 items-center gap-3">
      <S2VIconButton class="md:hidden" label="打开菜单" @click="onMenu?.()"><Menu class="h-4 w-4" /></S2VIconButton>
      <RouterLink :to="isClient ? '/app/overview' : '/admin/overview'" class="flex min-w-0 items-center">
        <S2VLogo size="md" />
      </RouterLink>
      <div class="s2v-header-context"><span>{{ isClient ? '创作空间' : '运营管理' }}</span><span>/</span><strong>{{ route.meta.title || 'Sub2Video' }}</strong></div>
    </div>
    <div class="flex items-center gap-3">
      <button v-if="isClient" type="button" class="s2v-header-wallet" aria-label="查看创作余额" @click="router.push('/app/wallet')">
        <Wallet class="h-3.5 w-3.5" /><span class="s2v-header-wallet-label">创作余额</span><span class="s2v-mono">¥ {{ balance.toFixed(2) }}</span>
      </button>
      <div class="relative border-l border-s2v-line pl-2" @keydown.esc.stop="closeDropdown">
        <button type="button" class="flex items-center gap-2.5" aria-label="用户菜单" :aria-expanded="dropdownOpen" @click="toggleDropdown" @keydown.esc="closeDropdown">
          <span class="s2v-header-avatar">{{ avatar }}</span>
          <strong class="hidden text-xs font-semibold text-s2v-ink sm:block">{{ label }}</strong>
        </button>
        <div v-if="dropdownOpen" class="fixed inset-0 z-50" @click="closeDropdown" />
        <Transition enter-active-class="transition duration-100" enter-from-class="scale-95 opacity-0" enter-to-class="scale-100 opacity-100" leave-active-class="transition duration-75" leave-from-class="scale-100 opacity-100" leave-to-class="scale-95 opacity-0">
          <div v-if="dropdownOpen" class="absolute right-0 top-full z-50 mt-2 w-44 origin-top-right rounded-lg border border-s2v-line bg-s2v-surface py-1 shadow-lg">
            <button v-if="isClient" type="button" class="flex w-full items-center gap-2.5 px-3 py-2 text-xs text-s2v-ink hover:bg-s2v-surface-muted" @click="goProfile">
              <UserRound class="h-3.5 w-3.5 text-s2v-muted" />个人资料
            </button>
            <div class="mx-2 border-t border-s2v-line/60" />
            <button type="button" class="flex w-full items-center gap-2.5 px-3 py-2 text-xs text-s2v-danger hover:bg-s2v-surface-muted" :disabled="loggingOut" @click="doLogout">
              <LogOut class="h-3.5 w-3.5" />退出登录
            </button>
          </div>
        </Transition>
      </div>
    </div>
  </header>
</template>
