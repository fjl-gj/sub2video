<script setup lang="ts">
import { shallowRef } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowLeft, Clapperboard, Download, Plus, RefreshCw, Save } from 'lucide-vue-next'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import DirectorShotList from './DirectorShotList.vue'
import DirectorShotEditor from './DirectorShotEditor.vue'
import DirectorCandidates from './DirectorCandidates.vue'
import DirectorTaskPicker from './DirectorTaskPicker.vue'
import { useDirectorWorkspace } from './useDirectorWorkspace'
const router = useRouter()
const { projectId, projectName, board, activeShotId, activeShot, dirty, selectedCount, loading, saving, error, message, load, save, addShot, updateShot, moveShot, removeShot, createShot, selectCandidate, attachCandidate, exportBoard } = useDirectorWorkspace()
const pickerOpen = shallowRef(false)
async function attach(id: string) { if (await attachCandidate(id)) pickerOpen.value = false }
</script>

<template>
  <section class="s2v-page director-workspace">
    <S2VPageHeader :title="`${projectName} · 导演台`" kicker="镜头编排">
      <template #description>写清每一镜，逐镜创作，再选出要用的版本。</template>
      <template #actions><S2VButton size="lg" @click="router.push({ path: '/app/assets', query: { project: projectId } })"><ArrowLeft class="h-4 w-4" aria-hidden="true" />返回作品</S2VButton><S2VButton size="lg" :disabled="!board" @click="exportBoard"><Download class="h-4 w-4" aria-hidden="true" />导出镜头清单</S2VButton><S2VButton size="lg" variant="primary" :loading="saving" :disabled="!dirty || loading" @click="save"><Save class="h-4 w-4" aria-hidden="true" />保存镜头板</S2VButton></template>
    </S2VPageHeader>
    <div class="director-summary"><p>{{ board?.shots.length || 0 }} 个镜头 · {{ selectedCount }} 个已选片 <span v-if="dirty">· 有未保存修改</span></p><S2VButton size="lg" variant="ghost" :disabled="saving" :loading="loading" @click="load(true)"><RefreshCw class="h-4 w-4" aria-hidden="true" />重新加载镜头板</S2VButton></div>
    <p v-if="error" class="director-error" role="alert">{{ error }}</p>
    <p v-if="message" class="director-message" role="status">{{ message }}</p>
    <div v-if="loading && !board" class="director-empty" role="status"><p>正在读取作品与镜头…</p></div>
    <div v-else-if="board && !board.shots.length" class="director-empty"><Clapperboard class="h-10 w-10" aria-hidden="true" /><h2>从第一镜开始组织故事</h2><p>新增镜头，写下画面描述，再前往创作准备素材。这里保留镜头顺序与选片结果，不自动生成或扣费。</p><S2VButton size="lg" variant="primary" :disabled="saving" @click="addShot"><Plus class="h-4 w-4" aria-hidden="true" />新增镜头</S2VButton></div>
    <div v-else-if="board" class="director-layout">
      <DirectorShotList :shots="board.shots" :active-id="activeShotId" :disabled="saving || loading" @select="activeShotId = $event" @move="moveShot" @add="addShot" />
      <div v-if="activeShot" class="director-main"><DirectorShotEditor :shot="activeShot" :disabled="saving || loading" :dirty="dirty" @update="updateShot" @create="createShot" @remove="removeShot" /><DirectorCandidates :key="activeShot.id" :project-id="projectId" :shot="activeShot" :disabled="saving || loading" @select="selectCandidate" @attach="pickerOpen = true" /></div>
    </div>
    <footer v-if="board" class="director-footer">清单导出包含镜头顺序和选片记录，不包含视频文件，也不是合成影片。成片可从任务详情下载，继续使用你熟悉的剪辑工具。</footer>
    <DirectorTaskPicker v-if="activeShot" v-model:open="pickerOpen" :attached-ids="activeShot.candidateTaskIds" :busy="saving" @attach="attach" />
  </section>
</template>

<style>
.director-workspace { container-name: director; container-type: inline-size; }
.director-summary { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px; padding-bottom: 18px; margin-bottom: 24px; border-bottom: 1px solid rgb(var(--s2v-line)); }
.director-summary p { color: rgb(var(--s2v-muted)); font-size: 13px; line-height: 1.8; }
.director-summary p span { color: rgb(var(--s2v-warning)); }
.director-layout { display: grid; grid-template-columns: minmax(220px, 280px) minmax(0, 1fr); gap: 32px; align-items: start; }
.director-main { display: grid; grid-template-columns: minmax(280px, 380px) minmax(0, 1fr); gap: 32px; min-width: 0; }
.director-workspace .director-section-title { display: flex; flex-wrap: wrap; gap: 12px; align-items: center; justify-content: space-between; min-height: 44px; }
.director-workspace .director-section-title h2 { color: rgb(var(--s2v-ink)); font-size: 16px; font-weight: 500; }
.director-workspace .director-section-title span, .director-workspace .director-section-title p { color: rgb(var(--s2v-muted)); font-size: 12px; line-height: 1.8; }
.director-workspace .director-section-title p { margin-top: 8px; }
.director-workspace .director-hint { color: rgb(var(--s2v-muted)); font-size: 12px; line-height: 1.8; margin-top: 16px; }
.director-workspace .director-error, .director-workspace .director-message { color: rgb(var(--s2v-danger)); font-size: 13px; line-height: 1.8; margin-bottom: 20px; }
.director-workspace .director-message { color: rgb(var(--s2v-success)); }
.director-workspace .director-link { display: inline-flex; align-items: center; gap: 7px; min-height: 44px; color: rgb(var(--s2v-primary)); font-size: 13px; }
.director-workspace .director-link:hover { text-decoration: underline; }
.director-workspace .director-link:focus-visible { outline: 2px solid rgb(var(--s2v-primary)); outline-offset: 3px; }
.director-empty { display: grid; place-items: center; align-content: center; gap: 18px; min-height: 340px; padding: 32px 24px; border: 1px dashed rgb(var(--s2v-line-strong)); border-radius: var(--s2v-radius); color: rgb(var(--s2v-muted)); text-align: center; }
.director-empty h2 { color: rgb(var(--s2v-ink)); font-size: 20px; font-weight: 500; }
.director-empty p { max-width: 520px; font-size: 14px; line-height: 1.9; }
.director-footer { color: rgb(var(--s2v-muted)); font-size: 12px; line-height: 1.9; margin-top: 32px; padding-top: 22px; border-top: 1px solid rgb(var(--s2v-line)); }
@container director (max-width: 1100px) { .director-main { grid-template-columns: minmax(0, 1fr); gap: 32px; } }
@container director (max-width: 720px) { .director-layout { grid-template-columns: minmax(0, 1fr); gap: 28px; } }
</style>
