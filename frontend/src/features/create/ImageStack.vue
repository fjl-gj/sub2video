<script setup lang="ts">
import { ArrowRight, Plus } from 'lucide-vue-next'
import { computed } from 'vue'
import CreateFrameTile from '@/features/create/CreateFrameTile.vue'
import ImageSlotCard from '@/features/create/ImageSlotCard.vue'
import type { CreateFrameSlot } from '@/shared/lib/create-draft'
import { MAX_REF_IMAGES } from '@/shared/lib/create-draft'

const props = defineProps<{
  variant: 'keyframes' | 'multi'
  firstFrame: CreateFrameSlot
  lastFrame: CreateFrameSlot
  extraFrames: CreateFrameSlot[]
  multiFrames: CreateFrameSlot[]
}>()

const emit = defineEmits<{
  upload: [slot: CreateFrameSlot]
  pick: [slot: CreateFrameSlot]
  clear: [slot: CreateFrameSlot]
  addExtra: []
  removeExtra: [index: number]
  addMulti: []
  removeMulti: [index: number]
}>()

const readyExtra = computed(() => props.extraFrames.filter((slot) => slot.uploaded || slot.assetId).length)
const readyMulti = computed(() => props.multiFrames.filter((slot) => slot.uploaded || slot.assetId).length)
</script>

<template>
  <div class="grid gap-4">
    <template v-if="variant === 'keyframes'">
      <div class="s2v-create-keyframes">
        <CreateFrameTile
          label="首帧"
          :slot-data="firstFrame"
          required
          emphasis
          @upload="emit('upload', firstFrame)"
          @pick="emit('pick', firstFrame)"
          @clear="emit('clear', firstFrame)"
        />
        <div aria-hidden="true" class="s2v-create-keyframes-arrow">
          <ArrowRight class="h-4 w-4" />
        </div>
        <CreateFrameTile
          label="尾帧"
          :slot-data="lastFrame"
          @upload="emit('upload', lastFrame)"
          @pick="emit('pick', lastFrame)"
          @clear="emit('clear', lastFrame)"
        />
      </div>

      <div class="grid gap-2 border-t border-s2v-line/60 pt-4">
        <div class="flex items-center justify-between gap-2">
          <p class="text-xs font-semibold text-s2v-ink">其他参考图 <span class="font-normal text-s2v-muted">（0~8）</span></p>
          <span class="s2v-mono text-[11px] text-s2v-muted">{{ readyExtra }} / {{ MAX_REF_IMAGES }}</span>
        </div>
        <div class="s2v-ref-rail">
          <ImageSlotCard
            v-for="(slot, index) in extraFrames"
            :key="`extra-${index}`"
            :label="`参考 ${index + 1}`"
            :slot-data="slot"
            split
            removable
            @upload="emit('upload', slot)"
            @pick="emit('pick', slot)"
            @clear="emit('clear', slot)"
            @remove="emit('removeExtra', index)"
          />
          <button
            v-if="extraFrames.length < MAX_REF_IMAGES"
            class="s2v-ref-card s2v-ref-card-compact grid gap-1"
            type="button"
            @click="emit('addExtra')"
          >
            <span class="min-h-3.5" />
            <span class="s2v-ref-slot s2v-ref-slot-add">
              <Plus class="h-3.5 w-3.5 text-s2v-primary" />
              <span class="text-[9px]">添加</span>
            </span>
          </button>
        </div>
      </div>
    </template>

    <template v-else>
      <div class="flex items-center justify-between gap-2">
        <p class="text-xs font-semibold text-s2v-ink">参考图 <span class="font-normal text-s2v-muted">（1~8，至少 1 张）</span></p>
        <span class="s2v-mono text-[11px] text-s2v-muted">{{ readyMulti }} / {{ multiFrames.length }}</span>
      </div>
      <div class="s2v-grid-2 gap-3 sm:grid-cols-3">
        <CreateFrameTile
          v-for="(slot, index) in multiFrames"
          :key="`multi-${index}`"
          :label="`参考 ${index + 1}`"
          :slot-data="slot"
          :required="index === 0"
          :emphasis="index === 0 && readyMulti === 0"
          @upload="emit('upload', slot)"
          @pick="emit('pick', slot)"
          @clear="emit('clear', slot)"
        />
        <button
          v-if="multiFrames.length < MAX_REF_IMAGES"
          class="s2v-create-slot grid place-content-center"
          type="button"
          @click="emit('addMulti')"
        >
          <Plus class="mx-auto h-5 w-5 text-s2v-primary" />
          <span class="mt-1 text-[11px] font-semibold text-s2v-ink">添加参考图</span>
        </button>
      </div>
    </template>
  </div>
</template>
