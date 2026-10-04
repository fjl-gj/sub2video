<script setup lang="ts">
import { ArrowUpRight, FolderOpen, Trash2 } from 'lucide-vue-next'
import type { WorkProject } from '@/data/projects'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'

defineProps<{
  project: WorkProject
  fileCount: number
  isDefault?: boolean
  isCurrent?: boolean
}>()

defineEmits<{ enter: []; trash: [] }>()
</script>

<template>
  <article class="s2v-project-card">
    <div class="s2v-project-card-top">
      <span class="s2v-project-card-icon"><FolderOpen class="h-6 w-6" /></span>
      <div class="flex flex-wrap justify-end gap-2">
        <S2VBadge v-if="isDefault" tone="primary">默认</S2VBadge>
        <S2VBadge v-if="isCurrent" tone="success">当前作品</S2VBadge>
      </div>
    </div>
    <button class="s2v-project-card-title" type="button" @click="$emit('enter')">{{ project.name }}<ArrowUpRight class="h-4 w-4" /></button>
    <p class="s2v-project-card-description">{{ project.desc || '集中保存这个作品的参考素材与生成成片。' }}</p>
    <div class="s2v-project-card-meta"><span>{{ fileCount }} 个文件</span><span>{{ project.createdAt }}</span></div>
    <footer class="s2v-project-card-actions">
      <S2VButton size="sm" @click="$emit('enter')">进入作品 <ArrowUpRight class="h-3.5 w-3.5" /></S2VButton>
      <S2VButton v-if="!isDefault" size="sm" variant="ghost" @click="$emit('trash')"><Trash2 class="h-3.5 w-3.5" />移入回收站</S2VButton>
    </footer>
  </article>
</template>

<style scoped>
.s2v-project-card { display: flex; flex-direction: column; min-width: 0; padding: 22px; border: 1px solid rgb(var(--s2v-line)); border-radius: var(--s2v-radius); background: rgb(var(--s2v-surface)); transition: border-color .18s ease; }
.s2v-project-card:hover { border-color: rgb(var(--s2v-primary) / .45); }
.s2v-project-card-top { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; margin-bottom: 24px; }
.s2v-project-card-icon { display: grid; place-items: center; flex-shrink: 0; width: 46px; height: 40px; border: 1px solid rgb(var(--s2v-primary) / .2); border-radius: var(--s2v-radius-control); color: rgb(var(--s2v-primary)); background: rgb(var(--s2v-primary) / .07); }
.s2v-project-card-title { display: flex; align-items: center; justify-content: space-between; gap: 12px; min-height: 36px; color: rgb(var(--s2v-ink)); text-align: left; overflow-wrap: anywhere; font-size: 15px; font-weight: 500; }
.s2v-project-card-title > svg { flex-shrink: 0; color: rgb(var(--s2v-muted)); }
.s2v-project-card-title:hover { color: rgb(var(--s2v-primary)); }
.s2v-project-card-description { flex: 1; margin: 6px 0 18px; color: rgb(var(--s2v-muted)); font-size: 12px; line-height: 1.8; }
.s2v-project-card-meta { display: flex; flex-wrap: wrap; justify-content: space-between; gap: 8px; color: rgb(var(--s2v-label)); font-size: 11px; }
.s2v-project-card-actions { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 10px; margin-top: 18px; padding-top: 16px; border-top: 1px solid rgb(var(--s2v-line)); }
@media (max-width: 640px) { .s2v-project-card { padding: 18px; } .s2v-project-card-top { margin-bottom: 16px; } .s2v-project-card-title { min-height: 44px; } }
@media (prefers-reduced-motion: reduce) { .s2v-project-card { transition: none; } }
</style>
