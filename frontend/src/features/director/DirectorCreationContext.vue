<script setup lang="ts">
import S2VButton from '@/shared/ui/S2VButton.vue'
defineProps<{ title?: string; applied: boolean; loading: boolean; blocked: boolean; saving: boolean; canSave: boolean; needsReapply: boolean; error: string; notice?: string; returnRoute: { path: string; query?: { shot: string } } }>()
defineEmits<{ apply: []; reload: []; save: [] }>()
</script>

<template>
  <section class="director-creation-context" aria-label="镜头创作上下文">
    <div><h2>{{ applied ? '正在为镜头创作' : '从导演台进入创作' }}{{ title ? `：${title}` : '' }}</h2><p>{{ applied ? '可先把描述、素材和参数保存到镜头，之后再确认估价并提交生成。普通创作草稿单独保留。' : '应用后填写素材与参数，生成结果会加入镜头候选。不会覆盖你的普通创作草稿。' }}</p><p>保存镜头会上传本地图片，不创建生成任务、不扣费。导演台暂不支持参考音频。</p><p v-if="notice" role="status">{{ notice }}</p><p v-if="error" class="director-context-error" role="alert">{{ error }}</p><p v-if="blocked">请先核对原提交结果，暂时不能更换镜头上下文。</p></div>
    <div class="director-context-actions"><RouterLink :to="returnRoute">返回导演台</RouterLink><S2VButton v-if="error" size="lg" :loading="loading" :disabled="saving || blocked" @click="$emit('reload')">重新读取镜头</S2VButton><S2VButton v-if="!applied || needsReapply" variant="primary" size="lg" :loading="loading" :disabled="saving || blocked || !!error || !title" @click="$emit('apply')">{{ applied ? '重新应用镜头' : '应用镜头到创作' }}</S2VButton><span v-else>已应用镜头</span><S2VButton size="lg" :disabled="!canSave" :loading="saving" @click="$emit('save')">保存到镜头</S2VButton></div>
  </section>
</template>

<style scoped>
.director-creation-context { display: flex; flex-wrap: wrap; justify-content: space-between; align-items: center; gap: 18px; padding: 20px; margin-bottom: 24px; border: 1px solid rgb(var(--s2v-line)); border-radius: var(--s2v-radius); background: rgb(var(--s2v-surface)); }
.director-creation-context > div:first-child { flex: 1; min-width: min(100%, 220px); }
.director-creation-context h2 { font-size: 15px; font-weight: 500; color: rgb(var(--s2v-ink)); }
.director-creation-context p { margin-top: 8px; font-size: 12px; line-height: 1.8; color: rgb(var(--s2v-muted)); }
.director-creation-context .director-context-error { color: rgb(var(--s2v-danger)); }
.director-context-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 16px; font-size: 13px; }
.director-context-actions a { display: inline-flex; align-items: center; min-height: 44px; color: rgb(var(--s2v-primary)); }
.director-context-actions a:focus-visible { outline: 2px solid rgb(var(--s2v-primary)); outline-offset: 3px; }
.director-context-actions span { color: rgb(var(--s2v-muted)); }
</style>
