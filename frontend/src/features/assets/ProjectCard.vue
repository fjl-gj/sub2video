<script setup lang="ts">
import { FolderKanban } from 'lucide-vue-next'
import type { WorkProject } from '@/data/projects'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'

defineProps<{
  project: WorkProject
  fileCount: number
  isDefault?: boolean
  isCurrent?: boolean
}>()

defineEmits<{
  enter: []
  trash: []
}>()
</script>

<template>
  <S2VCard variant="padded">
    <div class="flex items-start gap-3">
      <span class="s2v-icon-well s2v-icon-well-nav"><FolderKanban class="h-4 w-4" /></span>
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-2">
          <strong class="truncate text-sm text-s2v-ink">{{ project.name }}</strong>
          <S2VBadge v-if="isDefault" tone="primary">默认</S2VBadge>
          <S2VBadge v-if="isCurrent" tone="success">当前作品</S2VBadge>
        </div>
        <p class="mt-1 text-xs text-s2v-muted">{{ project.desc }}</p>
        <p class="mt-2 text-[11px] text-s2v-label">{{ project.createdAt }} · {{ fileCount }} 个文件</p>
      </div>
    </div>
    <div class="mt-4 flex flex-wrap gap-2">
      <S2VButton variant="primary" size="sm" @click="$emit('enter')">进入作品</S2VButton>
      <S2VButton v-if="!isDefault" size="sm" @click="$emit('trash')">移入回收站</S2VButton>
    </div>
  </S2VCard>
</template>
