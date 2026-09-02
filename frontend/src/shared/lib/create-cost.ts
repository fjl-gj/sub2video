export type CreateResolution = '720P' | '768P' | '1080P'

const RATES: Record<CreateResolution, number> = {
  '720P': 0.4,
  '768P': 0.5,
  '1080P': 0.8,
}

export function resolutionRate(resolution: CreateResolution) {
  return RATES[resolution]
}

export function estimateCreateCost(durationSec: number, resolution: CreateResolution) {
  if (!Number.isFinite(durationSec) || durationSec <= 0) return 0
  return durationSec * resolutionRate(resolution)
}
