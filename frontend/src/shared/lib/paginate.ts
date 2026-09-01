export const PAGE_SIZE = 10
export const PLAZA_BATCH_SIZE = 12

export function slicePage<T>(items: readonly T[], page: number, pageSize = PAGE_SIZE): T[] {
  const start = Math.max(0, (page - 1) * pageSize)
  return items.slice(start, start + pageSize)
}
