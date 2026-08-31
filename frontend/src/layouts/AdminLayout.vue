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
    <AppHeader surface="admin" :on-menu="() => menuOpen = !menuOpen" />
    <div class="s2v-main">
      <SidebarNav surface="admin" />
      <SidebarNav v-if="menuOpen" surface="admin" class="!fixed inset-y-[54px] left-0 z-40 !block w-[224px] shadow-subtle md:hidden" />
      <main class="s2v-content"><RouterView /></main>
    </div>
    <MobileNav surface="admin" /><ToastHost />
  </div>
</template>
