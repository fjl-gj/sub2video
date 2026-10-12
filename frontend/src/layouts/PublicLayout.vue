<script setup lang="ts">
import S2VLogo from '@/shared/ui/S2VLogo.vue'
import { useAuthStore } from '@/stores/auth'
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import '@/features/guides/guides.css'
const auth = useAuthStore()
const route = useRoute()
const isProductHome = computed(() => route.path === '/')
</script>
<template>
  <div class="s2v-public" :class="{ 's2v-public-home': isProductHome }">
    <header class="s2v-public-header">
      <RouterLink to="/" aria-label="Sub2Video 产品首页"><S2VLogo :size="isProductHome ? 'lg' : 'md'" /></RouterLink>
      <nav aria-label="产品导航">
        <RouterLink to="/#modes">创作方式</RouterLink>
        <RouterLink to="/guides/coast-letter">镜头教程</RouterLink>
        <RouterLink to="/app/tickets">帮助</RouterLink>
      </nav>
      <RouterLink :class="isProductHome ? 's2v-public-home-login' : 's2v-guide-link-button'" :to="auth.isAuthenticated ? '/app/overview' : '/auth/login'">{{ auth.isAuthenticated ? '我的工作台' : isProductHome ? '登录' : '登录 / 开始创作' }}</RouterLink>
    </header>
    <main class="s2v-public-main"><RouterView /></main>
    <footer class="s2v-public-footer">
      <div><S2VLogo :size="isProductHome ? 'lg' : 'sm'" /><p v-if="!isProductHome">从一个镜头开始，留下你的故事。</p></div>
      <nav aria-label="网站信息">
        <RouterLink to="/guides/coast-letter">{{ isProductHome ? '镜头教程' : '制作教程' }}</RouterLink>
        <RouterLink :to="isProductHome ? '/app/tickets' : '/app/create'">{{ isProductHome ? '帮助' : '进入创作' }}</RouterLink>
        <RouterLink to="/auth/terms">服务条款</RouterLink>
        <RouterLink to="/auth/privacy">隐私政策</RouterLink>
      </nav>
    </footer>
  </div>
</template>
