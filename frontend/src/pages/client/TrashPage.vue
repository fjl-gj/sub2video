<script setup lang="ts">
import { RotateCcw, Trash2 } from 'lucide-vue-next'
import { computed, onMounted, ref } from 'vue'
import { countProjectFiles } from '@/data/projects'
import { useAppStore } from '@/stores/app'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VDialog from '@/shared/ui/S2VDialog.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const app = useAppStore()
const purgeOpen = ref(false)
const pendingId = ref('')
const loading = ref(false)
const pendingName = computed(() => app.trashedProjects.find((item) => item.id === pendingId.value)?.name || '')

function askPurge(id: string) {
  pendingId.value = id
  purgeOpen.value = true
}

async function confirmPurge() {
  if (pendingId.value) await app.purgeProject(pendingId.value)
  pendingId.value = ''
}

onMounted(async () => {
  loading.value = true
  try {
    await app.loadProjects({ includeTrash: true })
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="回收站">
      <template #description>从资产里移出的作品会先到这里。可以恢复，也可以彻底删除。</template>
    </S2VPageHeader>

    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <S2VCard v-for="item in app.trashedProjects" :key="item.id" variant="padded">
        <div class="flex items-start gap-3">
          <span class="s2v-icon-well s2v-icon-well-nav"><Trash2 class="h-4 w-4" /></span>
          <div class="min-w-0 flex-1">
            <strong class="block truncate text-sm text-s2v-ink">{{ item.name }}</strong>
            <p class="mt-1 text-xs text-s2v-muted">{{ item.desc }}</p>
            <p class="mt-2 text-[11px] text-s2v-label">移入 {{ item.trashedAt || '—' }} · {{ countProjectFiles(item) }} 个文件</p>
          </div>
        </div>
        <div class="mt-4 flex flex-wrap gap-2">
          <S2VButton variant="primary" size="sm" @click="app.restoreProject(item.id)">
            <template #icon><RotateCcw class="h-3.5 w-3.5" /></template>
            恢复
          </S2VButton>
          <S2VButton variant="danger" size="sm" @click="askPurge(item.id)">彻底删除</S2VButton>
        </div>
      </S2VCard>
    </div>

    <S2VCard v-if="!loading && !app.trashedProjects.length" variant="padded" class="grid place-items-center text-center">
      <Trash2 class="h-6 w-6 text-s2v-muted" />
      <p class="mt-2 text-xs text-s2v-muted">回收站是空的</p>
    </S2VCard>

    <S2VDialog
      v-model:open="purgeOpen"
      tone="danger"
      title="彻底删除作品"
      :description="pendingName ? `「${pendingName}」删除后无法恢复。` : ''"
      confirm-label="彻底删除"
      @confirm="confirmPurge"
    />
  </section>
</template>
