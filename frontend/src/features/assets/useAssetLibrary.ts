import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { createProject, deleteProjectFile, setCurrentProject, uploadProjectFile } from '@/api/client'
import { countProjectFiles, findFileInProject, resolveNode, type FsFolder, type FsItem, type WorkProject } from '@/data/projects'
import { assetPreviewUrl } from '@/shared/lib/asset-preview'
import { PAGE_SIZE, slicePage } from '@/shared/lib/paginate'
import { useAppStore } from '@/stores/app'

function queryString(value: unknown) { return typeof value === 'string' ? value : '' }

export function useAssetLibrary() {
  const route = useRoute()
  const router = useRouter()
  const app = useAppStore()
  const search = ref('')
  const viewMode = ref('网格')
  const page = ref(1)
  const loading = ref(false)
  const loadError = ref('')
  const project = ref<WorkProject | null>(null)
  const creating = ref(false)
  const createBusy = ref(false)
  const createError = ref('')
  const newName = ref('')
  const trashOpen = ref(false)
  const trashBusy = ref(false)
  const trashError = ref('')
  const pendingTrashId = ref('')
  const removing = ref<FsItem | null>(null)
  const removeBusy = ref(false)
  const removeError = ref('')
  const preview = ref<FsItem | null>(null)
  const mediaFailed = ref(false)
  const uploading = ref(false)
  const createNavigationBusy = ref(false)
  const operationError = ref('')
  let requestVersion = 0
  let disposed = false

  const projectId = computed(() => queryString(route.query.project))
  const inside = computed(() => !!projectId.value)
  const path = computed(() => queryString(route.query.folder).split('/').filter(Boolean))
  const node = computed(() => project.value ? resolveNode(project.value, path.value) : { folders: {} as Record<string, FsFolder>, items: [] as FsItem[] })
  const invalidFolder = computed(() => !!project.value && path.value.length > 0 && !node.value.folder)
  const keyword = computed(() => search.value.trim().toLocaleLowerCase())
  const projects = computed(() => app.activeProjects.filter((item) => !keyword.value || item.name.toLocaleLowerCase().includes(keyword.value)))
  const pagedProjects = computed(() => slicePage(projects.value, page.value))
  const folders = computed(() => Object.entries(node.value.folders).filter(([name]) => !keyword.value || name.toLocaleLowerCase().includes(keyword.value)))
  const files = computed(() => node.value.items.filter((item) => !keyword.value || item.name.toLocaleLowerCase().includes(keyword.value)))
  const entries = computed(() => [
    ...folders.value.map(([name, folder]) => ({ kind: 'folder' as const, id: name, name, folder })),
    ...files.value.map((item) => ({ kind: 'file' as const, id: item.id, item })),
  ])
  const pagedEntries = computed(() => slicePage(entries.value, page.value))
  const previewOpen = computed({ get: () => !!preview.value, set: (value: boolean) => { if (!value) preview.value = null } })
  const removeOpen = computed({ get: () => !!removing.value, set: (value: boolean) => { if (!value) removing.value = null } })

  function fileUrl(item: FsItem) {
    if (!projectId.value || !item.id) return ''
    return assetPreviewUrl(item.id, `/api/v1/client/projects/${encodeURIComponent(projectId.value)}/files/${encodeURIComponent(item.id)}`)
  }

  async function load() {
    const request = ++requestVersion
    const id = projectId.value
    loading.value = true
    loadError.value = ''
    try {
      if (id) {
        const result = await app.loadProjectTree(id)
        if (disposed || request !== requestVersion) return
        if (result.trashed) {
          project.value = null
          loadError.value = '这个作品已在回收站中，请恢复后再打开。'
        } else project.value = result
      } else {
        await app.loadProjects({ includeTrash: false })
        if (disposed || request !== requestVersion) return
        project.value = null
      }
    } catch (error) {
      if (disposed || request !== requestVersion) return
      const status = (error as { response?: { status?: number } }).response?.status
      project.value = null
      loadError.value = status === 404 ? '未找到这个作品，或你没有查看权限。' : '资产暂时无法读取，请重新加载。'
    } finally {
      if (!disposed && request === requestVersion) loading.value = false
    }
  }

  function navigateProject(id: string, nextPath: string[] = []) {
    return router.push({ path: '/app/assets', query: { project: id, ...(nextPath.length ? { folder: nextPath.join('/') } : {}) } })
  }
  function exitProject() { return router.push('/app/assets') }
  function openFolder(name: string) { return navigateProject(projectId.value, [...path.value, name]) }
  function jumpTo(index: number) { return navigateProject(projectId.value, path.value.slice(0, index)) }

  async function confirmCreate() {
    if (createBusy.value) return
    const name = newName.value.trim()
    if (!name) { createError.value = '请填写作品名称。'; return }
    createBusy.value = true
    createError.value = ''
    try {
      const result = await createProject(name)
      if (disposed) return
      creating.value = false
      newName.value = ''
      app.notify(`已创建作品「${result.project.name}」`)
      await navigateProject(result.project.id)
      void app.refreshStorage()
    } catch {
      if (!disposed) createError.value = '创建作品失败，请稍后重试。'
    } finally { createBusy.value = false }
  }

  function askTrash(id: string) {
    pendingTrashId.value = id
    trashError.value = ''
    trashOpen.value = true
  }

  async function confirmTrash() {
    if (trashBusy.value || !pendingTrashId.value) return
    trashBusy.value = true
    const id = pendingTrashId.value
    const ok = await app.trashProject(id)
    trashBusy.value = false
    if (!ok) { trashError.value = '移入回收站失败，请稍后重试。'; return }
    trashOpen.value = false
    pendingTrashId.value = ''
    if (projectId.value === id) await exitProject()
  }

  async function upload(file: File) {
    if (!project.value || uploading.value) return
    const id = project.value.id
    const targetPath = path.value.length ? [...path.value] : ['资产', '素材']
    uploading.value = true
    operationError.value = ''
    try {
      await uploadProjectFile(id, targetPath, file)
      app.notify('素材已上传')
      if (projectId.value === id) await load()
      void app.refreshStorage()
    } catch { operationError.value = '素材上传失败，请检查文件和网络后重试。' }
    finally { uploading.value = false }
  }

  function askRemove(item: FsItem) { removing.value = item; removeError.value = '' }
  async function confirmRemove() {
    if (!removing.value || removeBusy.value || !project.value) return
    const id = project.value.id
    const fileId = removing.value.id
    removeBusy.value = true
    removeError.value = ''
    try {
      await deleteProjectFile(id, fileId)
      removing.value = null
      if (preview.value?.id === fileId) preview.value = null
      app.notify('已删除文件')
      if (projectId.value === id) await load()
      void app.refreshStorage()
    } catch { removeError.value = '文件删除失败，请稍后重试。' }
    finally { removeBusy.value = false }
  }

  function openPreview(item: FsItem) { preview.value = item; mediaFailed.value = false }
  async function reloadPreview() {
    const id = preview.value?.id
    await load()
    if (!id || !project.value) return
    const result = findFileInProject(project.value, id)
    if (result) { preview.value = { ...result.item }; mediaFailed.value = false }
    else { preview.value = null; operationError.value = '文件已不可用，请刷新目录后重试。' }
  }

  async function continueCreation() {
    if (!project.value || createNavigationBusy.value) return
    const id = project.value.id
    createNavigationBusy.value = true
    operationError.value = ''
    try {
      await setCurrentProject(id)
      if (disposed || projectId.value !== id) return
      app.currentProjectId = id
      await router.push('/app/create')
    } catch { operationError.value = '暂时无法切换当前作品，请稍后重试。' }
    finally { createNavigationBusy.value = false }
  }

  watch(projectId, () => {
    project.value = null
    preview.value = null
    removing.value = null
    trashOpen.value = false
    search.value = ''
    operationError.value = ''
    void load()
  }, { immediate: true })
  watch([search, projectId, path], () => { page.value = 1 })
  watch(path, () => { search.value = '' })
  watch(creating, (value) => { if (value) createError.value = '' })
  onBeforeUnmount(() => { disposed = true; requestVersion++ })

  return {
    app, search, viewMode, page, pageSize: PAGE_SIZE, loading, loadError, operationError, project, projectId, inside,
    path, node, invalidFolder, projects, pagedProjects, folders, files, entries, pagedEntries, creating, createBusy,
    createError, newName, trashOpen, trashBusy, trashError, removing, removeOpen, removeBusy, removeError,
    preview, previewOpen, mediaFailed, uploading, createNavigationBusy, countProjectFiles,
    load, navigateProject, exitProject, openFolder, jumpTo, confirmCreate, askTrash, confirmTrash, upload,
    askRemove, confirmRemove, openPreview, reloadPreview, fileUrl, continueCreation,
  }
}
