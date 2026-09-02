const PREVIEW_IMAGES = [
  'https://images.unsplash.com/photo-1518709268805-4e9042af9f23?w=400&auto=format&fit=crop&q=80',
  'https://images.unsplash.com/photo-1509198397868-475647b2a1e5?w=400&auto=format&fit=crop&q=80',
  'https://images.unsplash.com/photo-1542751371-adc38448a05e?w=400&auto=format&fit=crop&q=80',
  'https://images.unsplash.com/photo-1534447677768-be436bb09401?w=400&auto=format&fit=crop&q=80',
  'https://images.unsplash.com/photo-1579783902614-a3fb3927b675?w=400&auto=format&fit=crop&q=80',
]

export function assetPreviewUrl(assetId: string, previewUrl?: string) {
  if (previewUrl) return previewUrl
  if (!assetId) return ''
  let hash = 0
  for (let i = 0; i < assetId.length; i += 1) hash = (hash + assetId.charCodeAt(i) * (i + 1)) % PREVIEW_IMAGES.length
  return PREVIEW_IMAGES[hash] ?? PREVIEW_IMAGES[0]
}
