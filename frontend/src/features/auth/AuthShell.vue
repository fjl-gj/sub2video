<script setup lang="ts">
import { ArrowUpRight } from 'lucide-vue-next'
import AuthShowcase from './AuthShowcase.vue'
import S2VLogo from '@/shared/ui/S2VLogo.vue'
import ToastHost from '@/widgets/ToastHost.vue'
import './auth.css'
</script>

<template>
  <div class="s2v-auth-experience">
    <header class="s2v-auth-header">
      <RouterLink to="/" aria-label="Sub2Video 产品首页"><S2VLogo /></RouterLink>
      <span class="s2v-auth-header-caption">为每一个想讲故事的人</span>
      <RouterLink class="s2v-auth-home-link" to="/">探索 Sub2Video<ArrowUpRight :size="15" /></RouterLink>
    </header>
    <main class="s2v-auth-layout">
      <AuthShowcase />
      <section class="s2v-auth-panel" aria-label="账号入口"><slot /></section>
    </main>
    <footer class="s2v-auth-footer"><span>Sub2Video · 创作者的视频工作台</span><nav aria-label="法律信息"><RouterLink to="/auth/terms">服务条款</RouterLink><RouterLink to="/auth/privacy">隐私政策</RouterLink></nav></footer>
    <ToastHost />
  </div>
</template>
