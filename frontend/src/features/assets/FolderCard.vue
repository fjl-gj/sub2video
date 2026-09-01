<script setup lang="ts">
import { Folder, FolderLock } from 'lucide-vue-next'
import type { FsFolder } from '@/data/projects'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'

defineProps<{
  name: string
  folder: FsFolder
  count: number
}>()

defineEmits<{ open: [] }>()
</script>

<template>
  <S2VCard variant="compact">
    <button class="flex w-full items-start gap-3 text-left" type="button" @click="$emit('open')">
      <span class="s2v-icon-well s2v-icon-well-nav">
        <FolderLock v-if="folder.isSystem" class="h-4 w-4" />
        <Folder v-else class="h-4 w-4" />
      </span>
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <strong class="truncate text-xs text-s2v-ink">{{ name }}</strong>
          <S2VBadge v-if="folder.isSystem">系统</S2VBadge>
        </div>
        <p class="mt-1 text-[11px] text-s2v-muted">{{ folder.desc }}</p>
        <p class="mt-1.5 text-[11px] text-s2v-label">{{ count }} 项</p>
      </div>
    </button>
  </S2VCard>
</template>
