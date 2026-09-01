<script setup lang="ts">
import { DialogClose, DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { computed } from 'vue'
import { useAppStore } from '@/stores/app'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VChoice from '@/shared/ui/S2VChoice.vue'

const open = defineModel<boolean>('open', { default: false })

const emit = defineEmits<{ pick: [assetId: string] }>()

const app = useAppStore()
const imageAssets = computed(() => app.assets.filter((item) => item.type === '图片'))

function select(assetId: string) {
  emit('pick', assetId)
}
</script>

<template>
  <DialogRoot v-model:open="open">
    <DialogPortal>
      <DialogOverlay class="s2v-dialog-overlay" />
      <DialogContent class="s2v-dialog-panel">
        <DialogTitle class="text-sm font-bold text-s2v-ink">从作品资产库选取</DialogTitle>
        <DialogDescription class="mt-1 text-xs text-s2v-muted">支持命中角色、场景、道具与素材图片</DialogDescription>
        <div class="mt-4 grid max-h-64 gap-2 overflow-y-auto">
          <S2VChoice
            v-for="item in imageAssets"
            :key="item.id"
            class="text-left"
            @click="select(item.id)"
          >
            {{ item.name }} · {{ item.folder }}
          </S2VChoice>
          <p v-if="!imageAssets.length" class="text-xs text-s2v-muted">当前作品里还没有图片</p>
        </div>
        <div class="mt-4 flex justify-end">
          <DialogClose as-child>
            <S2VButton>取消</S2VButton>
          </DialogClose>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
