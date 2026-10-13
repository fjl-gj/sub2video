<script setup lang="ts">
import { FileText, Images, SquareStack } from 'lucide-vue-next'

const modes = [
  {
    id: 'text',
    name: '文生视频',
    hint: '只需提示词',
    icon: FileText,
  },
  {
    id: 'first_last',
    name: '首尾帧',
    hint: '提示词 + 首帧 + 尾帧',
    icon: SquareStack,
  },
  {
    id: 'multi_image',
    name: '多图生成',
    hint: '提示词 + 至少一张参考图',
    icon: Images,
  },
] as const
</script>

<template>
  <section id="modes" class="s2v-home-modes" aria-label="三种创作方式">
    <h2 class="s2v-home-modes-title">三种创作方式</h2>
    <div class="s2v-home-mode-rows">
      <RouterLink
        v-for="mode in modes"
        :key="mode.id"
        :to="{ path: '/app/create', query: { mode: mode.id } }"
        class="s2v-home-mode-row"
        :aria-label="`用${mode.name}开始创作`"
      >
        <span class="s2v-home-mode-icon" aria-hidden="true">
          <component :is="mode.icon" :size="22" />
        </span>
        <b>{{ mode.name }}</b>
        <span>{{ mode.hint }}</span>
      </RouterLink>
    </div>
  </section>
</template>
