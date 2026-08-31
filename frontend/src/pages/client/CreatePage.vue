<script setup lang="ts">
import { computed, ref } from 'vue'
import { Check, FolderOpen, ImagePlus, Info, Sparkles, Upload, WandSparkles } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VChoice from '@/shared/ui/S2VChoice.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VTextarea from '@/shared/ui/S2VTextarea.vue'

const router = useRouter()
const app = useAppStore()
const workflow = ref('image')
const prompt = ref(app.draftPrompt)
const title = ref('')
const ratio = ref('16:9')
const duration = ref('15 秒')
const uploaded = ref(false)
const picking = ref(false)
const selectedAssetId = ref('')
const loading = ref(false)
const cost = computed(() => duration.value === '10 秒' ? 5 : 7.5)
const imageAssets = computed(() => app.assets.filter((item) => item.type === '图片'))
const selectedAsset = computed(() => app.assets.find((item) => item.id === selectedAssetId.value))
const workflows = [
  { id: 'image', name: '图生视频', desc: '一张图片 + 文字描述', enabled: true },
  { id: 'text', name: '文生视频', desc: '纯文字生成画面', enabled: false },
  { id: 'multi', name: '多图生视频', desc: '多张参考图控制连续性', enabled: false },
]

function chooseTemplate(text: string) {
  prompt.value = prompt.value ? `${prompt.value}，${text}` : text
}

function pickAsset(id: string) {
  selectedAssetId.value = id
  uploaded.value = true
  picking.value = false
}

function submit() {
  loading.value = true
  window.setTimeout(() => {
    loading.value = false
    if (app.createTask(title.value || prompt.value.slice(0, 18), cost.value)) {
      app.draftPrompt = ''
      router.push('/app/works')
    }
  }, 420)
}
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader badge="Video Studio" kicker="纯文生视频 / 文+图生视频（首尾帧 / 参考图）" title="创建视频生成任务">
      <template #actions>
        <S2VBadge tone="success">钱包余额: ¥ {{ app.balance.toFixed(2) }}</S2VBadge>
      </template>
    </S2VPageHeader>

    <div class="mb-5 grid gap-3 md:grid-cols-3">
      <S2VChoice
        v-for="item in workflows"
        :key="item.id"
        variant="tile"
        class="w-full"
        :selected="workflow === item.id"
        :disabled="!item.enabled"
        @click="workflow = item.id"
      >
        <div class="flex items-start justify-between">
          <span class="s2v-icon-well s2v-icon-well-nav"><ImagePlus class="h-4 w-4" /></span>
          <S2VBadge v-if="item.enabled" tone="success">可用</S2VBadge>
          <S2VBadge v-else>即将开放</S2VBadge>
        </div>
        <strong class="mt-3 block text-sm text-s2v-ink">{{ item.name }}</strong>
        <span class="mt-1 block text-xs font-normal text-s2v-muted">{{ item.desc }}</span>
      </S2VChoice>
    </div>

    <div class="grid gap-5 lg:grid-cols-[minmax(0,1.5fr)_minmax(280px,.75fr)]">
      <div class="grid gap-5">
        <S2VCard>
          <div class="s2v-panel-header">
            <div>
              <h2 class="s2v-section-title">输入素材</h2>
              <p class="mt-1 text-xs text-s2v-muted">上传一张首帧图片，生成结果会自动关联到作品。</p>
            </div>
            <S2VBadge tone="primary">最多 5 张免费</S2VBadge>
          </div>
          <div class="s2v-panel-body grid gap-3 sm:grid-cols-2">
            <button class="s2v-dropzone" type="button" @click="uploaded = true; selectedAssetId = ''">
              <template v-if="!uploaded">
                <Upload class="h-6 w-6 text-s2v-primary" />
                <strong class="text-xs">上传首帧图片</strong>
                <span class="text-[11px] text-s2v-muted">JPG / PNG · 不超过 10 MB</span>
              </template>
              <template v-else>
                <Check class="h-7 w-7 text-s2v-success" />
                <strong class="text-xs">{{ selectedAsset ? selectedAsset.name : '首帧图片已选择' }}</strong>
                <span class="text-[11px] text-s2v-muted">点击重新选择</span>
              </template>
            </button>
            <button class="s2v-dropzone" type="button" @click="picking = !picking">
              <FolderOpen class="h-6 w-6 text-s2v-primary" />
              <strong class="text-xs">从资产中选择</strong>
              <span class="text-[11px] text-s2v-muted">{{ picking ? '选择一张已保存的图片' : '使用已保存的图片素材' }}</span>
            </button>
            <div v-if="picking" class="grid gap-2 sm:col-span-2">
              <S2VChoice v-for="item in imageAssets" :key="item.id" :selected="selectedAssetId === item.id" @click="pickAsset(item.id)">
                {{ item.name }} · {{ item.folder }}
              </S2VChoice>
              <p v-if="!imageAssets.length" class="text-[11px] text-s2v-muted">资产库里还没有图片</p>
            </div>
          </div>
        </S2VCard>

        <S2VCard>
          <div class="s2v-panel-header">
            <div>
              <h2 class="s2v-section-title">画面描述</h2>
              <p class="mt-1 text-xs text-s2v-muted">描述主体动作、镜头运动和环境变化。</p>
            </div>
            <WandSparkles class="h-4 w-4 text-s2v-primary" />
          </div>
          <div class="s2v-panel-body">
            <S2VTextarea v-model="prompt" placeholder="例如：镜头缓慢向前推进，雨滴落在玻璃上，霓虹灯在背景中柔和闪烁……" />
            <div class="mt-2 flex flex-wrap gap-2">
              <S2VChoice v-for="tag in ['电影感', '缓慢推进', '自然光', '细节丰富']" :key="tag" class="px-2" @click="chooseTemplate(tag)">+ {{ tag }}</S2VChoice>
            </div>
          </div>
        </S2VCard>
      </div>

      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">生成设置</h2>
            <p class="mt-1 text-xs text-s2v-muted">H3 图生视频</p>
          </div>
          <Sparkles class="h-4 w-4 text-s2v-primary" />
        </div>
        <div class="s2v-panel-body grid gap-4">
          <S2VInput v-model="title" label="作品标题" placeholder="给这次创作起个名字" />
          <div class="grid gap-1.5">
            <span class="text-xs font-semibold text-s2v-ink/70">画面比例</span>
            <div class="grid grid-cols-3 gap-2">
              <S2VChoice v-for="item in ['16:9', '9:16', '1:1']" :key="item" :selected="ratio === item" @click="ratio = item">{{ item }}</S2VChoice>
            </div>
          </div>
          <div class="grid gap-1.5">
            <span class="text-xs font-semibold text-s2v-ink/70">视频时长</span>
            <div class="grid grid-cols-2 gap-2">
              <S2VChoice v-for="item in ['6 秒', '10 秒', '15 秒', '自定义']" :key="item" :selected="duration === item" :disabled="item === '自定义'" @click="duration = item">{{ item }}</S2VChoice>
            </div>
          </div>
          <div class="s2v-note">
            <div class="flex items-center justify-between text-xs">
              <span>预计费用</span>
              <strong class="s2v-mono text-lg text-s2v-ink">¥ {{ cost.toFixed(2) }}</strong>
            </div>
            <div class="mt-1 flex items-center gap-1 text-[11px] text-s2v-muted"><Info class="h-3.5 w-3.5" />按 768P 输出秒数计费</div>
          </div>
          <S2VButton variant="primary" size="lg" :loading="loading" :block="true" @click="submit">提交生成任务</S2VButton>
          <p class="text-center text-[11px] text-s2v-muted">提交后预扣费用，生成失败会自动退回钱包</p>
        </div>
      </S2VCard>
    </div>
  </section>
</template>
