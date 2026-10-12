<script setup lang="ts">
import { ArrowUpRight, Trash2 } from 'lucide-vue-next'
import type { DirectorShot } from '@/api/director'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VTextarea from '@/shared/ui/S2VTextarea.vue'
const props = defineProps<{ shot: DirectorShot; disabled: boolean; dirty: boolean }>()
const emit = defineEmits<{ update: [value: Pick<DirectorShot, 'title' | 'prompt'>]; create: []; remove: [] }>()
function update(field: 'title' | 'prompt', value: string) { emit('update', { title: props.shot.title, prompt: props.shot.prompt, [field]: value }) }
</script>

<template>
  <section class="director-editor" aria-label="镜头编辑">
    <div class="director-section-title"><h2>镜头意图</h2><S2VButton size="lg" variant="ghost" :disabled="disabled" @click="$emit('remove')"><Trash2 class="h-4 w-4" aria-hidden="true" />移除镜头</S2VButton></div>
    <fieldset :disabled="disabled" class="director-editor-fields">
      <S2VInput :model-value="shot.title" label="镜头名称" maxlength="120" placeholder="例如：海岸的清晨" @update:model-value="update('title', $event)" />
      <S2VTextarea :model-value="shot.prompt" label="画面描述" :rows="5" placeholder="描述人物、场景、动作与镜头运动。参考素材和生成参数在创作页准备。" @update:model-value="update('prompt', $event)" />
    </fieldset>
    <div class="director-editor-settings"><span>最近使用的参数</span><p v-if="shot.modeId">{{ shot.durationSec }} 秒 · {{ shot.resolution }} · {{ shot.aspectRatio }} · {{ shot.generateAudio ? '生成音频' : '无自动音频' }}</p><p v-else>尚未生成。进入创作页后选择模式、素材与参数。</p></div>
    <S2VButton variant="primary" size="lg" :disabled="disabled || !shot.title.trim()" @click="$emit('create')">{{ dirty ? '保存并前往创作' : '前往创作' }}<ArrowUpRight class="h-4 w-4" aria-hidden="true" /></S2VButton>
    <p class="director-hint">仅进入创作页，不自动提交或扣费。确认素材、参数与估价后再生成。</p>
  </section>
</template>

<style scoped>
.director-editor { min-width: 0; }
.director-editor-fields { display: grid; gap: 20px; margin: 14px 0 20px; min-width: 0; }
.director-editor-settings { padding: 18px 0; margin-bottom: 10px; border-top: 1px solid rgb(var(--s2v-line)); }
.director-editor-settings span { font-size: 12px; color: rgb(var(--s2v-muted)); }
.director-editor-settings p { margin-top: 8px; font-size: 13px; line-height: 1.7; color: rgb(var(--s2v-body)); }
</style>
