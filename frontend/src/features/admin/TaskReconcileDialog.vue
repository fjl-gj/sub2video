<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { reconcileAdminTask, type AdminTaskReconcileAction } from '@/api/admin'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VDialog from '@/shared/ui/S2VDialog.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VTextarea from '@/shared/ui/S2VTextarea.vue'

const props = defineProps<{
  task: { id: string; title: string; providerRequestId?: string }
  action: AdminTaskReconcileAction
}>()
const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ resolved: [taskId: string] }>()
const upstreamTaskId = ref('')
const evidence = ref('')
const confirmed = ref(false)
const submitting = ref(false)
const error = ref('')
const isResume = computed(() => props.action === 'resume')
const evidenceLength = computed(() => Array.from(evidence.value.trim()).length)
const canSubmit = computed(() => confirmed.value && evidenceLength.value >= 10 && evidenceLength.value <= 1000
  && (!isResume.value || !!upstreamTaskId.value.trim()) && !submitting.value)

watch(() => [open.value, props.task.id, props.action] as const, ([value]) => {
  if (!value) return
  upstreamTaskId.value = props.task.providerRequestId || ''
  evidence.value = ''
  confirmed.value = false
  error.value = ''
}, { immediate: true })

async function submit() {
  if (!canSubmit.value) return
  submitting.value = true
  error.value = ''
  try {
    await reconcileAdminTask(props.task.id, {
      action: props.action,
      ...(isResume.value ? { upstreamTaskId: upstreamTaskId.value.trim() } : {}),
      evidence: evidence.value.trim(),
      confirmed: true,
    })
    open.value = false
    emit('resolved', props.task.id)
  } catch (cause) {
    const failure = cause as { response?: { data?: { message?: string } } }
    error.value = failure.response?.data?.message || '核对结果未能保存，请重新确认任务状态后重试。'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <S2VDialog
    v-model:open="open"
    :title="isResume ? '关联已受理任务并继续查询' : '确认未受理并退款'"
    :description="`${task.title} · ${task.id}`"
    hide-actions
  >
    <form @submit.prevent="submit">
      <fieldset class="grid gap-4" :disabled="submitting">
        <p class="s2v-note">
          {{ isResume
            ? '请先在供应商侧核实已受理任务。关联任务号后继续查询原任务，不会重新提交或再次扣费。'
            : '仅在供应商已明确确认未受理、未创建任务后使用。该操作将结束平台任务并退回原扣款，请勿仅因暂未查到结果而退款。' }}
        </p>
        <S2VInput v-if="isResume" v-model="upstreamTaskId" label="已核实的上游任务号" placeholder="填写供应商侧已创建的任务号" :disabled="Boolean(task.providerRequestId)" required />
        <p v-if="isResume && task.providerRequestId" class="s2v-note">平台已记录该上游任务号，只能核实并继续此任务。</p>
        <S2VTextarea v-model="evidence" label="核对记录（10–1000 字）" placeholder="记录核对时间、核对方式与确认结果；不要填写 Token 或其他凭据。" :rows="4" />
        <label class="flex items-start gap-2 text-xs">
          <input v-model="confirmed" type="checkbox" class="mt-0.5 h-4 w-4" />
          <span>{{ isResume ? '我已核实该上游任务属于当前平台任务' : '我已获得供应商确认：当前请求未受理且未创建任务' }}</span>
        </label>
        <p v-if="error" role="alert" class="s2v-note">{{ error }}</p>
        <div class="flex flex-wrap justify-end gap-2">
          <S2VButton type="button" @click="open = false">取消</S2VButton>
          <S2VButton type="submit" :variant="isResume ? 'primary' : 'danger'" :disabled="!canSubmit">
            {{ submitting ? '正在保存核对结果…' : isResume ? '确认关联并继续查询' : '确认未受理并退款' }}
          </S2VButton>
        </div>
      </fieldset>
    </form>
  </S2VDialog>
</template>
