<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Info, Lock } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import CreateModeGate from '@/features/create/CreateModeGate.vue'
import ImageStack from '@/features/create/ImageStack.vue'
import {
  clearCreateDraft,
  loadCreateDraft,
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
const modeTab = ref('图生视频')
const imageVariant = ref<ImageVariant>('keyframes')
const prompt = ref('')
const ratio = ref('16:9')
const durationSec = ref('15')
const loading = ref(false)
const draftReady = ref(false)

const firstFrame = ref<FrameSlot>({ uploaded: false, assetId: '', picking: false })
const lastFrame = ref<FrameSlot>({ uploaded: false, assetId: '', picking: false })
const multiFrames = ref<FrameSlot[]>([
  { uploaded: false, assetId: '', picking: false },
  { uploaded: false, assetId: '', picking: false },
  { uploaded: false, assetId: '', picking: false },
])
const referenceVideo = ref({ uploaded: false, picking: false })

let saveTimer: ReturnType<typeof setTimeout> | null = null

const modeTabs = ['图生视频', '文生视频', '视频生视频']
const modeFromTab: Record<string, CreateMode> = {
  图生视频: 'image',
  文生视频: 'text',
  视频生视频: 'video',
}
const tabFromMode: Record<CreateMode, string> = {
  image: '图生视频',
  text: '文生视频',
  video: '视频生视频',
}
const imageVariants = ['首尾帧', '多图参考']
const ratioOptions = [
  { label: '16:9 横屏', value: '16:9' },
  { label: '9:16 竖屏', value: '9:16' },
  { label: '1:1 方形', value: '1:1' },
]

const parsedDuration = computed(() => Number(durationSec.value))
const durationValid = computed(() => Number.isFinite(parsedDuration.value) && parsedDuration.value >= 1 && parsedDuration.value <= 15)
const cost = computed(() => durationValid.value ? parsedDuration.value * 0.5 : 0)
const modeLocked = computed(() => mode.value !== 'image')
const canSubmit = computed(() => mode.value === 'image' && durationValid.value && hasRequiredImageInput.value)
const workflowLabel = computed(() => {
  if (mode.value === 'text') return 'H3 文生视频'
  if (mode.value === 'video') return 'H3 视频生视频'
  return imageVariant.value === 'keyframes' ? 'H3 首尾帧图生视频' : 'H3 多图参考图生视频'
})

const hasRequiredImageInput = computed(() => {
  if (mode.value !== 'image') return false
  if (imageVariant.value === 'keyframes') return frameReady(firstFrame.value)
  return multiFrames.value.filter((slot) => frameReady(slot)).length >= 1
})

watch(modeTab, (tab) => {
  mode.value = modeFromTab[tab] ?? 'image'
})

function frameReady(slot: FrameSlot) {
  return slot.uploaded || Boolean(slot.assetId)
}

function markUpload(slot: FrameSlot) {
  slot.uploaded = true
  slot.assetId = ''
  closePicking()
}

function pickAsset(slot: FrameSlot, id: string) {
  slot.assetId = id
  slot.uploaded = true
  closePicking()
}

function togglePick(slot: FrameSlot) {
  const next = !slot.picking
  closePicking()
  slot.picking = next
}

function closePicking() {
  firstFrame.value.picking = false
  lastFrame.value.picking = false
  multiFrames.value.forEach((slot) => { slot.picking = false })
}

function chooseTemplate(text: string) {
  prompt.value = prompt.value ? `${prompt.value}，${text}` : text
}

function switchToImageMode() {
  modeTab.value = '图生视频'
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
  if (mode.value === 'video') return `视频生 ${taskStamp()} #${seq}`
  const variant = imageVariant.value === 'keyframes' ? '首尾帧' : '多图'
  return `图生·${variant} ${taskStamp()} #${seq}`
}

function applyDraftFromStorage() {
  const plazaPrompt = app.draftPrompt.trim()
  const saved = loadCreateDraft()
  if (saved) {
    mode.value = saved.mode
    modeTab.value = tabFromMode[saved.mode]
    imageVariant.value = saved.imageVariant
    ratio.value = saved.ratio
    durationSec.value = saved.durationSec
    firstFrame.value = { ...saved.firstFrame, picking: false }
    lastFrame.value = { ...saved.lastFrame, picking: false }
    multiFrames.value = saved.multiFrames.map((slot) => ({ ...slot, picking: false }))
    referenceVideo.value = { uploaded: saved.referenceVideoUploaded, picking: false }
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
      multiFrames: multiFrames.value,
      referenceVideoUploaded: referenceVideo.value.uploaded,
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

watch([mode, imageVariant, prompt, ratio, durationSec, firstFrame, lastFrame, multiFrames, referenceVideo], scheduleDraftSave, { deep: true })

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

    <div class="mb-4 space-y-2">
      <S2VSegmented v-model="modeTab" :options="modeTabs" />
      <p v-if="modeLocked" class="flex items-center gap-1.5 text-xs text-s2v-muted">
        <Lock class="h-3.5 w-3.5" />
        当前模式即将开放，可先配置描述与参数；提交请使用图生视频。
      </p>
    </div>

    <div v-if="mode === 'image'" class="mb-4">
      <S2VSegmented
        :model-value="imageVariant === 'keyframes' ? '首尾帧' : '多图参考'"
        :options="imageVariants"
        @update:model-value="imageVariant = $event === '首尾帧' ? 'keyframes' : 'multi'"
      />
    </div>

    <div class="grid gap-5 xl:grid-cols-[minmax(0,1.55fr)_minmax(300px,.75fr)]">
      <div class="grid gap-5">
        <S2VCard v-if="mode === 'image'">
          <div class="s2v-panel-header">
            <div>
              <h2 class="s2v-section-title">输入素材</h2>
              <p class="mt-1 text-xs text-s2v-muted">
                {{ imageVariant === 'keyframes' ? '首尾帧叠放预览，点击卡片上传或从资产选取。' : '参考图扇形叠放，至少选择 1 张。' }}
              </p>
            </div>
          </div>
          <div class="s2v-panel-body">
            <ImageStack
              :variant="imageVariant"
              :first-frame="firstFrame"
              :last-frame="lastFrame"
              :multi-frames="multiFrames"
              @upload="markUpload"
              @pick="pickAsset"
              @toggle-pick="togglePick"
            />
          </div>
        </S2VCard>

        <CreateModeGate
          v-else-if="mode === 'text'"
          title="文生视频即将开放"
          description="仅需画面描述即可生成视频，无需上传图片或视频素材。"
          @switch-image="switchToImageMode"
        />

        <CreateModeGate
          v-else
          title="视频生视频即将开放"
          description="上传参考视频，配合描述进行风格化或镜头再创作。"
          @switch-image="switchToImageMode"
        />

        <S2VCard>
          <div class="s2v-panel-header">
            <div>
              <h2 class="s2v-section-title">画面描述</h2>
              <p class="mt-1 text-xs text-s2v-muted">描述主体动作、镜头运动和环境变化。从广场套用时会预填到这里。</p>
            </div>
          </div>
          <div class="s2v-panel-body">
            <S2VTextarea v-model="prompt" :rows="8" placeholder="例如：镜头缓慢向前推进，雨滴落在玻璃上，霓虹灯在背景中柔和闪烁……" />
            <div class="mt-3 flex flex-wrap gap-2">
              <S2VChoice v-for="tag in ['电影感', '缓慢推进', '自然光', '细节丰富']" :key="tag" class="px-2" @click="chooseTemplate(tag)">+ {{ tag }}</S2VChoice>
            </div>
          </div>
        </S2VCard>
      </div>

      <S2VCard class="h-fit xl:sticky xl:top-6">
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">生成设置</h2>
            <p class="mt-1 text-xs text-s2v-muted">{{ workflowLabel }}</p>
          </div>
        </div>
        <div class="s2v-panel-body grid gap-4">
          <S2VSelect v-model="ratio" label="画面比例" :options="ratioOptions" />
          <S2VInput v-model="durationSec" label="视频时长" type="number" min="1" placeholder="1-15">
            <template #suffix><span class="text-[11px] text-s2v-muted">秒</span></template>
          </S2VInput>
          <div class="s2v-note">
            <div class="flex items-center justify-between text-xs">
              <span>预计费用</span>
              <strong class="s2v-mono text-lg text-s2v-ink">¥ {{ cost.toFixed(2) }}</strong>
            </div>
            <div class="mt-1 flex items-center gap-1 text-[11px] text-s2v-muted"><Info class="h-3.5 w-3.5" />768P · ¥0.50/秒</div>
          </div>
          <S2VButton variant="primary" size="lg" :loading="loading" :disabled="!canSubmit" :block="true" @click="submit">
            提交生成任务
          </S2VButton>
          <p class="text-[11px] leading-5 text-s2v-muted">
            任务名称按「模式 + 时间 + 序号」自动生成；未提交内容会静默保存在本机，提交成功后清除。
          </p>
        </div>
      </S2VCard>
    </div>
  </section>
</template>
