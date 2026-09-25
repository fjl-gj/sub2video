<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { fetchWallet } from '@/api/client'
import AppHeader from '@/widgets/AppHeader.vue'
import MobileNav from '@/widgets/MobileNav.vue'
import SidebarNav from '@/widgets/SidebarNav.vue'
import ToastHost from '@/widgets/ToastHost.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const menuOpen = ref(false)
const app = useAppStore()
const auth = useAuthStore()

async function syncWalletBalance() {
  try {
    const wallet = await fetchWallet()
    app.balance = wallet.balance
    auth.setBalance(wallet.balance)
  } catch {
    // The page can still render from the authenticated user or cached balance.
  }
}

onMounted(() => {
  void syncWalletBalance()
})
</script>

<template>
  <div class="s2v-shell">
    <AppHeader surface="client" :on-menu="() => menuOpen = !menuOpen" />
    <div class="s2v-main">
      <SidebarNav surface="client" />
      <SidebarNav v-if="menuOpen" surface="client" overlay />
      <main class="s2v-content"><RouterView /></main>
    </div>
    <MobileNav /><ToastHost />
  </div>
</template>
