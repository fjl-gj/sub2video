<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { AlertCircle, ArrowLeft, ArrowUpRight, Download, Folder, FolderOpen, LoaderCircle, Plus, RefreshCw, Search, Upload } from 'lucide-vue-next'
import { countFolderFiles } from '@/data/projects'
import AssetCard from './AssetCard.vue'
import FolderCard from './FolderCard.vue'
import ProjectCard from './ProjectCard.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VDialog from '@/shared/ui/S2VDialog.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VPagination from '@/shared/ui/S2VPagination.vue'
import S2VSegmented from '@/shared/ui/S2VSegmented.vue'
import { useAssetLibrary } from './useAssetLibrary'
import './library.css'

const {
  app, search, viewMode, page, pageSize, loading, loadError, operationError, project, inside, path, invalidFolder,
  projects, pagedProjects, entries, pagedEntries, creating, createBusy, createError, newName, trashOpen,
  trashBusy, trashError, removing, removeOpen, removeBusy, removeError, preview, previewOpen, mediaFailed,
  uploading, createNavigationBusy, countProjectFiles, load, navigateProject, exitProject, openFolder, jumpTo,
  confirmCreate, askTrash, confirmTrash, upload, askRemove, confirmRemove, openPreview, reloadPreview,
  fileUrl, continueCreation,
} = useAssetLibrary()
const fileInput = ref<HTMLInputElement | null>(null)
const router = useRouter()

function onFilePicked(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (file) void upload(file)
}
</script>

<template>
  <section class="s2v-page s2v-library">
    <S2VPageHeader v-if="!inside" title="资产" kicker="我的作品">
      <template #description>把参考素材、生成镜头与选好的版本，收在同一个作品里。</template>
      <template #actions><S2VButton variant="primary" size="lg" @click="creating = true"><Plus class="h-4 w-4" />新建作品</S2VButton></template>
    </S2VPageHeader>
    <S2VPageHeader v-else :title="project?.name || '作品'">
      <template #description>素材留在资产目录，满意的镜头保存到视频目录。</template>
      <template #actions>
        <S2VButton @click="exitProject"><ArrowLeft class="h-4 w-4" />返回作品列表</S2VButton>
        <S2VButton size="lg" :disabled="!project || loading" @click="router.push(`/app/projects/${encodeURIComponent(project!.id)}/director`)">导演台</S2VButton>
        <S2VButton :disabled="!project || loading" :loading="uploading" @click="fileInput?.click()"><Upload class="h-4 w-4" />上传素材</S2VButton>
        <S2VButton variant="primary" size="lg" :disabled="!project || loading" :loading="createNavigationBusy" @click="continueCreation">继续创作 <ArrowUpRight class="h-4 w-4" /></S2VButton>
      </template>
    </S2VPageHeader>
    <input ref="fileInput" type="file" class="hidden" accept="image/*,video/mp4,audio/*" aria-label="上传作品素材" @change="onFilePicked" />

    <div v-if="operationError" class="s2v-library-notice" role="alert"><AlertCircle class="h-4 w-4" /><p>{{ operationError }}</p></div>
    <div v-if="loading" class="s2v-library-empty" role="status"><LoaderCircle class="h-7 w-7 animate-spin" /><h2>正在读取{{ inside ? '作品内容' : '作品列表' }}</h2></div>
    <div v-else-if="loadError" class="s2v-library-empty" role="alert">
      <AlertCircle class="h-7 w-7" /><h2>{{ loadError }}</h2><S2VButton @click="load"><RefreshCw class="h-4 w-4" />重新加载</S2VButton>
    </div>
    <template v-else-if="!inside">
      <div class="s2v-library-toolbar">
        <div><h2>全部作品</h2><span>{{ app.activeProjects.length }} 个作品</span></div>
        <S2VInput v-model="search" label="查找作品" placeholder="搜索作品名称"><template #prefix><Search class="h-4 w-4" /></template></S2VInput>
      </div>
      <div v-if="projects.length" class="s2v-library-projects">
        <ProjectCard v-for="item in pagedProjects" :key="item.id" :project="item" :file-count="countProjectFiles(item)" :is-default="!!item.isDefault" :is-current="item.id === app.currentProjectId" @enter="navigateProject(item.id)" @trash="askTrash(item.id)" />
      </div>
      <div v-else class="s2v-library-empty">
        <FolderOpen class="h-8 w-8" /><h2>{{ search ? '没有匹配的作品' : '给下一个想法留个位置' }}</h2>
        <p>{{ search ? '换一个名称试试。' : '先建一个作品，也可以生成完成后再保存。' }}</p>
        <S2VButton v-if="!search" size="lg" variant="primary" @click="creating = true">新建作品</S2VButton>
        <S2VButton v-else @click="search = ''">清空搜索</S2VButton>
      </div>
      <S2VPagination v-if="projects.length > pageSize" v-model="page" :total="projects.length" :page-size="pageSize" class="mt-6" />
      <footer class="s2v-library-storage"><span>{{ app.storageUsed }} / {{ app.storageQuota }} GB 已使用 · {{ app.fileCount }} 个文件</span><span>作品移入回收站后可以恢复。</span></footer>
    </template>
    <template v-else-if="project">
      <nav class="s2v-library-breadcrumb" aria-label="当前作品路径">
        <button type="button" @click="jumpTo(0)">{{ project.name }}</button>
        <template v-for="(segment, index) in path" :key="`${segment}-${index}`"><span>/</span><button type="button" :aria-current="index === path.length - 1 ? 'page' : undefined" @click="jumpTo(index + 1)">{{ segment }}</button></template>
      </nav>
      <div class="s2v-library-interior">
        <nav class="s2v-library-folders" aria-label="作品目录">
          <button type="button" :class="{ selected: !path.length }" @click="jumpTo(0)"><FolderOpen class="h-4 w-4" />全部目录</button>
          <template v-for="(folder, name) in project.folders" :key="name">
            <button type="button" :class="{ selected: path[0] === name && path.length === 1 }" @click="navigateProject(project.id, [name])"><Folder class="h-4 w-4" />{{ name }}</button>
            <button v-for="(_, childName) in folder.folders" :key="childName" type="button" class="s2v-library-subfolder" :class="{ selected: path[0] === name && path[1] === childName }" @click="navigateProject(project.id, [name, childName])">{{ childName }}</button>
          </template>
        </nav>
        <div class="s2v-library-files">
          <div class="s2v-library-toolbar">
            <div><h2>{{ path.at(-1) || '作品目录' }}</h2><span>{{ entries.length }} 项</span></div>
            <S2VSegmented v-model="viewMode" :options="['网格', '列表']" />
          </div>
          <S2VInput v-model="search" class="mb-5" label="查找文件" placeholder="在当前目录检索"><template #prefix><Search class="h-4 w-4" /></template></S2VInput>
          <div v-if="invalidFolder" class="s2v-library-empty"><FolderOpen class="h-7 w-7" /><h2>这个目录不存在</h2><S2VButton @click="jumpTo(0)">返回作品目录</S2VButton></div>
          <div v-else-if="!entries.length" class="s2v-library-empty"><FolderOpen class="h-7 w-7" /><h2>{{ search ? '没有匹配的文件' : '这个目录还是空的' }}</h2><p>上传参考素材，或在任务详情中保存满意的成片。</p></div>
          <div v-else-if="viewMode === '网格'" class="s2v-library-file-grid">
            <template v-for="entry in pagedEntries" :key="entry.id">
              <FolderCard v-if="entry.kind === 'folder'" :name="entry.name" :folder="entry.folder" :count="countFolderFiles(entry.folder)" @open="openFolder(entry.name)" />
              <AssetCard v-else :item="entry.item" :media-url="fileUrl(entry.item)" @open="openPreview(entry.item)" @dismiss="askRemove(entry.item)" />
            </template>
          </div>
          <div v-else class="s2v-table-wrap">
            <table class="s2v-table">
              <thead><tr><th>名称</th><th>类型</th><th>大小 / 时间</th><th>操作</th></tr></thead>
              <tbody>
                <tr v-for="entry in pagedEntries" :key="entry.id">
                  <template v-if="entry.kind === 'folder'">
                    <td><button class="s2v-library-text-action" type="button" @click="openFolder(entry.name)">{{ entry.name }}</button></td><td>文件夹</td><td>{{ countFolderFiles(entry.folder) }} 项</td><td><S2VButton size="sm" @click="openFolder(entry.name)">打开</S2VButton></td>
                  </template>
                  <template v-else>
                    <td><button class="s2v-library-text-action" type="button" @click="openPreview(entry.item)">{{ entry.item.name }}</button></td><td>{{ entry.item.type === 'video' ? '视频' : entry.item.type === 'audio' ? '音频' : '图片' }}</td><td>{{ entry.item.size }} · {{ entry.item.time }}</td>
                    <td><div class="flex gap-2"><S2VButton size="sm" @click="openPreview(entry.item)">查看</S2VButton><S2VButton size="sm" variant="ghost" @click="askRemove(entry.item)">删除文件</S2VButton></div></td>
                  </template>
                </tr>
              </tbody>
            </table>
          </div>
          <S2VPagination v-if="entries.length > pageSize" v-model="page" :total="entries.length" :page-size="pageSize" class="mt-6" />
          <p class="s2v-library-footnote">每个已保存成片都能回到它的任务详情。作品用于整理素材与视频，方便下一次创作。</p>
        </div>
      </div>
    </template>

    <S2VDialog v-model:open="creating" title="新建作品" description="建立一个空间，集中保存参考素材与生成成片。" hide-actions>
      <form class="s2v-library-dialog" @submit.prevent="confirmCreate">
        <S2VInput v-model="newName" label="作品名称" placeholder="例如：海岸来信" :disabled="createBusy" />
        <p>包含资产与视频目录，创建后可直接上传素材。</p><p v-if="createError" class="s2v-library-dialog-error" role="alert">{{ createError }}</p>
        <div class="s2v-library-dialog-footer"><S2VButton :disabled="createBusy" @click="creating = false">取消</S2VButton><S2VButton type="submit" variant="primary" :loading="createBusy">创建作品</S2VButton></div>
      </form>
    </S2VDialog>
    <S2VDialog v-model:open="trashOpen" title="移入回收站" description="作品不会被立即删除，之后可以在回收站恢复。" hide-actions>
      <div class="s2v-library-dialog"><p v-if="trashError" class="s2v-library-dialog-error" role="alert">{{ trashError }}</p><div class="s2v-library-dialog-footer"><S2VButton :disabled="trashBusy" @click="trashOpen = false">取消</S2VButton><S2VButton variant="danger" :loading="trashBusy" @click="confirmTrash">移入回收站</S2VButton></div></div>
    </S2VDialog>
    <S2VDialog v-model:open="removeOpen" title="删除文件" :description="`删除「${removing?.name || ''}」后不能从回收站恢复，请确认不再需要。`" hide-actions>
      <div class="s2v-library-dialog"><p v-if="removeError" class="s2v-library-dialog-error" role="alert">{{ removeError }}</p><div class="s2v-library-dialog-footer"><S2VButton :disabled="removeBusy" @click="removeOpen = false">取消</S2VButton><S2VButton variant="danger" :loading="removeBusy" @click="confirmRemove">删除文件</S2VButton></div></div>
    </S2VDialog>
    <S2VDialog v-model:open="previewOpen" :title="preview?.name || '文件预览'" description="查看作品中的素材或已保存成片。" hide-actions size="wide">
      <div v-if="preview" class="s2v-library-preview">
        <div class="s2v-library-preview-stage">
          <div v-if="mediaFailed" class="s2v-library-preview-error" role="status"><AlertCircle class="h-7 w-7" /><p>文件预览暂不可用，可以下载后查看。</p><S2VButton :loading="loading" @click="reloadPreview">重新加载预览</S2VButton></div>
          <img v-else-if="preview.type === 'image'" :src="fileUrl(preview)" :alt="preview.name" @error="mediaFailed = true" />
          <video v-else-if="preview.type === 'video'" :src="fileUrl(preview)" :aria-label="`${preview.name} 视频预览`" controls playsinline preload="metadata" @error="mediaFailed = true" />
          <audio v-else :src="fileUrl(preview)" :aria-label="`${preview.name} 音频预览`" controls preload="metadata" @error="mediaFailed = true" />
        </div>
        <div class="s2v-library-preview-meta"><span>{{ preview.size }} · {{ preview.time }}</span><span>{{ preview.meta }}</span></div>
        <div class="s2v-library-dialog-footer">
          <RouterLink v-if="preview.sourceTask" :to="`/app/works/${encodeURIComponent(preview.sourceTask)}`" class="s2v-library-text-action">查看来源任务 <ArrowUpRight class="h-4 w-4" /></RouterLink>
          <a :href="fileUrl(preview)" target="_blank" rel="noopener noreferrer" download class="s2v-library-text-action"><Download class="h-4 w-4" />下载文件</a>
          <S2VButton @click="previewOpen = false">关闭</S2VButton>
        </div>
      </div>
    </S2VDialog>
  </section>
</template>
