<script setup lang="ts">
import { ArrowLeft, Check, LockKeyhole, Mail, UserRound } from 'lucide-vue-next'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import AuthBrand from '@/widgets/AuthBrand.vue'

const router = useRouter()
const auth = useAuthStore()
const app = useAppStore()
const name = ref('')
const email = ref('')
const password = ref('')
const agree = ref(true)
const loading = ref(false)
const error = ref('')

async function register() {
  if (!name.value || !email.value || !password.value || !agree.value) return
  error.value = ''
  loading.value = true
  try {
    const user = await auth.register(email.value.trim(), password.value, name.value.trim())
    app.balance = user.balance
    await router.push('/app/overview')
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message
    error.value = msg || '注册失败，请稍后重试'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="grid gap-5">
    <div class="flex items-center justify-between">
      <button class="flex items-center gap-1 text-xs font-semibold text-s2v-muted" type="button" @click="router.push('/auth/login')"><ArrowLeft class="h-3.5 w-3.5" />返回登录</button>
      <AuthBrand size="sm" />
    </div>
    <div>
      <h1 class="text-xl font-bold">创建账号</h1>
      <p class="mt-1.5 text-xs text-s2v-muted">注册后即可开始视频创作（UAT 暂跳过邮箱验证）。</p>
    </div>
    <S2VCard variant="padded">
      <div class="grid gap-4">
        <S2VInput v-model="name" label="昵称" placeholder="你的创作者昵称">
          <template #prefix><UserRound class="h-4 w-4 text-s2v-muted" /></template>
        </S2VInput>
        <S2VInput v-model="email" label="邮箱" type="email" placeholder="name@example.com">
          <template #prefix><Mail class="h-4 w-4 text-s2v-muted" /></template>
        </S2VInput>
        <S2VInput v-model="password" label="密码" type="password" placeholder="至少 8 位">
          <template #prefix><LockKeyhole class="h-4 w-4 text-s2v-muted" /></template>
        </S2VInput>
        <p v-if="error" class="text-xs text-s2v-danger">{{ error }}</p>
        <label class="flex items-start gap-2 text-[11px] leading-5 text-s2v-muted">
          <input v-model="agree" type="checkbox" class="mt-1 h-3.5 w-3.5 accent-s2v-primary" />
          <span>我已阅读并同意 <button class="font-semibold text-s2v-primary" type="button" @click="router.push('/auth/terms')">服务条款</button> 与 <button class="font-semibold text-s2v-primary" type="button" @click="router.push('/auth/privacy')">隐私政策</button></span>
        </label>
        <S2VButton variant="primary" size="lg" :loading="loading" :block="true" @click="register"><Check class="h-4 w-4" />注册并进入</S2VButton>
      </div>
    </S2VCard>
  </div>
</template>
