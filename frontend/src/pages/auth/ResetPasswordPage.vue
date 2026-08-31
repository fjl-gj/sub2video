<script setup lang="ts">
import { ArrowLeft, Check, LockKeyhole } from 'lucide-vue-next'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import NoticeScreen from '@/features/system/NoticeScreen.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'

const router = useRouter()
const password = ref('')
const confirm = ref('')
const loading = ref(false)

function submit() {
  if (!password.value || password.value !== confirm.value) return
  loading.value = true
  window.setTimeout(() => router.push('/auth/login'), 400)
}
</script>

<template>
  <NoticeScreen title="设置新密码">
    <template #description>重置链接有效。请设置至少 8 位的新密码。</template>
    <div class="grid gap-4 text-left">
      <S2VInput v-model="password" label="新密码" type="password" placeholder="至少 8 位">
        <template #prefix><LockKeyhole class="h-4 w-4 text-s2v-muted" /></template>
      </S2VInput>
      <S2VInput v-model="confirm" label="确认密码" type="password">
        <template #prefix><LockKeyhole class="h-4 w-4 text-s2v-muted" /></template>
      </S2VInput>
      <S2VButton variant="primary" size="lg" :block="true" :loading="loading" :disabled="!password || password !== confirm" @click="submit"><Check class="h-4 w-4" />保存并返回登录</S2VButton>
    </div>
    <template #footer>
      <button class="flex items-center gap-1 font-semibold text-s2v-muted" type="button" @click="router.push('/auth/login')"><ArrowLeft class="h-3.5 w-3.5" />返回登录</button>
    </template>
  </NoticeScreen>
</template>
