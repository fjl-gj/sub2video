<script setup lang="ts">
import { ImagePlus, Plus, X } from 'lucide-vue-next'
import { computed } from 'vue'
import type { CreateFrameSlot, ImageVariant } from '@/shared/lib/create-draft'
import { MAX_REF_IMAGES } from '@/shared/lib/create-draft'
import { assetPreviewUrl } from '@/shared/lib/asset-preview'
import { useAppStore } from '@/stores/app'
import { cn } from '@/shared/lib/cn'

export interface SlotDescriptor {
  key: string
  label: string
  required?: boolean
  slot?: CreateFrameSlot
  add?: 'multi'
}

const props = defineProps<{
  variant: ImageVariant
  firstFrame: CreateFrameSlot
  lastFrame: CreateFrameSlot
  extraFrames: CreateFrameSlot[]
  multiFrames: CreateFrameSlot[]
  maxImages?: number
  activeSlotKey: string | null
}>()

const emit = defineEmits<{
  slotClick: [descriptor: SlotDescriptor]
  clearSlot: [descriptor: SlotDescriptor]
  addMulti: []
}>()

const app = useAppStore()

const slotDescriptors = computed<SlotDescriptor[]>(() => {
  const items: SlotDescriptor[] = []
  if (props.variant === 'keyframes') {
    items.push({ key: 'first', label: '首帧', required: true, slot: props.firstFrame })
    items.push({ key: 'last', label: '尾帧', required: true, slot: props.lastFrame })
  }
  else {
    props.multiFrames.forEach((slot, index) => {
      items.push({ key: `mu-${index}`, label: `参考图 ${index + 1}`, required: index === 0, slot })
    })
    if (props.multiFrames.length < (props.maxImages || MAX_REF_IMAGES)) items.push({ key: 'add-mu', label: '', add: 'multi' })
  }
  return items
})

const filledCount = computed(() => {
  if (props.variant === 'keyframes') {
    return [props.firstFrame, props.lastFrame].filter((slot) => slot.uploaded || slot.assetId).length
  }
  return props.multiFrames.filter((slot) => slot.uploaded || slot.assetId).length
})

const sectionTitle = computed(() => (
  props.variant === 'keyframes' ? '首帧与尾帧' : '参考素材'
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
  const asset = app.assets.find((item) => item.id === slot.assetId)
  return assetPreviewUrl(slot.assetId, slot.previewUrl || asset?.url)
}

function onClick(descriptor: SlotDescriptor) {
  if (descriptor.add === 'multi') {
    emit('addMulti')
    return
  }
  emit('slotClick', descriptor)
}

function onClear(event: MouseEvent, descriptor: SlotDescriptor) {
  event.stopPropagation()
  emit('clearSlot', descriptor)
}
</script>

<template>
  <div class="s2v-studio-section">
    <div class="s2v-studio-field-heading">
      <span>{{ sectionTitle }}</span>
      <span class="s2v-mono">{{ filledCount }} / {{ variant === 'keyframes' ? 2 : maxImages || MAX_REF_IMAGES }} 张</span>
    </div>
    <div class="s2v-studio-slots">
      <div v-for="descriptor in slotDescriptors" :key="descriptor.key" class="s2v-studio-slot-cell">
        <span v-if="descriptor.label" class="s2v-studio-slot-label">
          {{ descriptor.label }}<em v-if="descriptor.required">*</em>
        </span>
        <span v-else class="s2v-studio-slot-label">&nbsp;</span>

        <div
          class="s2v-studio-slot"
          :class="cn(
            descriptor.slot && slotReady(descriptor.slot) && 's2v-studio-slot-filled',
            activeSlotKey === descriptor.key && 's2v-studio-slot-active',
          )"
          :aria-label="descriptor.add ? '添加参考图' : `选择${descriptor.label}`"
          role="button"
          tabindex="0"
          @click="onClick(descriptor)"
          @keydown.enter.self.prevent="onClick(descriptor)"
          @keydown.space.self.prevent="onClick(descriptor)"
        >
          <template v-if="descriptor.add">
            <Plus class="h-4 w-4 text-s2v-muted" />
            <span class="s2v-studio-caption">添加</span>
          </template>
          <template v-else-if="descriptor.slot && slotReady(descriptor.slot)">
            <img :src="slotImage(descriptor.slot)" :alt="slotName(descriptor.slot)">
            <button
              type="button"
              class="s2v-studio-slot-clear"
              aria-label="清除图片"
              @click="onClear($event, descriptor)"
            >
              <X class="h-3 w-3" />
            </button>
          </template>
          <template v-else>
            <ImagePlus class="h-4 w-4 text-s2v-muted" />
            <span class="s2v-studio-caption">上传 / 选取</span>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>
