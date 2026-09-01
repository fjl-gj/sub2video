import { onUnmounted, watch, type Ref } from 'vue'

export function useInfiniteScroll(
  sentinel: Ref<HTMLElement | null>,
  onLoadMore: () => void,
  options?: { rootMargin?: string },
) {
  let observer: IntersectionObserver | null = null

  function disconnect() {
    observer?.disconnect()
    observer = null
  }

  function observe(element: HTMLElement) {
    disconnect()
    observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) onLoadMore()
      },
      { rootMargin: options?.rootMargin ?? '240px' },
    )
    observer.observe(element)
  }

  watch(
    sentinel,
    (element) => {
      if (element) observe(element)
      else disconnect()
    },
    { immediate: true },
  )

  onUnmounted(disconnect)
}
