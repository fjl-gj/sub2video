<script setup lang="ts">
import { FileImage, FileMusic, FileVideo, Folder, Plus, Search, Upload } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { assets } from '@/data/mock'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VSegmented from '@/shared/ui/S2VSegmented.vue'
import S2VProgress from '@/shared/ui/S2VProgress.vue'

const app = useAppStore()
const search = ref('')
const selectedFolder = ref('全部')
const folders = ['全部', '场景', '人物', '产品', '视频', '品牌', '音频']
const filtered = computed(() => assets.filter((item) => (selectedFolder.value === '全部' || item.folder === selectedFolder.value) && (!search.value || item.name.includes(search.value))))
const iconFor = (type: string) => type === '视频' ? FileVideo : type === '音频' ? FileMusic : FileImage
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader badge="Assets" title="资产库">
      <template #description>只保存你主动留下的素材，任务临时文件不会无限占用空间。</template>
      <template #actions>
        <S2VButton variant="primary" @click="app.notify('已模拟打开文件选择器')">
          <template #icon><Upload class="h-4 w-4" /></template>
          上传资产
        </S2VButton>
      </template>
    </S2VPageHeader>

    <S2VCard variant="padded" class="mb-5">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div class="flex items-center gap-2">
            <Folder class="h-4 w-4 text-s2v-primary" />
            <strong class="text-xs">存储空间</strong>
            <S2VBadge tone="primary">1.9 / 5 GB</S2VBadge>
          </div>
          <S2VProgress class="mt-3" :value="38" />
        </div>
        <span class="text-[11px] text-s2v-muted">任务输入 7 天自动清理 · 结果 30 天自动清理</span>
      </div>
    </S2VCard>

    <div class="mb-4 flex flex-wrap items-center gap-2">
      <S2VSegmented v-model="selectedFolder" :options="folders" />
      <S2VInput v-model="search" class="ml-auto max-w-xs flex-1" size="sm" placeholder="搜索资产">
        <template #prefix><Search class="h-3.5 w-3.5 text-s2v-muted" /></template>
      </S2VInput>
    </div>

    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      <S2VCard v-for="item in filtered" :key="item.id" variant="media">
        <div class="grid aspect-[4/3] place-items-center" :style="{ background: `linear-gradient(135deg, #${item.color}, #${item.accent})` }">
          <component :is="iconFor(item.type)" class="h-8 w-8 text-white/80" />
        </div>
        <div class="s2v-panel-body-sm">
          <div class="flex items-center justify-between gap-2">
            <strong class="truncate text-xs">{{ item.name }}</strong>
            <S2VBadge>{{ item.type }}</S2VBadge>
          </div>
          <div class="mt-1.5 flex items-center justify-between text-[11px] text-s2v-muted">
            <span>{{ item.size }} · {{ item.folder }}</span>
            <button aria-label="资产更多操作" class="rounded-nav p-1 text-s2v-muted hover:bg-s2v-canvas" type="button"><Plus class="h-3.5 w-3.5 rotate-45" /></button>
          </div>
        </div>
      </S2VCard>
    </div>

    <S2VCard v-if="!filtered.length" variant="padded" class="mt-4 grid place-items-center text-center">
      <Folder class="h-6 w-6 text-s2v-muted" />
      <p class="mt-2 text-xs text-s2v-muted">暂无匹配资产</p>
    </S2VCard>
  </section>
</template>
