<script setup lang="ts">
import { Check, FolderOpen, Image as ImageIcon, Upload, X } from 'lucide-vue-next'
import { computed } from 'vue'
import type { CreateFrameSlot } from '@/shared/lib/create-draft'
import { useAppStore } from '@/stores/app'
import { cn } from '@/shared/lib/cn'

const props = withDefaults(defineProps<{
  label: string
  slotData: CreateFrameSlot
  required?: boolean
  emphasis?: boolean
}>(), {
  required: false,
  emphasis: false,
})

const emit = defineEmits<{
  upload: []
  pick: []
  clear: []
}>()

const app = useAppStore()
const ready = computed(() => props.slotData.uploaded || Boolean(props.slotData.assetId))
const asset = computed(() => app.assets.find((item) => item.id === props.slotData.assetId))
const previewName = computed(() => asset.value?.name || '已选择')
</script>

<template>
  <article
    class="s2v-create-slot"
    :class="cn(
      ready && 's2v-create-slot-ready',
      !ready && emphasis && 's2v-create-slot-focus',
    )"
  >
    <div class="flex items-start justify-between gap-2">
      <div>
        <p class="text-xs font-semibold text-s2v-ink">{{ label }}</p>
        <p class="mt-0.5 text-[11px] text-s2v-muted">{{ required ? '必填' : '选填' }}</p>
      </div>
      <button
        v-if="ready"
        aria-label="清除"
        class="rounded-nav p-0.5 text-s2v-muted hover:text-s2v-ink"
        type="button"
        @click="$emit('clear')"
      >
        <X class="h-3.5 w-3.5" />
      </button>
    </div>

    <button class="s2v-create-slot-preview" type="button" @click="$emit('upload')">
      <template v-if="!ready">
        <ImageIcon class="h-5 w-5 text-s2v-primary" />
        <span class="text-[11px] font-semibold text-s2v-ink">{{ emphasis ? '点击上传首帧' : '点击上传' }}</span>
      </template>
      <template v-else>
        <Check class="h-5 w-5 text-s2v-success" />
        <span class="line-clamp-2 text-[11px] font-semibold leading-4 text-s2v-ink">{{ previewName }}</span>
      </template>
    </button>

    <div class="grid grid-cols-2 gap-1.5">
      <button class="s2v-create-slot-action" type="button" @click="$emit('upload')">
        <Upload class="h-3 w-3" />
        上传
      </button>
      <button class="s2v-create-slot-action" type="button" @click="$emit('pick')">
        <FolderOpen class="h-3 w-3" />
        资产
      </button>
    </div>
  </article>
</template>
