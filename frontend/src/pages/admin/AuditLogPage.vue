<script setup lang="ts">
import { FileClock, ShieldCheck } from 'lucide-vue-next'
import { auditLogs } from '@/data/mock'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader badge="Audit" title="操作日志">
      <template #description>记录管理员对任务、定价、工作流和用户做过的关键操作。</template>
      <template #actions>
        <S2VBadge tone="primary"><ShieldCheck class="h-3 w-3" />审计留痕</S2VBadge>
      </template>
    </S2VPageHeader>

    <S2VCard>
      <div class="s2v-panel-header">
        <div>
          <h2 class="s2v-section-title">最近操作</h2>
          <p class="mt-1 text-xs text-s2v-muted">只读记录，不在此处修改业务数据。</p>
        </div>
        <FileClock class="h-4 w-4 text-s2v-muted" />
      </div>
      <div class="overflow-x-auto">
        <table class="s2v-table">
          <thead>
            <tr>
              <th>时间</th>
              <th>操作人</th>
              <th>动作</th>
              <th>目标</th>
              <th>结果</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in auditLogs" :key="item.time">
              <td class="font-mono">{{ item.time }}</td>
              <td>{{ item.operator }}</td>
              <td>{{ item.action }}</td>
              <td class="font-mono">{{ item.target }}</td>
              <td><S2VBadge tone="success">{{ item.result }}</S2VBadge></td>
            </tr>
          </tbody>
        </table>
      </div>
    </S2VCard>
  </section>
</template>
