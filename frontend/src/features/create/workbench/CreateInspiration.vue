<script setup lang="ts">
const cards = [
  {
    title: '海岸建立镜头',
    note: '可带入提示词',
    badge: '文生 · 参考',
    src: '/brand/auth/iceland-coast.jpg',
    alt: '海岸构图参考画面，不是生成结果',
    prompt: '黄昏海岸，人物侧脸回望，镜头缓慢推进，风吹动发丝。',
    mode: 'text' as const,
  },
  {
    title: '人物转场',
    note: '需自备首尾帧',
    badge: '首尾帧 · 参考',
    src: '/brand/auth/creator-studio-poster.jpg',
    alt: '工作室人物构图参考画面，不是生成结果',
    prompt: '从冷色夜景，过渡到温暖的镜头光影。',
    mode: 'first_last' as const,
  },
  {
    title: '产品光影',
    note: '至少一张参考图',
    badge: '多图 · 参考',
    src: '/brand/auth/creator-studio-last.jpg',
    alt: '镜头光影参考画面，不是生成结果',
    prompt: '保持人物与工作室构图，探索不同光影。',
    mode: 'multi_image' as const,
  },
  {
    title: '情绪收束',
    note: '示例卡片',
    badge: '参考',
    src: '/brand/auth/iceland-coast.jpg',
    alt: '情绪收束构图参考画面，不是生成结果',
    prompt: '镜头缓缓拉远，余晖落在水面，情绪安静收束。',
    mode: 'text' as const,
  },
]

const emit = defineEmits<{
  apply: [payload: { mode: 'text' | 'first_last' | 'multi_image'; prompt: string }]
}>()
</script>

<template>
  <section class="s2v-studio-inspiration" aria-labelledby="create-inspiration-title">
    <div class="s2v-studio-inspiration-head">
      <h2 id="create-inspiration-title">灵感参考</h2>
      <span>构图参考 · 非生成作品</span>
    </div>
    <div class="s2v-studio-inspiration-grid">
      <button
        v-for="card in cards"
        :key="card.title"
        type="button"
        class="s2v-studio-inspiration-card"
        :aria-label="`将「${card.title}」带入创作`"
        @click="emit('apply', { mode: card.mode, prompt: card.prompt })"
      >
        <span class="s2v-studio-inspiration-thumb">
          <img :src="card.src" :alt="card.alt" loading="lazy">
          <span class="s2v-studio-inspiration-badge">{{ card.badge }}</span>
        </span>
        <span class="s2v-studio-inspiration-body">
          <b>{{ card.title }}</b>
          <span>{{ card.note }}</span>
        </span>
      </button>
    </div>
  </section>
</template>
