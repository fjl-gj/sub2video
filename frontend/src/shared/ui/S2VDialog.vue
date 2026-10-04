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
  hideActions?: boolean
  size?: 'default' | 'wide'
}>(), { confirmLabel: '确认', cancelLabel: '取消', tone: 'primary', hideActions: false, size: 'default' })

const emit = defineEmits<{ confirm: [] }>()

function confirm() {
  emit('confirm')
  open.value = false
}
</script>

<template>
  <DialogRoot v-model:open="open">
    <DialogPortal>
      <DialogOverlay class="s2v-modal-overlay" />
      <DialogContent :class="['s2v-modal-content', size === 'wide' && 's2v-modal-wide']">
        <DialogTitle class="text-sm font-bold text-s2v-ink">{{ props.title }}</DialogTitle>
        <DialogDescription v-if="description" class="mt-2 text-xs leading-5 text-s2v-muted">{{ description }}</DialogDescription>
        <div v-if="$slots.default" class="mt-5">
          <slot />
        </div>
        <div v-if="!hideActions" class="mt-5 flex justify-end gap-2">
          <DialogClose as-child>
            <S2VButton>{{ cancelLabel }}</S2VButton>
          </DialogClose>
          <S2VButton :variant="tone === 'danger' ? 'danger' : 'primary'" @click="confirm">{{ confirmLabel }}</S2VButton>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
