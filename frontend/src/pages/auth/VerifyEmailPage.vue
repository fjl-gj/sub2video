<script setup lang="ts">
import { ArrowLeft, CheckCircle2, Film, Mail, RefreshCw } from 'lucide-vue-next'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
const router = useRouter(); const sent = ref(false); const email = ref('linmo@example.com')
function resend() { sent.value = true; window.setTimeout(() => { sent.value = false }, 2800) }
</script>
<template><div class="grid gap-5 text-center"><span class="mx-auto grid h-11 w-11 place-items-center rounded-[10px] bg-indigo-600 text-white shadow-subtle"><Film class="h-5 w-5" /></span><div><h1 class="text-xl font-bold">验证你的邮箱</h1><p class="mt-1.5 text-xs leading-5 text-s2v-muted">我们已将验证链接发送到<br /><strong class="text-s2v-ink">{{ email }}</strong></p></div><div class="rounded-panel border border-s2v-line bg-s2v-surface p-6 shadow-subtle"><Mail class="mx-auto h-9 w-9 text-indigo-600" /><S2VBadge tone="primary" class="mt-4">等待邮箱验证</S2VBadge><p class="mt-3 text-xs leading-5 text-s2v-muted">点击邮件中的链接完成验证。没有收到邮件时，请检查垃圾邮件或重新发送。</p><S2VButton variant="primary" :loading="sent" :block="true" class="mt-5" @click="resend"><RefreshCw class="h-3.5 w-3.5" />{{ sent ? '已重新发送' : '重新发送验证邮件' }}</S2VButton></div><div class="flex items-center justify-center gap-4 text-xs"><button class="flex items-center gap-1 font-semibold text-s2v-muted" @click="router.push('/auth/login')"><ArrowLeft class="h-3.5 w-3.5" />返回登录</button><button class="flex items-center gap-1 font-semibold text-indigo-600" @click="router.push('/app/overview')"><CheckCircle2 class="h-3.5 w-3.5" />原型直接进入</button></div></div></template>
