<script setup lang="ts">
import { nextTick, shallowRef, watch } from 'vue'
import { ArrowDown, ArrowUp, Check, Film, Plus } from 'lucide-vue-next'
import type { DirectorShot } from '@/api/director'
import S2VButton from '@/shared/ui/S2VButton.vue'
const props = defineProps<{ shots: DirectorShot[]; activeId: string; disabled: boolean }>()
defineEmits<{ select: [id: string]; move: [id: string, direction: -1 | 1]; add: [] }>()
const list = shallowRef<HTMLOListElement | null>(null)
watch(() => [props.activeId, ...props.shots.map((shot) => shot.id)], async () => {
  await nextTick()
  const container = list.value
  const selected = Array.from(container?.children || []).find((item) => item.getAttribute('data-shot-id') === props.activeId)
  if (!container || !selected || container.scrollHeight <= container.clientHeight) return
  const outer = container.getBoundingClientRect()
  const inner = selected.getBoundingClientRect()
  if (inner.bottom > outer.bottom) container.scrollTop += inner.bottom - outer.bottom
  else if (inner.top < outer.top) container.scrollTop -= outer.top - inner.top
}, { flush: 'post' })
</script>

<template>
  <aside class="director-shot-list" aria-label="镜头列表">
    <div class="director-section-title"><h2>镜头顺序</h2><span>{{ shots.length }} / 100</span></div>
    <ol ref="list" class="director-shots">
      <li v-for="(shot, index) in shots" :key="shot.id" :data-shot-id="shot.id" class="director-shot" :class="{ 'is-active': shot.id === activeId }">
        <button type="button" class="director-shot-select" :aria-current="shot.id === activeId ? 'true' : undefined" :disabled="disabled" @click="$emit('select', shot.id)">
          <span class="director-shot-number">{{ String(index + 1).padStart(2, '0') }}</span>
          <span class="director-shot-copy"><strong>{{ shot.title || '未命名镜头' }}</strong><span>{{ shot.selectedTaskId ? '已选用成片' : shot.candidateTaskIds.length ? `${shot.candidateTaskIds.length} 个候选 · 待选片` : '准备描述与素材' }}</span></span>
          <Check v-if="shot.selectedTaskId" class="h-4 w-4" aria-hidden="true" /><Film v-else class="h-4 w-4" aria-hidden="true" />
        </button>
        <div class="director-shot-order"><S2VButton size="lg" variant="ghost" :aria-label="`上移${shot.title}`" :disabled="disabled || index === 0" @click="$emit('move', shot.id, -1)"><ArrowUp class="h-4 w-4" aria-hidden="true" /></S2VButton><S2VButton size="lg" variant="ghost" :aria-label="`下移${shot.title}`" :disabled="disabled || index === shots.length - 1" @click="$emit('move', shot.id, 1)"><ArrowDown class="h-4 w-4" aria-hidden="true" /></S2VButton></div>
      </li>
    </ol>
    <S2VButton size="lg" block :disabled="disabled || shots.length >= 100" @click="$emit('add')"><Plus class="h-4 w-4" aria-hidden="true" />新增镜头</S2VButton>
    <p class="director-hint">先确定镜头顺序，再逐镜生成和选片。排序不会改变原任务。</p>
  </aside>
</template>

<style scoped>
.director-shots { display: grid; gap: 10px; margin: 18px 0; padding: 0; list-style: none; }
.director-shot { border: 1px solid rgb(var(--s2v-line)); border-radius: var(--s2v-radius-control); background: rgb(var(--s2v-surface)); }
.director-shot.is-active { border-color: rgb(var(--s2v-primary)); }
.director-shot-select { display: flex; gap: 12px; align-items: center; width: 100%; padding: 14px; min-height: 72px; text-align: left; }
.director-shot-select:focus-visible { outline: 2px solid rgb(var(--s2v-primary)); outline-offset: 2px; }
.director-shot-number { color: rgb(var(--s2v-muted)); font-size: 12px; font-variant-numeric: tabular-nums; }
.director-shot-copy { display: grid; gap: 7px; flex: 1; min-width: 0; }
.director-shot-copy strong { font-size: 13px; font-weight: 500; overflow-wrap: anywhere; color: rgb(var(--s2v-ink)); }
.director-shot-copy span { color: rgb(var(--s2v-muted)); font-size: 12px; line-height: 1.5; }
.director-shot-order { display: flex; justify-content: flex-end; padding: 0 8px 6px; }
@container director (max-width: 720px) {
  .director-shots { max-height: 224px; overflow-y: auto; overscroll-behavior: contain; gap: 8px; margin: 12px 0; }
  .director-shot { display: grid; grid-template-columns: minmax(0, 1fr) auto; align-items: center; }
  .director-shot-select { padding: 12px 10px; gap: 8px; min-height: 66px; }
  .director-shot-select > svg { display: none; }
  .director-shot-order { padding: 0 4px 0 0; }
  .director-shot-list > .director-hint { margin-top: 10px; }
}
</style>
