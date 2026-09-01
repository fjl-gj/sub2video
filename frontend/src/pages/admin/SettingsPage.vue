<script setup lang="ts">
import { Bell, Database, Save, ShieldCheck, SlidersHorizontal } from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const app = useAppStore()
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="系统设置">
      <template #description>管理存储留存、通知和系统级开关，不包含供应商业务配置。</template>
      <template #actions>
        <S2VButton variant="primary" @click="app.saveSettings()"><Save class="h-3.5 w-3.5" />保存设置</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="grid gap-5 lg:grid-cols-2">
      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">存储留存</h2>
            <p class="mt-1 text-xs text-s2v-muted">用于控制临时素材和生成结果的生命周期。</p>
          </div>
          <Database class="h-4 w-4 text-s2v-primary" />
        </div>
        <div class="s2v-panel-body grid gap-4">
          <S2VInput v-model="app.settings.inputRetention" label="任务输入保留（天）" type="number" min="1" />
          <S2VInput v-model="app.settings.resultRetention" label="生成结果保留（天）" type="number" min="1" />
          <div class="s2v-note">用户主动保存的资产不受任务留存规则影响，但仍受个人资产额度和平台策略约束。</div>
        </div>
      </S2VCard>

      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">平台开关</h2>
            <p class="mt-1 text-xs text-s2v-muted">控制通知和审计能力。</p>
          </div>
          <SlidersHorizontal class="h-4 w-4 text-s2v-primary" />
        </div>
        <div class="divide-y divide-s2v-line/70 px-4">
          <label class="flex items-center justify-between gap-3 py-4 text-xs">
            <span class="flex items-center gap-2">
              <Bell class="h-4 w-4 text-s2v-muted" />
              <span>
                <strong class="block">任务通知</strong>
                <span class="mt-1 block text-[11px] text-s2v-muted">任务完成或失败时提醒用户</span>
              </span>
            </span>
            <input v-model="app.settings.notify" type="checkbox" class="h-4 w-4 accent-s2v-primary" />
          </label>
          <label class="flex items-center justify-between gap-3 py-4 text-xs">
            <span class="flex items-center gap-2">
              <ShieldCheck class="h-4 w-4 text-s2v-muted" />
              <span>
                <strong class="block">管理员审计</strong>
                <span class="mt-1 block text-[11px] text-s2v-muted">记录关键操作和配置变更</span>
              </span>
            </span>
            <input v-model="app.settings.audit" type="checkbox" class="h-4 w-4 accent-s2v-primary" />
          </label>
        </div>
      </S2VCard>
    </div>
  </section>
</template>
