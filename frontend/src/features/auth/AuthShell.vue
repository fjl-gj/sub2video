<script setup lang="ts">
import AuthShowcase from './AuthShowcase.vue'
import S2VLogo from '@/shared/ui/S2VLogo.vue'
import ToastHost from '@/widgets/ToastHost.vue'
import './auth.css'
</script>

<template>
  <div class="s2v-auth-experience s2v-auth-cinematic">
    <AuthShowcase>
      <nav aria-label="法律信息">
        <RouterLink to="/auth/terms">服务条款</RouterLink>
        <RouterLink to="/auth/privacy">隐私政策</RouterLink>
      </nav>
    </AuthShowcase>
    <header class="s2v-auth-header">
      <RouterLink to="/" aria-label="Sub2Video 产品首页"><S2VLogo /></RouterLink>
    </header>
    <main class="s2v-auth-layout">
      <section class="s2v-auth-panel" aria-label="账号入口"><slot /></section>
    </main>
    <ToastHost />
  </div>
</template>
