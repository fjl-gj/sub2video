<script setup lang="ts">
import { RouterLink } from 'vue-router'
import type { GuideShot, ShotGuide } from './catalog'
import S2VButton from '@/shared/ui/S2VButton.vue'
import './guides.css'

defineProps<{ guide?: ShotGuide; shot?: GuideShot; invalid: boolean; unavailable: boolean; applied: boolean; adjusted: boolean; loading: boolean }>()
defineEmits<{ apply: [] }>()
</script>

<template>
  <aside class="guide-create-context" aria-label="镜头教程">
    <div>
      <template v-if="invalid"><h2>未找到这个镜头教程</h2><p>请返回广场选择教程，或退出教程继续自由创作。</p></template>
      <template v-else-if="guide && shot">
        <span class="s2v-guide-eyebrow">镜头教程 / {{ guide.modeName }}</span>
        <h2>{{ guide.title }} · {{ shot.title }}</h2>
        <p>{{ shot.intent }}</p>
        <p v-if="unavailable" role="status">教程模式暂不可用</p>
        <p v-else-if="!applied">已保留你的草稿。应用教程设置会替换画面描述与生成设置，已选素材会保留。</p>
        <p v-else-if="adjusted" role="status">教程推荐参数已按当前可用能力调整，请核对后提交。</p>
        <p v-else>提示词与设置供你修改；实际能力和费用以当前服务为准。</p>
      </template>
    </div>
    <div class="guide-create-actions">
      <S2VButton v-if="guide && shot && !invalid" :disabled="unavailable || loading" @click="$emit('apply')">应用教程设置</S2VButton>
      <RouterLink v-if="guide" :to="`/guides/${guide.id}`">查看教程</RouterLink>
      <RouterLink to="/app/create">退出教程</RouterLink>
    </div>
  </aside>
</template>
