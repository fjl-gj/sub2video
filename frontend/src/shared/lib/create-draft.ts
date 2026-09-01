export type CreateMode = 'image' | 'text' | 'video'
export type ImageVariant = 'keyframes' | 'multi'

export interface CreateFrameSlot {
  uploaded: boolean
  assetId: string
  picking: boolean
}

export interface CreateDraft {
  version: 1
  mode: CreateMode
  imageVariant: ImageVariant
  prompt: string
  ratio: string
  durationSec: string
  firstFrame: CreateFrameSlot
  lastFrame: CreateFrameSlot
  multiFrames: CreateFrameSlot[]
  referenceVideoUploaded: boolean
}

const STORAGE_KEY = 'sub2video.create-draft'

function emptySlot(): CreateFrameSlot {
  return { uploaded: false, assetId: '', picking: false }
}

export function emptyCreateDraft(): CreateDraft {
  return {
    version: 1,
    mode: 'image',
    imageVariant: 'keyframes',
    prompt: '',
    ratio: '16:9',
    durationSec: '15',
    firstFrame: emptySlot(),
    lastFrame: emptySlot(),
    multiFrames: [emptySlot(), emptySlot(), emptySlot()],
    referenceVideoUploaded: false,
  }
}

function sanitizeSlot(slot: CreateFrameSlot): CreateFrameSlot {
  return { uploaded: slot.uploaded, assetId: slot.assetId, picking: false }
}

function isCreateMode(value: unknown): value is CreateMode {
  return value === 'image' || value === 'text' || value === 'video'
}

function isImageVariant(value: unknown): value is ImageVariant {
  return value === 'keyframes' || value === 'multi'
}

function parseSlot(value: unknown): CreateFrameSlot {
  if (!value || typeof value !== 'object') return emptySlot()
  const slot = value as Partial<CreateFrameSlot>
  return {
    uploaded: Boolean(slot.uploaded),
    assetId: typeof slot.assetId === 'string' ? slot.assetId : '',
    picking: false,
  }
}

export function loadCreateDraft(): CreateDraft | null {
  if (typeof window === 'undefined') return null
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as Partial<CreateDraft>
    if (parsed.version !== 1) return null
    if (!isCreateMode(parsed.mode) || !isImageVariant(parsed.imageVariant)) return null
    return {
      version: 1,
      mode: parsed.mode,
      imageVariant: parsed.imageVariant,
      prompt: typeof parsed.prompt === 'string' ? parsed.prompt : '',
      ratio: typeof parsed.ratio === 'string' ? parsed.ratio : '16:9',
      durationSec: typeof parsed.durationSec === 'string' ? parsed.durationSec : '15',
      firstFrame: parseSlot(parsed.firstFrame),
      lastFrame: parseSlot(parsed.lastFrame),
      multiFrames: Array.isArray(parsed.multiFrames)
        ? parsed.multiFrames.slice(0, 3).map(parseSlot)
        : emptyCreateDraft().multiFrames,
      referenceVideoUploaded: Boolean(parsed.referenceVideoUploaded),
    }
  }
  catch {
    return null
  }
}

export function saveCreateDraft(draft: Omit<CreateDraft, 'version'>) {
  if (typeof window === 'undefined') return
  const payload: CreateDraft = {
    version: 1,
    mode: draft.mode,
    imageVariant: draft.imageVariant,
    prompt: draft.prompt,
    ratio: draft.ratio,
    durationSec: draft.durationSec,
    firstFrame: sanitizeSlot(draft.firstFrame),
    lastFrame: sanitizeSlot(draft.lastFrame),
    multiFrames: draft.multiFrames.map(sanitizeSlot),
    referenceVideoUploaded: draft.referenceVideoUploaded,
  }
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(payload))
}

export function clearCreateDraft() {
  if (typeof window === 'undefined') return
  window.localStorage.removeItem(STORAGE_KEY)
}
