<script setup lang="ts">
import { DialogClose, DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import S2VButton from './S2VButton.vue'

const open = defineModel<boolean>('open', { default: false })
const props = withDefaults(defineProps<{
  title: string
  description?: string
  confirmLabel?: string
  cancelLabel?: string
  tone?: 'primary' | 'danger'
}>(), { confirmLabel: '确认', cancelLabel: '取消', tone: 'primary' })

const emit = defineEmits<{ confirm: [] }>()

function confirm() {
  emit('confirm')
  open.value = false
}
</script>

<template>
  <DialogRoot v-model:open="open">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-50 bg-slate-900/40 backdrop-blur-[2px]" />
      <DialogContent class="fixed left-1/2 top-1/2 z-50 w-[min(420px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-panel border border-s2v-line/80 bg-s2v-surface p-5 shadow-subtle">
        <DialogTitle class="text-sm font-bold text-s2v-ink">{{ props.title }}</DialogTitle>
        <DialogDescription v-if="description" class="mt-2 text-xs leading-5 text-s2v-muted">{{ description }}</DialogDescription>
        <div class="mt-5 flex justify-end gap-2">
          <DialogClose as-child>
            <S2VButton>{{ cancelLabel }}</S2VButton>
          </DialogClose>
          <S2VButton :variant="tone === 'danger' ? 'danger' : 'primary'" @click="confirm">{{ confirmLabel }}</S2VButton>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
