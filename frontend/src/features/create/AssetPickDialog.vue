<script setup lang="ts">
import { DialogClose, DialogContent, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { Upload } from 'lucide-vue-next'
import { computed, useTemplateRef } from 'vue'
import { useAppStore } from '@/stores/app'
import { assetPreviewUrl } from '@/shared/lib/asset-preview'

const open = defineModel<boolean>('open', { default: false })

const emit = defineEmits<{
  pick: [assetId: string]
  upload: [payload: { file: File; fileName: string; previewUrl: string }]
}>()

const app = useAppStore()
const fileInput = useTemplateRef<HTMLInputElement>('fileInput')
const imageAssets = computed(() => app.assets.filter((item) => item.type === '图片'))

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
  <DialogRoot v-model:open="open">
    <DialogPortal>
      <DialogOverlay class="s2v-dialog-overlay" />
      <DialogContent class="s2v-dialog-panel">
        <DialogTitle class="text-sm font-bold text-s2v-ink">添加或选取素材</DialogTitle>

        <button type="button" class="s2v-wb-asset-upload mt-4" @click="triggerUpload">
          <Upload class="h-4 w-4" />
          <span>从本地电脑上传图片</span>
        </button>
        <input ref="fileInput" type="file" accept="image/*" class="hidden" @change="onFileChange">

        <div class="s2v-wb-asset-divider">
          <span class="line" />
          <span>或从已有资产库选取</span>
          <span class="line" />
        </div>

        <div class="grid max-h-52 gap-2 overflow-y-auto">
          <button
            v-for="item in imageAssets"
            :key="item.id"
            type="button"
            class="s2v-wb-asset-item"
            @click="select(item.id)"
          >
            <img class="s2v-wb-asset-thumb" :src="assetPreviewUrl(item.id, item.url)" :alt="item.name">
            <span>{{ item.name }}</span>
          </button>
          <p v-if="!imageAssets.length" class="text-xs text-s2v-muted">当前作品里还没有图片</p>
        </div>

        <div class="mt-4 flex justify-end">
          <DialogClose as-child>
            <button type="button" class="s2v-wb-stage-btn">取消</button>
          </DialogClose>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
