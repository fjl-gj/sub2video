<script setup lang="ts">
import { ref } from 'vue'
import AppHeader from '@/widgets/AppHeader.vue'
import MobileNav from '@/widgets/MobileNav.vue'
import SidebarNav from '@/widgets/SidebarNav.vue'
import ToastHost from '@/widgets/ToastHost.vue'

const menuOpen = ref(false)
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
