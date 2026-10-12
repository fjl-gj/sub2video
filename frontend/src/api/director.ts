import { http } from '@/api/http'
import type { ClientTaskDetail } from '@/api/client'

export type DirectorContext = { projectId: string; shotId: string; expectedRevision: number }
export type DirectorShot = {
  id: string
  title: string
  prompt: string
  modeId: string
  carrierId: string
  modelId: string
  durationSec: number
  resolution: string
  aspectRatio: string
  generateAudio: boolean
  sourceMeta: NonNullable<ClientTaskDetail['sourceMeta']>
  candidateTaskIds: string[]
  selectedTaskId: string
}
export type DirectorBoard = { projectId: string; revision: number; shots: DirectorShot[]; updatedAt: string }
export type DirectorShotInput = Pick<DirectorShot, 'prompt' | 'modeId' | 'carrierId' | 'modelId' | 'durationSec' | 'resolution' | 'aspectRatio' | 'generateAudio' | 'sourceMeta'>
export function directorShotDocument(shot: DirectorShot): DirectorShot {
  const ref = (value: { type: 'upload' | 'asset'; id: string }) => ({ type: value.type, id: value.id })
  const source = shot.sourceMeta || {}
  return {
    id: shot.id, title: shot.title, prompt: shot.prompt, modeId: shot.modeId,
    carrierId: shot.carrierId, modelId: shot.modelId, durationSec: shot.durationSec,
    resolution: shot.resolution, aspectRatio: shot.aspectRatio, generateAudio: shot.generateAudio,
    sourceMeta: {
      ...(source.first ? { first: ref(source.first) } : {}), ...(source.last ? { last: ref(source.last) } : {}),
      ...(source.refs ? { refs: source.refs.map(ref) } : {}), ...(source.extras ? { extras: source.extras.map(ref) } : {}),
      ...(source.audio ? { audio: ref(source.audio) } : {}),
    },
    candidateTaskIds: [...shot.candidateTaskIds], selectedTaskId: shot.selectedTaskId,
  }
}
const boardPath = (projectId: string) => `/client/projects/${encodeURIComponent(projectId)}/director-board`

export async function fetchDirectorBoard(projectId: string) {
  const { data } = await http.get<DirectorBoard>(boardPath(projectId))
  return data
}
export async function saveDirectorBoard(projectId: string, expectedRevision: number, shots: DirectorShot[]) {
  const { data } = await http.put<DirectorBoard>(boardPath(projectId), { expectedRevision, shots: shots.map(directorShotDocument) })
  return data
}
export async function attachDirectorCandidate(projectId: string, shotId: string, taskId: string) {
  const { data } = await http.post<DirectorBoard>(`${boardPath(projectId)}/shots/${encodeURIComponent(shotId)}/candidates`, { taskId })
  return data
}

export function directorErrorMessage(error: unknown, fallback: string) {
  const response = (error as { response?: { status?: number; data?: { code?: string } } }).response
  const code = response?.data?.code
  if (code === 'DIRECTOR_BOARD_CONFLICT' || code === 'DIRECTOR_SHOT_MISSING') return '镜头板已在其他页面更新。你的编辑仍保留，可先导出清单，再重新加载后核对。'
  if (code === 'DIRECTOR_SELECTION_NOT_READY') return '这一版尚未完成交付，请刷新候选状态后再选用。'
  if (code === 'DIRECTOR_REFERENCE_INVALID') return '参考素材已不可用或不属于当前账号，请在创作页重新选择。'
  if (code === 'DIRECTOR_TASK_INVALID') return '这个任务无法关联，请选择当前账号可访问的任务。'
  if (code === 'DIRECTOR_BOARD_INVALID') return '镜头内容不符合保存要求，请检查名称、描述和候选数量。'
  if (response?.status === 404) return '作品或镜头不存在，可能已移入回收站，或你没有访问权限。'
  return fallback
}
