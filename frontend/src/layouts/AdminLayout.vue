<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import AppHeader from '@/widgets/AppHeader.vue'
import SidebarNav from '@/widgets/SidebarNav.vue'
import ToastHost from '@/widgets/ToastHost.vue'

const menuOpen = ref(false)
const route = useRoute()
watch(() => route.fullPath, () => { menuOpen.value = false })
</script>

<template>
  <div class="s2v-shell" @keydown.esc="menuOpen = false">
    <AppHeader surface="admin" :on-menu="() => menuOpen = !menuOpen" />
    <div class="s2v-main">
      <SidebarNav surface="admin" />
      <button v-if="menuOpen" class="s2v-drawer-backdrop" aria-label="关闭菜单" @click="menuOpen = false" />
      <SidebarNav v-if="menuOpen" surface="admin" overlay />
      <main class="s2v-content"><RouterView /></main>
    </div>
    <ToastHost />
  </div>
</template>
