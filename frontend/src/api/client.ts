import { http } from '@/api/http'

export async function fetchOverview() {
  const { data } = await http.get('/client/overview')
  return data
}

export async function fetchWallet() {
  const { data } = await http.get('/client/wallet')
  return data as { balance: number; products: Array<{ id: string; amount: number; bonus: number; price: number }> }
}

export async function redeemCode(code: string) {
  const { data } = await http.post('/client/wallet/redeem', { code })
  return data as { credited: number; wallet: { balance: number } }
}

export async function quoteTask(payload: { workflowId: string; durationSeconds: number; resolution?: string }) {
  const { data } = await http.post('/client/tasks/quote', payload)
  return data as {
    estimatedCost: number
    pricePerSecond: number
    priceSnapshot: { ruleId: string; version: string; pricePerSecond: number }
  }
}

export async function createTask(payload: Record<string, unknown>) {
  const { data } = await http.post('/client/tasks', payload)
  return data as {
    task: { id: string; title: string; status: string; cost: number; workflow: string }
    wallet: { balance: number }
  }
}

export async function uploadImage(file: File) {
  const form = new FormData()
  form.append('file', file)
  const { data } = await http.post('/client/uploads', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 60000,
  })
  return data as {
    id: string
    fileName: string
    mimeType: string
    sizeBytes: number
    url: string
  }
}

export async function listTasks(params?: { status?: string; page?: number; pageSize?: number }) {
  const { data } = await http.get('/client/tasks', { params })
  return data
}
