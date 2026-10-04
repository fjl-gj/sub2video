<script setup lang="ts">
import { ArrowLeft, ArrowRight, Check } from 'lucide-vue-next'
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { findGuide, guideCreateRoute } from './catalog'
import GuideImage from './GuideImage.vue'
const route = useRoute()
const guide = computed(() => findGuide(route.params.id))
</script>
<template>
  <article v-if="guide" class="s2v-guides s2v-guide-detail">
    <RouterLink to="/#guides" class="s2v-guide-back"><ArrowLeft class="h-4 w-4" />全部镜头教程</RouterLink>
    <div class="s2v-guide-intro"><GuideImage :src="guide.poster" :alt="`${guide.title}构图参考`" /><div><p class="s2v-guide-eyebrow">{{ guide.category }} / 镜头教程</p><h1>{{ guide.title }}</h1><p>{{ guide.description }}</p><div class="s2v-guide-tags"><span>{{ guide.modeName }}</span><span>{{ guide.shots.length }} 个独立镜头</span><span>自行准备素材</span></div><RouterLink :to="guideCreateRoute(guide)" class="s2v-guide-link-button s2v-guide-link-primary">从第一镜开始<ArrowRight class="h-4 w-4" /></RouterLink><p class="s2v-guide-materials">{{ guide.materials }}</p></div></div>
    <section class="s2v-guide-preparation"><h2>开始前，准备好这些</h2><ul><li v-for="item in guide.preparation" :key="item"><Check class="h-4 w-4" />{{ item }}</li></ul></section>
    <section aria-label="镜头拆解"><div class="s2v-guide-section-heading"><div><p class="s2v-guide-eyebrow">一步一步，完成画面</p><h2>镜头拆解</h2><p>每个镜头单独生成、单独计费。你可以从任意一镜开始。</p></div></div><div class="s2v-guide-shot-grid"><article v-for="(item, index) in guide.shots" :key="item.id" class="s2v-guide-shot"><div class="s2v-guide-shot-number">镜头 {{ String(index + 1).padStart(2, '0') }} <span>{{ item.durationSeconds }} 秒建议</span></div><h3>{{ item.title }}</h3><p>{{ item.intent }}</p><blockquote>{{ item.prompt }}</blockquote><div class="s2v-guide-shot-parameters">{{ item.aspectRatio }} · {{ item.resolution }} · 不使用音频</div><RouterLink :to="guideCreateRoute(guide, item)" class="s2v-guide-link-button" aria-label="制作这个镜头">制作这个镜头<ArrowRight class="h-4 w-4" /></RouterLink></article></div></section>
    <div class="s2v-guide-after"><section><h2>生成之后，怎样变成短片？</h2><p>{{ guide.finishing }}</p></section><section><h2>关于结果与费用</h2><p>这些是创作练习，参考照片不代表实际生成效果。重做可能产生新费用；请先查看创作页的可用能力与估价，再决定是否提交。</p></section></div>
  </article>
  <section v-else class="s2v-guides s2v-guide-empty"><h1>未找到这个镜头教程</h1><p>链接可能已失效，可以回到教程列表重新选择。</p><RouterLink to="/#guides" class="s2v-guide-link-button">查看镜头教程</RouterLink></section>
</template>
