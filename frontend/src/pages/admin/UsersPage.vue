<script setup lang="ts">
import { MoreHorizontal, Search, UserRoundCheck } from 'lucide-vue-next'
import { ref } from 'vue'
import { users } from '@/data/mock'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VIconButton from '@/shared/ui/S2VIconButton.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const app = useAppStore()
const search = ref('')
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader badge="Users" title="用户管理">
      <template #description>查看账户状态、余额、任务和订单，处理需要人工介入的账户。</template>
      <template #actions>
        <S2VBadge tone="success"><UserRoundCheck class="h-3 w-3" />{{ users.filter((u) => u.status === '正常').length }} 个正常账户</S2VBadge>
      </template>
    </S2VPageHeader>

    <div class="mb-4 max-w-xs">
      <S2VInput v-model="search" size="sm" placeholder="昵称 / 邮箱 / UID">
        <template #prefix><Search class="h-3.5 w-3.5 text-s2v-muted" /></template>
      </S2VInput>
    </div>

    <div class="s2v-table-wrap">
      <table class="s2v-table">
        <thead>
          <tr>
            <th>用户</th>
            <th>邮箱</th>
            <th>状态</th>
            <th>余额</th>
            <th>任务数</th>
            <th>注册时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in users.filter((row) => !search || `${row.id}${row.name}${row.email}`.includes(search))" :key="user.id">
            <td>
              <div class="flex items-center gap-2">
                <span class="grid h-8 w-8 place-items-center rounded-full bg-slate-700 text-xs font-bold text-white">{{ user.name.slice(0, 1) }}</span>
                <div>
                  <strong class="block text-xs text-s2v-ink">{{ user.name }}</strong>
                  <span class="mt-1 block font-mono text-[10px] text-s2v-muted">{{ user.id }}</span>
                </div>
              </div>
            </td>
            <td>{{ user.email }}</td>
            <td><S2VBadge :tone="user.status === '正常' ? 'success' : user.status === '已冻结' ? 'danger' : 'warning'">{{ user.status }}</S2VBadge></td>
            <td class="font-mono">¥{{ user.balance.toFixed(2) }}</td>
            <td>{{ user.tasks }}</td>
            <td>{{ user.joined }}</td>
            <td><S2VIconButton label="查看用户详情" @click="app.notify(`已打开 ${user.name} 详情`)"><MoreHorizontal class="h-4 w-4" /></S2VIconButton></td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
