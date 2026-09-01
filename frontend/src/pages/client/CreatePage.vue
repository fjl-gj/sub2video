<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Info } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import AssetPickDialog from '@/features/create/AssetPickDialog.vue'
import CreateModeCards from '@/features/create/CreateModeCards.vue'
import CreateModeGate from '@/features/create/CreateModeGate.vue'
import ImageStack from '@/features/create/ImageStack.vue'
import {
  clearCreateDraft,
  emptySlot,
  loadCreateDraft,
  MAX_REF_IMAGES,
  saveCreateDraft,
  type CreateFrameSlot,
  type CreateMode,
  type ImageVariant,
} from '@/shared/lib/create-draft'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VChoice from '@/shared/ui/S2VChoice.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VSegmented from '@/shared/ui/S2VSegmented.vue'
import S2VSelect from '@/shared/ui/S2VSelect.vue'
import S2VTextarea from '@/shared/ui/S2VTextarea.vue'

interface FrameSlot extends CreateFrameSlot {}

const router = useRouter()
const app = useAppStore()
const mode = ref<CreateMode>('image')
const imageVariant = ref<ImageVariant>('keyframes')
const prompt = ref('')
const ratio = ref('16:9')
const durationSec = ref('15')
const loading = ref(false)
const draftReady = ref(false)

const firstFrame = ref<FrameSlot>(emptySlot())
const lastFrame = ref<FrameSlot>(emptySlot())
const extraFrames = ref<FrameSlot[]>([])
const multiFrames = ref<FrameSlot[]>([emptySlot()])

let saveTimer: ReturnType<typeof setTimeout> | null = null
const pickingSlotRef = ref<FrameSlot | null>(null)

const assetDialogOpen = computed({
  get: () => pickingSlotRef.value !== null,
  set: (open: boolean) => {
    if (!open) pickingSlotRef.value = null
  },
})

const imageVariants = ['首尾帧', '多图参考']
const ratioOptions = [
  { label: '16:9 横屏', value: '16:9' },
  { label: '9:16 竖屏', value: '9:16' },
  { label: '1:1 方形', value: '1:1' },
]

const parsedDuration = computed(() => Number(durationSec.value))
const durationValid = computed(() => Number.isFinite(parsedDuration.value) && parsedDuration.value >= 1 && parsedDuration.value <= 15)
const cost = computed(() => durationValid.value ? parsedDuration.value * 0.5 : 0)
const canSubmit = computed(() => mode.value === 'image' && durationValid.value && hasRequiredImageInput.value)
const workflowLabel = computed(() => {
  if (mode.value === 'text') return 'H3 文生视频'
  return imageVariant.value === 'keyframes' ? 'H3 首尾帧图生视频' : 'H3 多图参考图生视频'
})

const hasRequiredImageInput = computed(() => {
  if (mode.value !== 'image') return false
  if (imageVariant.value === 'keyframes') return frameReady(firstFrame.value)
  return multiFrames.value.some((slot) => frameReady(slot))
})

function frameReady(slot: FrameSlot) {
  return slot.uploaded || Boolean(slot.assetId)
}

function markUpload(slot: FrameSlot) {
  slot.uploaded = true
  slot.assetId = ''
  pickingSlotRef.value = null
}

function openAssetPick(slot: FrameSlot) {
  pickingSlotRef.value = slot
}

function confirmAssetPick(assetId: string) {
  if (!pickingSlotRef.value) return
  pickingSlotRef.value.assetId = assetId
  pickingSlotRef.value.uploaded = true
  pickingSlotRef.value = null
}

function clearSlot(slot: FrameSlot) {
  slot.uploaded = false
  slot.assetId = ''
  if (pickingSlotRef.value === slot) pickingSlotRef.value = null
}

function addExtra() {
  if (extraFrames.value.length >= MAX_REF_IMAGES) return
  extraFrames.value.push(emptySlot())
}

function removeExtra(index: number) {
  extraFrames.value.splice(index, 1)
  pickingSlotRef.value = null
}

function addMulti() {
  if (multiFrames.value.length >= MAX_REF_IMAGES) return
  multiFrames.value.push(emptySlot())
}

function removeMulti(index: number) {
  if (multiFrames.value.length <= 1) return
  multiFrames.value.splice(index, 1)
  pickingSlotRef.value = null
}

function chooseTemplate(text: string) {
  prompt.value = prompt.value ? `${prompt.value}，${text}` : text
}

function switchToImageMode() {
  mode.value = 'image'
}

function taskStamp() {
  const now = new Date()
  const pad = (value: number) => String(value).padStart(2, '0')
  return `${pad(now.getMonth() + 1)}${pad(now.getDate())} ${pad(now.getHours())}${pad(now.getMinutes())}`
}

function buildTaskTitle() {
  const seq = app.tasks.length + 1
  if (mode.value === 'text') return `文生 ${taskStamp()} #${seq}`
  const variant = imageVariant.value === 'keyframes' ? '首尾帧' : '多图'
  return `图生·${variant} ${taskStamp()} #${seq}`
}

function applyDraftFromStorage() {
  const plazaPrompt = app.draftPrompt.trim()
  const saved = loadCreateDraft()
  if (saved) {
    mode.value = saved.mode
    imageVariant.value = saved.imageVariant
    ratio.value = saved.ratio
    durationSec.value = saved.durationSec
    firstFrame.value = { ...saved.firstFrame, picking: false }
    lastFrame.value = { ...saved.lastFrame, picking: false }
    extraFrames.value = saved.extraFrames.map((slot) => ({ ...slot, picking: false }))
    multiFrames.value = saved.multiFrames.length
      ? saved.multiFrames.map((slot) => ({ ...slot, picking: false }))
      : [emptySlot()]
    prompt.value = plazaPrompt || saved.prompt
  }
  else {
    prompt.value = plazaPrompt
  }
  if (plazaPrompt) app.draftPrompt = ''
}

function scheduleDraftSave() {
  if (!draftReady.value) return
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => {
    saveCreateDraft({
      mode: mode.value,
      imageVariant: imageVariant.value,
      prompt: prompt.value,
      ratio: ratio.value,
      durationSec: durationSec.value,
      firstFrame: firstFrame.value,
      lastFrame: lastFrame.value,
      extraFrames: extraFrames.value,
      multiFrames: multiFrames.value,
    })
  }, 320)
}

onMounted(() => {
  applyDraftFromStorage()
  draftReady.value = true
})

onBeforeUnmount(() => {
  if (saveTimer) clearTimeout(saveTimer)
})

watch([mode, imageVariant, prompt, ratio, durationSec, firstFrame, lastFrame, extraFrames, multiFrames], scheduleDraftSave, { deep: true })

function submit() {
  if (!canSubmit.value) {
    app.notify(imageVariant.value === 'keyframes' ? '请先上传或选择首帧图片' : '请至少选择一张参考图')
    return
  }
  loading.value = true
  window.setTimeout(() => {
    loading.value = false
    if (app.createTask(cost.value, {
      title: buildTaskTitle(),
      workflow: workflowLabel.value,
      durationSec: parsedDuration.value,
      aspectRatio: ratio.value,
    })) {
      app.draftPrompt = ''
      prompt.value = ''
      clearCreateDraft()
      router.push('/app/works')
    }
  }, 420)
}
</script>

<template>
  <section class="s2v-page s2v-page-studio">
    <S2VPageHeader title="创作">
      <template #actions>
        <S2VBadge tone="success">钱包余额: ¥ {{ app.balance.toFixed(2) }}</S2VBadge>
      </template>
    </S2VPageHeader>

    <S2VCard class="mb-5">
      <div class="s2v-panel-header">
        <div>
          <h2 class="s2v-section-title">选择生成形式</h2>
          <p class="mt-1 text-xs text-s2v-muted">图生支持首尾帧或多张参考图；文生即将开放</p>
        </div>
      </div>
      <div class="s2v-panel-body grid gap-4">
        <CreateModeCards v-model:mode="mode" />
        <div v-if="mode === 'image'" class="border-t border-s2v-line/60 pt-4">
          <p class="mb-2 text-xs font-semibold text-s2v-ink">图生细分</p>
          <S2VSegmented
            :model-value="imageVariant === 'keyframes' ? '首尾帧' : '多图参考'"
            :options="imageVariants"
            @update:model-value="imageVariant = $event === '首尾帧' ? 'keyframes' : 'multi'"
          />
        </div>
      </div>
    </S2VCard>

    <div class="grid gap-5 xl:grid-cols-[minmax(0,1.6fr)_minmax(280px,.8fr)]">
      <div class="grid gap-5">
        <S2VCard v-if="mode === 'image'">
          <div class="s2v-panel-header">
            <div>
              <h2 class="s2v-section-title">输入素材</h2>
              <p class="mt-1 text-xs text-s2v-muted">{{ workflowLabel }}</p>
            </div>
          </div>
          <div class="s2v-panel-body">
            <ImageStack
              :variant="imageVariant"
              :first-frame="firstFrame"
              :last-frame="lastFrame"
              :extra-frames="extraFrames"
              :multi-frames="multiFrames"
              @upload="markUpload"
              @pick="openAssetPick"
              @clear="clearSlot"
              @add-extra="addExtra"
              @remove-extra="removeExtra"
              @add-multi="addMulti"
              @remove-multi="removeMulti"
            />
          </div>
        </S2VCard>

        <CreateModeGate
          v-else
          title="文生视频即将开放"
          description="仅需画面描述即可生成视频，无需上传图片素材。"
          @switch-image="switchToImageMode"
        />

        <S2VCard>
          <div class="s2v-panel-header">
            <div>
              <h2 class="s2v-section-title">画面描述</h2>
              <p class="mt-1 text-xs text-s2v-muted">描述主体动作、镜头运动和环境变化</p>
            </div>
          </div>
          <div class="s2v-panel-body">
            <S2VTextarea v-model="prompt" :rows="6" placeholder="例如：镜头缓慢向前推进，雨滴落在玻璃上，霓虹灯在背景中柔和闪烁……" />
            <div class="mt-3 flex flex-wrap gap-2">
              <S2VChoice v-for="tag in ['电影感', '缓慢推进', '自然光', '细节丰富']" :key="tag" class="px-2" @click="chooseTemplate(tag)">+ {{ tag }}</S2VChoice>
            </div>
          </div>
        </S2VCard>
      </div>

      <S2VCard class="h-fit xl:sticky xl:top-6">
        <div class="s2v-panel-header">
          <h2 class="s2v-section-title">生成设置</h2>
        </div>
        <div class="s2v-panel-body grid gap-4">
          <S2VSelect v-model="ratio" label="画面比例" :options="ratioOptions" />
          <S2VInput v-model="durationSec" label="视频时长" type="number" min="1" placeholder="1-15">
            <template #suffix><span class="text-[11px] text-s2v-muted">秒</span></template>
          </S2VInput>
          <div class="s2v-create-submit-panel">
            <div class="flex items-center justify-between">
              <span class="text-xs font-semibold text-s2v-ink">预计费用</span>
              <strong class="s2v-mono text-xl text-s2v-ink">¥ {{ cost.toFixed(2) }}</strong>
            </div>
            <div class="flex items-center gap-1 text-[11px] text-s2v-muted"><Info class="h-3.5 w-3.5" />768P · ¥0.50/秒</div>
            <S2VButton variant="primary" size="lg" :loading="loading" :disabled="!canSubmit" :block="true" @click="submit">
              提交生成任务
            </S2VButton>
          </div>
        </div>
      </S2VCard>
    </div>

    <AssetPickDialog v-model:open="assetDialogOpen" @pick="confirmAssetPick" />
  </section>
</template>
