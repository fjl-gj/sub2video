<script setup lang="ts">
import { Upload } from 'lucide-vue-next'
import { computed, useTemplateRef } from 'vue'
import { useAppStore } from '@/stores/app'
import { assetPreviewUrl } from '@/shared/lib/asset-preview'
import S2VDialog from '@/shared/ui/S2VDialog.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import type { Asset } from '@/data/projects'

const props = defineProps<{ assets?: Asset[] }>()

const open = defineModel<boolean>('open', { default: false })

const emit = defineEmits<{
  pick: [assetId: string]
  upload: [payload: { file: File; fileName: string; previewUrl: string }]
}>()

const app = useAppStore()
const fileInput = useTemplateRef<HTMLInputElement>('fileInput')
const imageAssets = computed(() => (props.assets || app.assets).filter((item) => item.type === '图片'))

function select(assetId: string) {
  emit('pick', assetId)
}

function triggerUpload() {
  fileInput.value?.click()
}

function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file || !file.type.startsWith('image/')) return
  const previewUrl = URL.createObjectURL(file)
  emit('upload', { file, fileName: file.name, previewUrl })
  input.value = ''
}
</script>

<template>
  <S2VDialog v-model:open="open" title="添加或选取素材" description="上传本地图片，或使用当前作品中的图片素材。" hide-actions>
        <button type="button" class="s2v-studio-asset-upload" @click="triggerUpload">
          <Upload class="h-4 w-4" />
          <span>从本地电脑上传图片</span>
        </button>
        <input ref="fileInput" type="file" accept="image/*" class="hidden" aria-label="参考图片文件" @change="onFileChange">

        <p class="s2v-studio-asset-divider">或从已有资产库选取</p>

        <div class="grid max-h-52 gap-2 overflow-y-auto">
          <button
            v-for="item in imageAssets"
            :key="item.id"
            type="button"
            class="s2v-studio-asset-item"
            @click="select(item.id)"
          >
            <img :src="assetPreviewUrl(item.id, item.url)" :alt="item.name">
            <span>{{ item.name }}</span>
          </button>
          <p v-if="!imageAssets.length" class="text-xs text-s2v-muted">当前作品里还没有图片</p>
        </div>

        <div class="mt-4 flex justify-end">
          <S2VButton @click="open = false">取消</S2VButton>
        </div>
  </S2VDialog>
</template>
