export type StudioMode = {
  type: 'text' | 'multi_image' | 'first_last'
  title: string
  label: string
  description: string
  available: boolean
  availability: string
}

export type OverviewTask = {
  id: string
  title: string
  mode: string
  status: string
  progress: number
  duration: string
  resolution: string
  cost: number
  thumbUrl?: string
}

export type RecentFilm = Pick<OverviewTask, 'id' | 'title' | 'status' | 'resolution' | 'duration' | 'cost' | 'thumbUrl'>

export type OverviewData = {
  workspace: { displayName: string }
  wallet: { balance: number }
  storage: { usedGb: number; quotaGb: number; fileCount: number }
  tasks: {
    activeCount: number
    activeTask: OverviewTask | null
    recentCompleted: RecentFilm[]
  }
}
