export type FsItemType = 'image' | 'video' | 'audio'
export type AssetType = '图片' | '视频' | '音频'

export interface FsItem {
  id: string
  name: string
  type: FsItemType
  size: string
  time: string
  meta?: string
  sourceTask?: string
  url?: string
  remoteUrl?: string
}

export interface FsFolder {
  isSystem: boolean
  desc: string
  folders: Record<string, FsFolder>
  items: FsItem[]
}

export interface WorkProject {
  id: string
  name: string
  desc: string
  createdAt: string
  trashed: boolean
  trashedAt?: string
  isDefault?: boolean
  fileCount?: number
  folders: Record<string, FsFolder>
}

export interface Asset {
  id: string
  name: string
  type: AssetType
  size: string
  folder: string
  updated: string
  sourceTask?: string
  url?: string
}

export const SYSTEM_ROOTS = ['资产', '视频'] as const
export const ASSET_BUCKETS = ['人物', '场景', '道具', '素材'] as const
export const DEFAULT_VIDEO_ALBUM = '正片镜头'

function emptyFolder(desc: string, isSystem = true): FsFolder {
  return { isSystem, desc, folders: {}, items: [] }
}

export function createStandardFolders(): Record<string, FsFolder> {
  return {
    资产: {
      isSystem: true,
      desc: '核心资产固定目录',
      folders: {
        人物: emptyFolder('角色与人设图'),
        场景: emptyFolder('环境与场景参考'),
        道具: emptyFolder('装备与物品特写'),
        素材: emptyFolder('纹理、贴图与音效'),
      },
      items: [],
    },
    视频: {
      isSystem: true,
      desc: '视频成片输出主目录',
      folders: {
        [DEFAULT_VIDEO_ALBUM]: emptyFolder('默认镜头分集', false),
      },
      items: [],
    },
  }
}

export function cloneFolders(folders: Record<string, FsFolder>): Record<string, FsFolder> {
  return Object.fromEntries(
    Object.entries(folders).map(([name, folder]) => [
      name,
      {
        isSystem: folder.isSystem,
        desc: folder.desc,
        folders: cloneFolders(folder.folders),
        items: folder.items.map((item) => ({ ...item })),
      },
    ]),
  )
}

export function createWorkProject(id: string, name: string, desc: string, createdAt: string): WorkProject {
  return {
    id,
    name,
    desc,
    createdAt,
    trashed: false,
    folders: cloneFolders(createStandardFolders()),
  }
}

export function resolveNode(project: WorkProject, path: string[]): { folders: Record<string, FsFolder>; items: FsItem[]; folder?: FsFolder } {
  if (!path.length) return { folders: project.folders, items: [] }
  let folder: FsFolder | undefined
  let folders = project.folders
  for (const segment of path) {
    folder = folders[segment]
    if (!folder) return { folders: {}, items: [] }
    folders = folder.folders
  }
  if (!folder) return { folders: {}, items: [] }
  return { folders: folder.folders, items: folder.items, folder }
}

export function countFolderFiles(folder: FsFolder): number {
  return folder.items.length + Object.values(folder.folders).reduce((sum, child) => sum + countFolderFiles(child), 0)
}

export function countProjectFiles(project: WorkProject): number {
  if (typeof project.fileCount === 'number') return project.fileCount
  return Object.values(project.folders).reduce((sum, folder) => sum + countFolderFiles(folder), 0)
}

function toAssetType(type: FsItemType): AssetType {
  if (type === 'video') return '视频'
  if (type === 'audio') return '音频'
  return '图片'
}

export function flattenProject(project: WorkProject): Asset[] {
  const assets: Asset[] = []
  function walk(folders: Record<string, FsFolder>) {
    for (const [name, folder] of Object.entries(folders)) {
      for (const item of folder.items) {
        assets.push({
          id: item.id,
          name: item.name,
          type: toAssetType(item.type),
          size: item.size,
          folder: name,
          updated: item.time,
          sourceTask: item.sourceTask,
          url: item.url || item.remoteUrl,
        })
      }
      walk(folder.folders)
    }
  }
  walk(project.folders)
  return assets
}

export function findFileInProject(project: WorkProject, id: string): { folder: FsFolder; index: number; item: FsItem } | null {
  function walk(folders: Record<string, FsFolder>): { folder: FsFolder; index: number; item: FsItem } | null {
    for (const folder of Object.values(folders)) {
      const index = folder.items.findIndex((item) => item.id === id)
      if (index >= 0) {
        const item = folder.items[index]
        if (item) return { folder, index, item }
      }
      const nested = walk(folder.folders)
      if (nested) return nested
    }
    return null
  }
  return walk(project.folders)
}
