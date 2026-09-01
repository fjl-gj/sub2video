<script setup lang="ts">
import { FolderOpen, Plus, RefreshCw, Upload, X } from 'lucide-vue-next'
import { computed } from 'vue'
import type { CreateFrameSlot } from '@/shared/lib/create-draft'
import { useAppStore } from '@/stores/app'
import { cn } from '@/shared/lib/cn'

const props = withDefaults(defineProps<{
  label: string
  slotData: CreateFrameSlot
  required?: boolean
  removable?: boolean
  split?: boolean
  placeholder?: string
  compact?: boolean
}>(), {
  split: false,
  compact: false,
})

const emit = defineEmits<{
  upload: []
  pick: []
  clear: []
  remove: []
}>()

const app = useAppStore()
const ready = computed(() => props.slotData.uploaded || Boolean(props.slotData.assetId))
const previewLabel = computed(() => {
  if (!ready.value) return ''
  const asset = app.assets.find((item) => item.id === props.slotData.assetId)
  return asset?.name || '已选'
})
const emptyHint = computed(() => props.placeholder || props.label)
const useSplit = computed(() => props.split || props.compact)
</script>

<template>
  <div :class="cn('s2v-ref-card', compact && 's2v-ref-card-compact')">
    <div class="flex min-h-3.5 items-center justify-between gap-1">
      <span class="truncate text-[10px] font-semibold text-s2v-ink/80">{{ label }}</span>
      <button
        v-if="removable"
        aria-label="移除"
        class="rounded-nav p-0.5 text-s2v-muted hover:bg-s2v-canvas hover:text-s2v-ink"
        type="button"
        @click="$emit('remove')"
      >
        <X class="h-3 w-3" />
      </button>
    </div>

    <div v-if="ready" class="s2v-ref-slot s2v-ref-slot-filled">
      <button
        aria-label="清除"
        class="absolute right-0.5 top-0.5 z-10 rounded-nav bg-s2v-surface/90 p-0.5 text-s2v-muted hover:text-s2v-ink"
        type="button"
        @click.stop="$emit('clear')"
      >
        <X class="h-3 w-3" />
      </button>
      <span class="line-clamp-2 px-0.5 text-[9px] leading-3">{{ previewLabel }}</span>
      <div class="s2v-ref-slot-overlay">
        <RefreshCw class="h-3 w-3" />
        <div class="flex gap-0.5">
          <button class="rounded-nav bg-white/15 px-1 py-0.5 text-[9px]" type="button" @click.stop="$emit('upload')">传</button>
          <button class="rounded-nav bg-white/15 px-1 py-0.5 text-[9px]" type="button" @click.stop="$emit('pick')">库</button>
        </div>
      </div>
    </div>

    <div v-else-if="useSplit" class="s2v-ref-slot s2v-ref-slot-split">
      <button class="s2v-ref-slot-action" type="button" @click="$emit('upload')">
        <Upload class="h-3 w-3" />
        <span>上传</span>
      </button>
      <button class="s2v-ref-slot-action" type="button" @click="$emit('pick')">
        <FolderOpen class="h-3 w-3" />
        <span>资产</span>
      </button>
    </div>

    <button v-else class="s2v-ref-slot" type="button" @click="$emit('upload')">
      <Plus class="h-3.5 w-3.5 text-s2v-primary" />
      <span class="text-[9px] leading-3">{{ required ? '必填' : emptyHint }}</span>
    </button>

    <button
      v-if="!ready && !useSplit && !compact"
      class="text-center text-[10px] text-s2v-primary hover:underline"
      type="button"
      @click="$emit('pick')"
    >
      从资产库选
    </button>
  </div>
</template>
