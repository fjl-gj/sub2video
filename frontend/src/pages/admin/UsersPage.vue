<script setup lang="ts">
import { computed, ref } from 'vue'
import { Search, ShieldBan, UserRoundCheck } from 'lucide-vue-next'
import MobileRecordCard from '@/features/lists/MobileRecordCard.vue'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VDialog from '@/shared/ui/S2VDialog.vue'
import S2VIconButton from '@/shared/ui/S2VIconButton.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const app = useAppStore()
const search = ref('')
const confirmOpen = ref(false)
const pending = ref<(typeof app.users)[number] | null>(null)
const list = computed(() => app.users.filter((row) => !search.value || `${row.id}${row.name}${row.email}`.includes(search.value)))
const activeCount = computed(() => app.users.filter((user) => user.status === '正常').length)

function askToggle(user: (typeof app.users)[number]) {
  pending.value = user
  confirmOpen.value = true
}

function confirmToggle() {
  if (pending.value) app.toggleUserStatus(pending.value.id)
}
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="用户管理">
      <template #description>查看账户状态、余额、任务和订单，处理需要人工介入的账户。</template>
      <template #actions>
        <S2VBadge tone="success"><UserRoundCheck class="h-3 w-3" />{{ activeCount }} 个正常账户</S2VBadge>
      </template>
    </S2VPageHeader>

    <div class="mb-4 max-w-xs">
      <S2VInput v-model="search" size="sm" placeholder="昵称 / 邮箱 / UID">
        <template #prefix><Search class="h-3.5 w-3.5 text-s2v-muted" /></template>
      </S2VInput>
    </div>

    <div class="s2v-table-wrap hidden md:block">
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
          <tr v-for="user in list" :key="user.id">
            <td>
              <div class="flex items-center gap-2">
                <span class="s2v-avatar">{{ user.name.slice(0, 1) }}</span>
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
            <td>
              <S2VIconButton :label="user.status === '已冻结' ? '解冻账号' : '冻结账号'" @click="askToggle(user)"><ShieldBan class="h-4 w-4" /></S2VIconButton>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="grid gap-3 md:hidden">
      <MobileRecordCard v-for="user in list" :key="user.id" :title="user.name" :meta="user.id">
        <template #badge>
          <S2VBadge :tone="user.status === '正常' ? 'success' : user.status === '已冻结' ? 'danger' : 'warning'">{{ user.status }}</S2VBadge>
        </template>
        <template #body>{{ user.email }} · ¥{{ user.balance.toFixed(2) }} · {{ user.tasks }} 个任务</template>
        <template #actions>
          <S2VIconButton :label="user.status === '已冻结' ? '解冻账号' : '冻结账号'" @click="askToggle(user)"><ShieldBan class="h-4 w-4" /></S2VIconButton>
        </template>
      </MobileRecordCard>
    </div>

    <S2VDialog
      v-model:open="confirmOpen"
      tone="danger"
      title="变更账户状态"
      :description="pending ? `确认将 ${pending.name} ${pending.status === '已冻结' ? '解冻' : '冻结'}？冻结后不能创建任务或使用余额。` : ''"
      :confirm-label="pending?.status === '已冻结' ? '解冻' : '冻结'"
      @confirm="confirmToggle"
    />
  </section>
</template>
