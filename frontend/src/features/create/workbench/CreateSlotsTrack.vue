<script setup lang="ts">
import { ImagePlus, Plus } from 'lucide-vue-next'
import { computed } from 'vue'
import type { CreateFrameSlot, ImageVariant } from '@/shared/lib/create-draft'
import { MAX_KEYFRAME_EXTRAS, MAX_REF_IMAGES } from '@/shared/lib/create-draft'
import { assetPreviewUrl } from '@/shared/lib/asset-preview'
import { useAppStore } from '@/stores/app'
import { cn } from '@/shared/lib/cn'

export interface SlotDescriptor {
  key: string
  label: string
  required?: boolean
  slot?: CreateFrameSlot
  add?: 'extra' | 'multi'
}

const props = defineProps<{
  variant: ImageVariant
  firstFrame: CreateFrameSlot
  lastFrame: CreateFrameSlot
  extraFrames: CreateFrameSlot[]
  multiFrames: CreateFrameSlot[]
  activeSlotKey: string | null
}>()

const emit = defineEmits<{
  slotClick: [descriptor: SlotDescriptor]
  addExtra: []
  addMulti: []
}>()

const app = useAppStore()

const slotDescriptors = computed<SlotDescriptor[]>(() => {
  const items: SlotDescriptor[] = []
  if (props.variant === 'keyframes') {
    items.push({ key: 'first', label: '首帧', required: true, slot: props.firstFrame })
    items.push({ key: 'last', label: '尾帧', slot: props.lastFrame })
    props.extraFrames.forEach((slot, index) => {
      items.push({ key: `ex-${index}`, label: `参${index + 1}`, slot })
    })
    if (props.extraFrames.length < MAX_KEYFRAME_EXTRAS) items.push({ key: 'add-ex', label: '', add: 'extra' })
  }
  else {
    props.multiFrames.forEach((slot, index) => {
      items.push({ key: `mu-${index}`, label: `参${index + 1}`, required: index === 0, slot })
    })
    if (props.multiFrames.length < MAX_REF_IMAGES) items.push({ key: 'add-mu', label: '', add: 'multi' })
  }
  return items
})

const filledCount = computed(() => {
  if (props.variant === 'keyframes') {
    return [props.firstFrame, props.lastFrame, ...props.extraFrames].filter((slot) => slot.uploaded || slot.assetId).length
  }
  return props.multiFrames.filter((slot) => slot.uploaded || slot.assetId).length
})

const sectionTitle = computed(() => (
  props.variant === 'keyframes' ? '首帧 + 尾帧 + 多图' : '多图参考矩阵'
))

function slotReady(slot?: CreateFrameSlot) {
  return Boolean(slot && (slot.uploaded || slot.assetId))
}

function slotName(slot?: CreateFrameSlot) {
  if (!slot) return ''
  if (slot.fileName) return slot.fileName
  const asset = app.assets.find((item) => item.id === slot.assetId)
  return asset?.name ?? '已选择'
}

function slotImage(slot?: CreateFrameSlot) {
  if (!slot || !slotReady(slot)) return ''
  return assetPreviewUrl(slot.assetId, slot.previewUrl)
}

function onClick(descriptor: SlotDescriptor) {
  if (descriptor.add === 'extra') {
    emit('addExtra')
    return
  }
  if (descriptor.add === 'multi') {
    emit('addMulti')
    return
  }
  emit('slotClick', descriptor)
}
</script>

<template>
  <div class="s2v-wb-panel-section">
    <div class="s2v-wb-section-head">
      <span>{{ sectionTitle }}</span>
      <span class="s2v-wb-section-count s2v-mono">{{ filledCount }} 张</span>
    </div>
    <div class="s2v-wb-slots-track">
      <div v-for="descriptor in slotDescriptors" :key="descriptor.key" class="s2v-wb-slot-cell">
        <span v-if="descriptor.label" class="s2v-wb-slot-label">
          {{ descriptor.label }}<em v-if="descriptor.required">*</em>
        </span>
        <span v-else class="s2v-wb-slot-label">&nbsp;</span>

        <button
          type="button"
          class="s2v-wb-slot-box"
          :class="cn(
            descriptor.slot && slotReady(descriptor.slot) && 's2v-wb-slot-box-filled',
            activeSlotKey === descriptor.key && 's2v-wb-slot-box-active',
          )"
          @click="onClick(descriptor)"
        >
          <template v-if="descriptor.add">
            <Plus class="h-4 w-4 text-s2v-muted" />
            <span class="s2v-wb-slot-hint">添加</span>
          </template>
          <template v-else-if="descriptor.slot && slotReady(descriptor.slot)">
            <img :src="slotImage(descriptor.slot)" :alt="slotName(descriptor.slot)">
            <span class="s2v-wb-slot-badge">{{ slotName(descriptor.slot) }}</span>
          </template>
          <template v-else>
            <ImagePlus class="h-4 w-4 text-s2v-muted" />
            <span class="s2v-wb-slot-hint">上传/选取</span>
          </template>
        </button>
      </div>
    </div>
  </div>
</template>
