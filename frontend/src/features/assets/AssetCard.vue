<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { FileImage, FileMusic, FileVideo } from 'lucide-vue-next'
import type { FsItem } from '@/data/projects'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'

const props = defineProps<{ item: FsItem; mediaUrl: string }>()
defineEmits<{ dismiss: []; open: [] }>()
const mediaFailed = ref(false)
const icon = computed(() => props.item.type === 'video' ? FileVideo : props.item.type === 'audio' ? FileMusic : FileImage)
const label = computed(() => props.item.type === 'video' ? '视频' : props.item.type === 'audio' ? '音频' : '图片')
watch(() => props.mediaUrl, () => { mediaFailed.value = false })
</script>

<template>
  <article class="s2v-library-file">
    <button type="button" class="s2v-library-file-media" :aria-label="`预览 ${item.name}`" @click="$emit('open')">
      <img v-if="item.type === 'image' && mediaUrl && !mediaFailed" :src="mediaUrl" :alt="item.name" loading="lazy" @error="mediaFailed = true" />
      <component :is="icon" v-else class="h-8 w-8" />
    </button>
    <div class="s2v-library-file-body">
      <div class="s2v-library-file-title"><strong>{{ item.name }}</strong><S2VBadge>{{ label }}</S2VBadge></div>
      <p class="s2v-library-file-meta">{{ item.size }}{{ item.meta ? ` · ${item.meta}` : '' }}</p>
      <div class="s2v-library-file-actions">
        <S2VButton size="sm" @click="$emit('open')">查看文件</S2VButton>
        <S2VButton size="sm" variant="ghost" aria-label="移除文件" @click="$emit('dismiss')">删除</S2VButton>
      </div>
      <RouterLink v-if="item.sourceTask" :to="`/app/works/${encodeURIComponent(item.sourceTask)}`" class="s2v-library-text-action">查看来源任务</RouterLink>
    </div>
  </article>
</template>
