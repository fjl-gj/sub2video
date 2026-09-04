import { getAccessToken } from '@/api/http'

/** Build an authenticated media URL for <img>/<video> tags. */
export function assetPreviewUrl(assetId: string, previewUrl?: string) {
  if (previewUrl) {
    if (previewUrl.startsWith('http://') || previewUrl.startsWith('https://') || previewUrl.startsWith('blob:') || previewUrl.startsWith('data:')) {
      return previewUrl
    }
    const token = getAccessToken()
    if (!token) return previewUrl
    const sep = previewUrl.includes('?') ? '&' : '?'
    return `${previewUrl}${sep}access_token=${encodeURIComponent(token)}`
  }
  return ''
}
