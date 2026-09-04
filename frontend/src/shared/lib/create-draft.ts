import type { CreateResolution } from '@/shared/lib/create-cost'

export type CreateMode = 'image' | 'text'
export type ImageVariant = 'keyframes' | 'multi'

export const MAX_REF_IMAGES = 9
export const MAX_KEYFRAME_EXTRAS = 6

export interface CreateFrameSlot {
  uploaded: boolean
  assetId: string
  picking: boolean
  previewUrl?: string
  fileName?: string
  /** Local file retained until uploaded; never persisted to draft storage. */
  file?: File
  uploadId?: string
}

export interface CreateDraft {
  version: 2
  mode: CreateMode
  imageVariant: ImageVariant
  prompt: string
  ratio: string
  durationSec: string
  resolution: CreateResolution
  firstFrame: CreateFrameSlot
  lastFrame: CreateFrameSlot
  extraFrames: CreateFrameSlot[]
  multiFrames: CreateFrameSlot[]
}

const STORAGE_KEY = 'sub2video.create-draft'

export function emptySlot(): CreateFrameSlot {
  return { uploaded: false, assetId: '', picking: false }
}

export function emptyCreateDraft(): CreateDraft {
  return {
    version: 2,
    mode: 'image',
    imageVariant: 'keyframes',
    prompt: '',
    ratio: '9:16',
    durationSec: '15',
    resolution: '768P',
    firstFrame: emptySlot(),
    lastFrame: emptySlot(),
    extraFrames: [],
    multiFrames: [emptySlot()],
  }
}

function sanitizeSlot(slot: CreateFrameSlot): CreateFrameSlot {
  const previewUrl = typeof slot.previewUrl === 'string' && !slot.previewUrl.startsWith('blob:')
    ? slot.previewUrl
    : undefined
  return {
    uploaded: slot.uploaded,
    assetId: slot.assetId,
    picking: false,
    previewUrl,
    fileName: typeof slot.fileName === 'string' ? slot.fileName : undefined,
    uploadId: typeof slot.uploadId === 'string' ? slot.uploadId : undefined,
  }
}

function isCreateMode(value: unknown): value is CreateMode {
  return value === 'image' || value === 'text'
}

function isImageVariant(value: unknown): value is ImageVariant {
  return value === 'keyframes' || value === 'multi'
}

function parseResolution(value: unknown): CreateResolution {
  if (value === '720P' || value === '768P' || value === '1080P') return value
  return '768P'
}

function parseSlot(value: unknown): CreateFrameSlot {
  if (!value || typeof value !== 'object') return emptySlot()
  const slot = value as Partial<CreateFrameSlot>
  return {
    uploaded: Boolean(slot.uploaded),
    assetId: typeof slot.assetId === 'string' ? slot.assetId : '',
    picking: false,
    previewUrl: typeof slot.previewUrl === 'string' && !slot.previewUrl.startsWith('blob:')
      ? slot.previewUrl
      : undefined,
    fileName: typeof slot.fileName === 'string' ? slot.fileName : undefined,
    uploadId: typeof slot.uploadId === 'string' ? slot.uploadId : undefined,
  }
}

function parseSlots(value: unknown, max: number, fallback: CreateFrameSlot[] = []): CreateFrameSlot[] {
  if (!Array.isArray(value)) return fallback.map((slot) => sanitizeSlot(slot))
  return value.slice(0, max).map(parseSlot)
}

type LegacyCreateDraft = Partial<Omit<CreateDraft, 'version'>> & {
  version?: 1 | 2
  mode?: CreateMode | 'video'
  referenceVideoUploaded?: boolean
}

function normalizeMode(value: unknown): CreateMode | null {
  if (value === 'video') return 'image'
  return isCreateMode(value) ? value : null
}

function migrateDraft(parsed: LegacyCreateDraft): CreateDraft | null {
  const mode = normalizeMode(parsed.mode)
  if (!mode || !isImageVariant(parsed.imageVariant)) return null

  const legacyMulti = Array.isArray(parsed.multiFrames) ? parsed.multiFrames : []

  return {
    version: 2,
    mode,
    imageVariant: parsed.imageVariant,
    prompt: typeof parsed.prompt === 'string' ? parsed.prompt : '',
    ratio: typeof parsed.ratio === 'string' ? parsed.ratio : '9:16',
    durationSec: typeof parsed.durationSec === 'string' ? parsed.durationSec : '15',
    resolution: parseResolution(parsed.resolution),
    firstFrame: parseSlot(parsed.firstFrame),
    lastFrame: parseSlot(parsed.lastFrame),
    extraFrames: parseSlots(parsed.extraFrames, MAX_REF_IMAGES),
    multiFrames: parsed.version === 2
      ? parseSlots(parsed.multiFrames, MAX_REF_IMAGES, [emptySlot()])
      : parseSlots(legacyMulti, MAX_REF_IMAGES, [emptySlot()]),
  }
}

export function loadCreateDraft(): CreateDraft | null {
  if (typeof window === 'undefined') return null
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as LegacyCreateDraft
    if (parsed.version !== 1 && parsed.version !== 2) return null
    const draft = migrateDraft(parsed)
    if (!draft) return null
    if (draft.imageVariant === 'multi' && !draft.multiFrames.length) draft.multiFrames = [emptySlot()]
    return draft
  }
  catch {
    return null
  }
}

export function saveCreateDraft(draft: Omit<CreateDraft, 'version'>) {
  if (typeof window === 'undefined') return
  const payload: CreateDraft = {
    version: 2,
    mode: draft.mode,
    imageVariant: draft.imageVariant,
    prompt: draft.prompt,
    ratio: draft.ratio,
    durationSec: draft.durationSec,
    resolution: draft.resolution,
    firstFrame: sanitizeSlot(draft.firstFrame),
    lastFrame: sanitizeSlot(draft.lastFrame),
    extraFrames: draft.extraFrames.map(sanitizeSlot),
    multiFrames: draft.multiFrames.map(sanitizeSlot),
  }
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(payload))
}

export function clearCreateDraft() {
  if (typeof window === 'undefined') return
  window.localStorage.removeItem(STORAGE_KEY)
}
