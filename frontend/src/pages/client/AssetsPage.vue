<script setup lang="ts">
import { ArrowLeft, Folder, Search, Upload } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import AssetCard from '@/features/assets/AssetCard.vue'
import FolderCard from '@/features/assets/FolderCard.vue'
import ProjectCard from '@/features/assets/ProjectCard.vue'
import { countFolderFiles, countProjectFiles, DEFAULT_PROJECT_ID, resolveNode, type FsFolder, type FsItem } from '@/data/projects'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VDialog from '@/shared/ui/S2VDialog.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VProgress from '@/shared/ui/S2VProgress.vue'
import S2VSegmented from '@/shared/ui/S2VSegmented.vue'

const app = useAppStore()
const search = ref('')
const viewMode = ref('网格')
const creating = ref(false)
const newName = ref('')
const inside = ref(false)
const path = ref<string[]>([])
const trashOpen = ref(false)
const pendingTrashId = ref('')

const project = computed(() => app.currentProject)
const node = computed(() => project.value ? resolveNode(project.value, path.value) : { folders: {} as Record<string, FsFolder>, items: [] as FsItem[] })

const folders = computed(() => {
  const entries = Object.entries(node.value.folders)
  const keyword = search.value.trim()
  return entries.filter(([name]) => !keyword || name.includes(keyword))
})
const files = computed(() => {
  const keyword = search.value.trim()
  return node.value.items.filter((item) => !keyword || item.name.includes(keyword))
})

function enterProject(id: string) {
  app.selectProject(id)
  inside.value = true
  path.value = []
  search.value = ''
}

function exitProject() {
  inside.value = false
  path.value = []
  search.value = ''
  creating.value = false
}

function openFolder(name: string) {
  path.value = [...path.value, name]
  search.value = ''
}

function jumpTo(index: number) {
  path.value = path.value.slice(0, index)
  search.value = ''
}

function confirmCreate() {
  app.createProject(newName.value)
  newName.value = ''
  creating.value = false
  inside.value = true
  path.value = []
}

function askTrash(id: string) {
  pendingTrashId.value = id
  trashOpen.value = true
}

function confirmTrash() {
  if (pendingTrashId.value) app.trashProject(pendingTrashId.value)
  pendingTrashId.value = ''
  inside.value = false
  path.value = []
}

function uploadHere() {
  if (!project.value) return
  app.uploadAsset(project.value.id, path.value.length ? path.value : ['资产', '素材'])
}
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader v-if="!inside" title="资产">
      <template #description>每个作品都是一套标准工程：资产（人物 / 场景 / 道具 / 素材）与视频。删除作品只会进入回收站。</template>
      <template #actions>
        <S2VButton v-if="!creating" variant="primary" @click="creating = true">新建作品</S2VButton>
      </template>
    </S2VPageHeader>
    <S2VPageHeader v-else :title="project?.name || '作品'">
      <template #actions>
        <S2VButton @click="exitProject">
          <template #icon><ArrowLeft class="h-4 w-4" /></template>
          返回作品列表
        </S2VButton>
        <S2VButton variant="primary" @click="uploadHere">
          <template #icon><Upload class="h-4 w-4" /></template>
          上传素材
        </S2VButton>
      </template>
    </S2VPageHeader>

    <S2VCard variant="padded" class="mb-5">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div class="flex items-center gap-2">
            <Folder class="h-4 w-4 text-s2v-primary" />
            <strong class="text-xs">存储空间</strong>
            <S2VBadge tone="primary">{{ app.storageUsed }} / {{ app.storageQuota }} GB</S2VBadge>
          </div>
          <S2VProgress class="mt-3" :value="app.storagePercent" />
        </div>
        <span class="text-[11px] text-s2v-muted">{{ app.fileCount }} 个已保存文件 · {{ app.activeProjects.length }} 个作品</span>
      </div>
    </S2VCard>

    <template v-if="!inside">
      <div v-if="creating" class="mb-4 flex flex-wrap items-end gap-2">
        <S2VInput v-model="newName" class="max-w-xs flex-1" size="sm" placeholder="作品名称，例如 春日短片" />
        <S2VButton variant="primary" size="sm" @click="confirmCreate">按模板创建</S2VButton>
        <S2VButton size="sm" @click="creating = false">取消</S2VButton>
      </div>

      <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <ProjectCard
          v-for="item in app.activeProjects"
          :key="item.id"
          :project="item"
          :file-count="countProjectFiles(item)"
          :is-default="item.id === DEFAULT_PROJECT_ID"
          :is-current="item.id === app.currentProject?.id"
          @enter="enterProject(item.id)"
          @trash="askTrash(item.id)"
        />
      </div>
      <S2VCard v-if="!app.activeProjects.length" variant="padded" class="mt-4 grid place-items-center text-center">
        <Folder class="h-6 w-6 text-s2v-muted" />
        <p class="mt-2 text-xs text-s2v-muted">还没有作品，按标准模板新建一个，或从回收站恢复。</p>
      </S2VCard>
    </template>

    <template v-else>
      <div class="mb-4 flex flex-wrap items-center gap-2">
        <nav class="flex min-w-0 flex-1 flex-wrap items-center gap-1.5 text-xs">
          <button class="font-semibold text-s2v-primary hover:underline" type="button" @click="path = []">{{ project?.name }}</button>
          <template v-for="(segment, index) in path" :key="`${segment}-${index}`">
            <span class="text-s2v-label">/</span>
            <button class="font-semibold text-s2v-primary hover:underline" type="button" @click="jumpTo(index + 1)">{{ segment }}</button>
          </template>
        </nav>
        <S2VSegmented v-model="viewMode" :options="['网格', '列表']" />
        <S2VInput v-model="search" class="max-w-xs flex-1" size="sm" placeholder="在当前目录检索">
          <template #prefix><Search class="h-3.5 w-3.5 text-s2v-muted" /></template>
        </S2VInput>
      </div>

      <div v-if="viewMode === '网格'" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        <FolderCard
          v-for="[name, folder] in folders"
          :key="name"
          :name="name"
          :folder="folder"
          :count="countFolderFiles(folder)"
          @open="openFolder(name)"
        />
        <AssetCard v-for="item in files" :key="item.id" :item="item" @dismiss="app.removeAsset(item.id)" />
      </div>

      <div v-else class="s2v-table-wrap">
        <table class="s2v-table">
          <thead>
            <tr>
              <th>名称</th>
              <th>类型</th>
              <th>说明</th>
              <th>大小 / 时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="[name, folder] in folders" :key="name" class="cursor-pointer" @click="openFolder(name)">
              <td>
                <strong class="text-xs">{{ name }}</strong>
              </td>
              <td>文件夹</td>
              <td>{{ folder.desc }}</td>
              <td>{{ countFolderFiles(folder) }} 项</td>
            </tr>
            <tr v-for="item in files" :key="item.id">
              <td>
                <strong class="text-xs">{{ item.name }}</strong>
              </td>
              <td>{{ item.type === 'video' ? '视频' : item.type === 'audio' ? '音频' : '图片' }}</td>
              <td>{{ item.meta || '—' }}</td>
              <td>{{ item.size }} · {{ item.time }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <S2VCard v-if="!folders.length && !files.length" variant="padded" class="mt-4 grid place-items-center text-center">
        <Folder class="h-6 w-6 text-s2v-muted" />
        <p class="mt-2 text-xs text-s2v-muted">这个目录还是空的</p>
      </S2VCard>
    </template>

    <S2VDialog
      v-model:open="trashOpen"
      tone="danger"
      title="移入回收站"
      description="作品不会被立即删除，之后可以在回收站恢复。"
      confirm-label="移入回收站"
      @confirm="confirmTrash"
    />
  </section>
</template>
