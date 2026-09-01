<script setup lang="ts">
import { Check, FolderOpen, Plus, Upload } from 'lucide-vue-next'
import { computed } from 'vue'
import type { CreateFrameSlot } from '@/shared/lib/create-draft'
import { useAppStore } from '@/stores/app'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VChoice from '@/shared/ui/S2VChoice.vue'

const props = defineProps<{
  variant: 'keyframes' | 'multi'
  firstFrame: CreateFrameSlot
  lastFrame: CreateFrameSlot
  multiFrames: CreateFrameSlot[]
}>()

const emit = defineEmits<{
  upload: [slot: CreateFrameSlot]
  pick: [slot: CreateFrameSlot, assetId: string]
  togglePick: [slot: CreateFrameSlot]
}>()

const app = useAppStore()
const imageAssets = computed(() => app.assets.filter((item) => item.type === '图片'))

const keyframeLayers = computed(() => [
  { slot: props.lastFrame, label: '尾帧', optional: true, offset: 'left-3 top-5 -rotate-6 scale-95' },
  { slot: props.firstFrame, label: '首帧', optional: false, offset: 'left-14 top-0 rotate-3' },
])

const readyMulti = computed(() => props.multiFrames.filter((slot) => frameReady(slot)))

function frameReady(slot: CreateFrameSlot) {
  return slot.uploaded || Boolean(slot.assetId)
}

function frameLabel(slot: CreateFrameSlot, fallback: string) {
  if (!frameReady(slot)) return fallback
  const asset = app.assets.find((item) => item.id === slot.assetId)
  return asset?.name || '图片已选择'
}

function slotKey(slot: CreateFrameSlot) {
  return `${slot.assetId}:${slot.uploaded}`
}
</script>

<template>
  <div class="grid gap-4">
    <template v-if="variant === 'keyframes'">
      <div class="relative mx-auto h-52 w-full max-w-md">
        <button
          v-for="layer in keyframeLayers"
          :key="layer.label"
          type="button"
          class="absolute w-40 overflow-hidden rounded-panel border border-s2v-line bg-s2v-surface shadow-subtle transition hover:border-s2v-primary"
          :class="layer.offset"
          @click="emit('upload', layer.slot)"
        >
          <div class="grid aspect-[3/4] place-items-center bg-s2v-canvas p-3 text-center">
            <template v-if="!frameReady(layer.slot)">
              <Upload class="h-5 w-5 text-s2v-primary" />
              <strong class="mt-2 block text-xs text-s2v-ink">{{ layer.label }}</strong>
              <span class="mt-1 block text-[10px] text-s2v-muted">{{ layer.optional ? '可选' : '必填' }}</span>
            </template>
            <template v-else>
              <Check class="h-6 w-6 text-s2v-success" />
              <strong class="mt-2 block truncate px-1 text-xs text-s2v-ink">{{ frameLabel(layer.slot, layer.label) }}</strong>
            </template>
          </div>
        </button>
      </div>
      <div class="flex flex-wrap justify-center gap-2">
        <S2VButton size="sm" variant="ghost" @click="emit('togglePick', firstFrame)">
          <template #icon><FolderOpen class="h-3.5 w-3.5" /></template>
          首帧从资产选
        </S2VButton>
        <S2VButton size="sm" variant="ghost" @click="emit('togglePick', lastFrame)">
          <template #icon><FolderOpen class="h-3.5 w-3.5" /></template>
          尾帧从资产选
        </S2VButton>
      </div>
      <div v-if="firstFrame.picking || lastFrame.picking" class="grid gap-2">
        <S2VChoice
          v-for="item in imageAssets"
          :key="item.id"
          :selected="(firstFrame.picking && firstFrame.assetId === item.id) || (lastFrame.picking && lastFrame.assetId === item.id)"
          @click="emit('pick', firstFrame.picking ? firstFrame : lastFrame, item.id)"
        >
          {{ item.name }} · {{ item.folder }}
        </S2VChoice>
        <p v-if="!imageAssets.length" class="text-center text-[11px] text-s2v-muted">当前作品里还没有图片</p>
      </div>
    </template>

    <template v-else>
      <div class="flex min-h-48 items-center justify-center">
        <div class="relative h-40 w-full max-w-lg">
          <button
            v-for="(slot, index) in multiFrames"
            :key="slotKey(slot)"
            type="button"
            class="absolute top-2 w-28 overflow-hidden rounded-panel border border-s2v-line bg-s2v-surface shadow-subtle transition hover:border-s2v-primary"
            :class="index === 0 ? 'left-1/2 z-30 -translate-x-1/2 rotate-0' : index === 1 ? 'left-[18%] z-20 -rotate-6' : 'right-[18%] z-10 rotate-6'"
            @click="emit('upload', slot)"
          >
            <div class="grid aspect-square place-items-center bg-s2v-canvas p-2 text-center">
              <template v-if="!frameReady(slot)">
                <Plus class="h-4 w-4 text-s2v-primary" />
                <span class="mt-1 block text-[10px] text-s2v-muted">参考 {{ index + 1 }}</span>
              </template>
              <template v-else>
                <Check class="h-5 w-5 text-s2v-success" />
                <span class="mt-1 block truncate px-1 text-[10px] text-s2v-ink">{{ frameLabel(slot, `参考 ${index + 1}`) }}</span>
              </template>
            </div>
          </button>
        </div>
      </div>
      <p class="text-center text-[11px] text-s2v-muted">已选 {{ readyMulti.length }} / {{ multiFrames.length }} 张 · 至少 1 张</p>
      <div class="flex flex-wrap justify-center gap-2">
        <S2VButton
          v-for="(slot, index) in multiFrames"
          :key="`pick-${index}`"
          size="sm"
          variant="ghost"
          @click="emit('togglePick', slot)"
        >
          <template #icon><FolderOpen class="h-3.5 w-3.5" /></template>
          参考 {{ index + 1 }}
        </S2VButton>
      </div>
      <div v-if="multiFrames.some((slot) => slot.picking)" class="grid gap-2">
        <template v-for="(slot, index) in multiFrames" :key="`asset-${index}`">
          <div v-if="slot.picking" class="grid gap-2">
            <p class="text-xs font-semibold text-s2v-ink/70">为参考图 {{ index + 1 }} 选择资产</p>
            <S2VChoice
              v-for="item in imageAssets"
              :key="item.id"
              :selected="slot.assetId === item.id"
              @click="emit('pick', slot, item.id)"
            >
              {{ item.name }}
            </S2VChoice>
          </div>
        </template>
      </div>
    </template>
  </div>
</template>
