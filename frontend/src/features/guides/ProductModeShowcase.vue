<script setup lang="ts">
import { ArrowUpRight } from 'lucide-vue-next'
import { TabsContent, TabsList, TabsRoot, TabsTrigger } from 'reka-ui'
import { shallowRef } from 'vue'
import ProductReferenceImage from './ProductReferenceImage.vue'

const poster = '/brand/auth/creator-studio-poster.jpg'
const lastFrame = '/brand/auth/creator-studio-last.jpg'
const selectedMode = shallowRef('text')
// Illustrative inputs only. Capabilities and prices load inside creation.
const modes = [
  {
    id: 'text', label: '文生视频', name: '文生视频',
    description: '描述画面，让故事动起来。', materialLabel: '画面描述',
    prompt: '暮色中的城市工作室。人物缓缓调节镜头，镜头向前推进。',
    mobilePrompt: '描述人物、场景与镜头运动。', images: [], reference: poster,
  },
  {
    id: 'first_last', label: '首尾帧', name: '首尾帧生成视频',
    description: '给出起点与终点，连接画面变化。', materialLabel: '首帧与尾帧',
    prompt: '从冷色夜景，过渡到温暖的镜头光影。', mobilePrompt: '',
    images: [{ src: poster, alt: '首帧构图', label: '首帧' }, { src: lastFrame, alt: '尾帧构图', label: '尾帧' }], reference: lastFrame,
  },
  {
    id: 'multi_image', label: '多图', name: '多图生成视频',
    description: '带上参考图片，探索动态表达。', materialLabel: '参考图片',
    prompt: '保持人物与工作室构图，探索不同光影。', mobilePrompt: '',
    images: [{ src: poster, alt: '工作室人物参考', label: '' }, { src: lastFrame, alt: '镜头光影参考', label: '' }], reference: lastFrame,
  },
]
</script>

<template>
  <section id="modes" class="s2v-home-section" aria-labelledby="home-modes-title">
    <h2 id="home-modes-title">从一个镜头开始。</h2>
    <TabsRoot v-model="selectedMode" activation-mode="automatic">
      <TabsList class="s2v-home-mode-tabs" aria-label="选择创作方式">
        <TabsTrigger v-for="mode in modes" :key="mode.id" :value="mode.id" class="s2v-home-mode-tab" :aria-label="mode.name">{{ mode.label }}</TabsTrigger>
      </TabsList>
      <TabsContent v-for="mode in modes" :key="mode.id" :value="mode.id" class="s2v-home-mode-panel">
        <p class="s2v-home-mode-description">{{ mode.description }}</p>
        <div class="s2v-home-workbench">
          <p class="s2v-home-demo-heading">创作界面示意</p>
          <div class="s2v-home-demo-body">
            <div class="s2v-home-demo-inputs">
              <div class="s2v-home-demo-materials">
                <p class="s2v-home-demo-label">{{ mode.materialLabel }}</p>
                <div v-if="mode.images.length" class="s2v-home-demo-assets">
                  <div v-for="image in mode.images" :key="image.src">
                    <ProductReferenceImage :src="image.src" :alt="image.alt" />
                    <p v-if="image.label" class="s2v-home-demo-asset-name">{{ image.label }}</p>
                  </div>
                </div>
                <p class="s2v-home-demo-prompt">{{ mode.prompt }}</p>
                <p v-if="mode.mobilePrompt" class="s2v-home-demo-mobile-prompt">{{ mode.mobilePrompt }}</p>
              </div>
              <p class="s2v-home-demo-settings">5 秒 · 16:9 · 768P</p>
            </div>
            <div class="s2v-home-demo-preview">
              <ProductReferenceImage :src="mode.reference" alt="品牌影像参考画面，不是生成结果" />
              <span class="s2v-home-reference-label">参考画面</span>
            </div>
            <div class="s2v-home-mode-action">
              <RouterLink :to="{ path: '/app/create', query: { mode: mode.id } }" class="s2v-home-mode-entry" :aria-label="`尝试${mode.name}`">去创作<ArrowUpRight :size="20" aria-hidden="true" /></RouterLink>
            </div>
          </div>
        </div>
      </TabsContent>
    </TabsRoot>
  </section>
</template>
