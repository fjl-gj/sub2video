<script setup lang="ts">
import { computed, ref } from 'vue'
import { Check, FolderOpen, ImagePlus, Info, Sparkles, Upload, WandSparkles } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VBadge from '@/shared/ui/S2VBadge.vue'

const router = useRouter()
const app = useAppStore()
const workflow = ref('image')
const prompt = ref(app.draftPrompt)
const title = ref('')
const ratio = ref('16:9')
const duration = ref('15 秒')
const uploaded = ref(false)
const loading = ref(false)
const cost = computed(() => duration.value === '10 秒' ? 5 : 7.5)

function chooseTemplate(text: string) { prompt.value = prompt.value ? `${prompt.value}，${text}` : text }
function submit() {
  loading.value = true
  window.setTimeout(() => { loading.value = false; if (app.createTask(title.value || prompt.value.slice(0, 18), cost.value)) { app.draftPrompt = ''; router.push('/app/works') } }, 420)
}
</script>

<template>
  <section class="s2v-page">
    <header class="s2v-page-header"><div><p class="s2v-eyebrow">创作空间 / 创作</p><h1 class="s2v-page-title">创建视频</h1><p class="s2v-subcopy">先选择工作流，再配置素材和镜头描述。每个工作流都有独立的输入要求。</p></div><S2VBadge tone="success"><span class="h-1.5 w-1.5 rounded-full bg-emerald-500" />服务正常</S2VBadge></header>
    <div class="mb-5 grid gap-3 md:grid-cols-3"><button v-for="item in [{id:'image',name:'图生视频',desc:'一张图片 + 文字描述',enabled:true},{id:'text',name:'文生视频',desc:'纯文字生成画面',enabled:false},{id:'multi',name:'多图生视频',desc:'多张参考图控制连续性',enabled:false}]" :key="item.id" :disabled="!item.enabled" :class="['rounded-panel border p-4 text-left transition', workflow === item.id ? 'border-s2v-primary bg-s2v-primary-soft/40' : 'border-s2v-line bg-s2v-surface hover:border-s2v-primary/40', !item.enabled ? 'cursor-not-allowed opacity-55' : '']" @click="workflow = item.id"><div class="flex items-start justify-between"><span class="grid h-8 w-8 place-items-center rounded-[7px] bg-s2v-primary-soft text-s2v-primary"><ImagePlus class="h-4 w-4" /></span><S2VBadge v-if="item.enabled" tone="success">可用</S2VBadge><S2VBadge v-else>即将开放</S2VBadge></div><strong class="mt-3 block text-sm">{{ item.name }}</strong><span class="mt-1 block text-xs text-s2v-muted">{{ item.desc }}</span></button></div>

    <div class="grid gap-5 lg:grid-cols-[minmax(0,1.5fr)_minmax(280px,.75fr)]">
      <div class="grid gap-5"><S2VCard><div class="s2v-panel-header"><div><h2 class="s2v-section-title">输入素材</h2><p class="mt-1 text-xs text-s2v-muted">上传一张首帧图片，生成结果会自动关联到作品。</p></div><S2VBadge tone="primary">最多 5 张免费</S2VBadge></div><div class="grid gap-3 p-4 sm:grid-cols-2"><button class="grid min-h-44 place-items-center gap-2 rounded-[7px] border border-dashed border-s2v-line-strong bg-s2v-canvas p-5 text-center transition hover:border-s2v-primary hover:bg-s2v-primary-soft/40" @click="uploaded = true"><template v-if="!uploaded"><Upload class="h-6 w-6 text-s2v-primary" /><strong class="text-xs">上传首帧图片</strong><span class="text-[11px] text-s2v-muted">JPG / PNG · 不超过 10 MB</span></template><template v-else><Check class="h-7 w-7 text-emerald-600" /><strong class="text-xs">首帧图片已选择</strong><span class="text-[11px] text-s2v-muted">点击重新选择</span></template></button><button class="grid min-h-44 place-items-center gap-2 rounded-[7px] border border-s2v-line bg-s2v-surface p-5 text-center transition hover:border-s2v-primary hover:bg-s2v-primary-soft/40" @click="app.notify('资产选择器将在下一步接入')"><FolderOpen class="h-6 w-6 text-s2v-primary" /><strong class="text-xs">从资产中选择</strong><span class="text-[11px] text-s2v-muted">使用已保存的图片素材</span></button></div></S2VCard>
      <S2VCard><div class="s2v-panel-header"><div><h2 class="s2v-section-title">画面描述</h2><p class="mt-1 text-xs text-s2v-muted">描述主体动作、镜头运动和环境变化。</p></div><WandSparkles class="h-4 w-4 text-s2v-primary" /></div><div class="p-4"><textarea v-model="prompt" class="min-h-36 w-full resize-y rounded-[7px] border border-s2v-line-strong bg-s2v-surface p-3 text-sm leading-6 text-s2v-ink placeholder:text-s2v-muted/65" placeholder="例如：镜头缓慢向前推进，雨滴落在玻璃上，霓虹灯在背景中柔和闪烁……" /><div class="mt-2 flex flex-wrap gap-2"><button v-for="tag in ['电影感','缓慢推进','自然光','细节丰富']" :key="tag" class="rounded-full border border-s2v-line px-2 py-1 text-[11px] text-s2v-muted hover:border-s2v-primary hover:text-s2v-primary" @click="chooseTemplate(tag)">+ {{ tag }}</button></div></div></S2VCard></div>
      <S2VCard><div class="s2v-panel-header"><div><h2 class="s2v-section-title">生成设置</h2><p class="mt-1 text-xs text-s2v-muted">H3 图生视频</p></div><Sparkles class="h-4 w-4 text-s2v-primary" /></div><div class="grid gap-4 p-4"><label class="grid gap-1.5 text-xs font-semibold text-s2v-ink/70"><span>作品标题</span><input v-model="title" class="h-10 rounded-[7px] border border-s2v-line-strong px-3 text-sm font-normal" placeholder="给这次创作起个名字" /></label><div class="grid gap-1.5"><span class="text-xs font-semibold text-s2v-ink/70">画面比例</span><div class="grid grid-cols-3 gap-2"><button v-for="item in ['16:9','9:16','1:1']" :key="item" :class="['h-9 rounded-[7px] border text-xs font-semibold', ratio === item ? 'border-s2v-primary bg-s2v-primary-soft text-s2v-primary' : 'border-s2v-line text-s2v-muted']" @click="ratio = item">{{ item }}</button></div></div><div class="grid gap-1.5"><span class="text-xs font-semibold text-s2v-ink/70">视频时长</span><div class="grid grid-cols-2 gap-2"><button v-for="item in ['6 秒','10 秒','15 秒','自定义']" :key="item" :disabled="item === '自定义'" :class="['h-9 rounded-[7px] border text-xs font-semibold', duration === item ? 'border-s2v-primary bg-s2v-primary-soft text-s2v-primary' : 'border-s2v-line text-s2v-muted', item === '自定义' ? 'opacity-50' : '']" @click="duration = item">{{ item }}</button></div></div><div class="rounded-[7px] bg-s2v-canvas p-3"><div class="flex items-center justify-between text-xs text-s2v-muted"><span>预计费用</span><strong class="font-mono text-lg text-s2v-ink">¥ {{ cost.toFixed(2) }}</strong></div><div class="mt-1 flex items-center gap-1 text-[11px] text-s2v-muted"><Info class="h-3.5 w-3.5" />按 768P 输出秒数计费</div></div><S2VButton variant="primary" size="lg" :loading="loading" :block="true" @click="submit">提交生成任务</S2VButton><p class="text-center text-[11px] text-s2v-muted">提交后预扣费用，生成失败会自动退回钱包</p></div></S2VCard>
    </div>
  </section>
</template>
