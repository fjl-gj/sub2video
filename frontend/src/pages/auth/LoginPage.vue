<script setup lang="ts">
import { LockKeyhole, Mail, ShieldCheck } from 'lucide-vue-next'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import AuthBrand from '@/widgets/AuthBrand.vue'

const router = useRouter()
const email = ref('linmo@example.com')
const password = ref('password')
const loading = ref(false)

function login() {
  loading.value = true
  window.setTimeout(() => {
    loading.value = false
    router.push('/app/overview')
  }, 350)
}
</script>

<template>
  <div class="grid gap-5">
    <div class="text-center">
      <AuthBrand class="mx-auto" />
      <h1 class="mt-4 text-xl font-bold">登录 Sub2Video</h1>
      <p class="mt-1.5 text-xs text-s2v-muted">继续你的创作工作流</p>
    </div>
    <S2VCard variant="padded">
      <div class="grid gap-4">
        <S2VInput v-model="email" label="邮箱" type="email">
          <template #prefix><Mail class="h-4 w-4 text-s2v-muted" /></template>
        </S2VInput>
        <S2VInput v-model="password" label="密码" type="password">
          <template #prefix><LockKeyhole class="h-4 w-4 text-s2v-muted" /></template>
        </S2VInput>
        <div class="flex items-center justify-between text-xs">
          <label class="flex items-center gap-2 text-s2v-muted"><input type="checkbox" checked class="h-3.5 w-3.5 accent-s2v-primary" />记住我</label>
          <button class="font-semibold text-s2v-primary" type="button" @click="router.push('/auth/verify-email')">忘记密码？</button>
        </div>
        <S2VButton variant="primary" size="lg" :loading="loading" :block="true" @click="login">登录</S2VButton>
      </div>
      <div class="my-4 flex items-center gap-3 text-[11px] text-s2v-muted"><span class="h-px flex-1 bg-s2v-line" />或<span class="h-px flex-1 bg-s2v-line" /></div>
      <S2VButton :block="true" @click="router.push('/auth/register')">创建新账号</S2VButton>
    </S2VCard>
    <div class="flex items-center justify-center gap-1.5 text-[11px] text-s2v-muted"><ShieldCheck class="h-3.5 w-3.5" />邮箱验证后才能开始创作</div>
    <p class="text-center text-[11px] text-s2v-muted">登录即表示同意服务条款和隐私政策</p>
  </div>
</template>
