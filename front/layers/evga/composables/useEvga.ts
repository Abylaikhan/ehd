import type {
  EvgaBulkReport,
  EvgaCard,
  EvgaHistoryEntry,
  EvgaReferences,
  EvgaRegistryResponse,
  EvgaStatusChange,
} from '~~/shared/api/types'
import { filenameFromDisposition } from '../../reporter/utils/format'

// Параметры реестра (backend-спека 007, FR-6/7). Пустые значения не передаются.
export interface EvgaRegistryParams {
  profile_id?: number | null
  status_id?: number | null
  god?: number | null
  mes?: number | null
  paymentdate_from?: string | null
  paymentdate_to?: string | null
  gu?: string
  sendername?: string
  iin?: string
  amount_from?: number | null
  amount_to?: number | null
  in_notice?: boolean | null
  notice_num?: string
  department_id?: number | null
  page?: number
  page_size?: number
  sort?: string
  order?: 'asc' | 'desc'
}

function toQuery(p: EvgaRegistryParams): Record<string, string> {
  const q: Record<string, string> = {}
  for (const [k, v] of Object.entries(p)) {
    if (v === undefined || v === null || v === '') continue
    q[k] = String(v)
  }
  return q
}

// Клиент API модуля ОБМ ЕВГА.
export function useEvga() {
  const api = useApi()

  const registry = (p: EvgaRegistryParams) =>
    api<EvgaRegistryResponse>('/api/v1/evga/registry', { query: toQuery(p) })

  const card = (id: number) => api<EvgaCard>(`/api/v1/evga/registry/${id}`)

  const references = () => api<EvgaReferences>('/api/v1/evga/references')

  // Excel: blob + имя из Content-Disposition, скачивание браузером (как в reporter).
  const exportRegistry = async (p: EvgaRegistryParams) => {
    const res = await api.raw('/api/v1/evga/registry/export', {
      query: toQuery(p),
      responseType: 'blob',
    })
    const blob = res._data as Blob
    const name = filenameFromDisposition(res.headers.get('content-disposition'))
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = name
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
  }

  // --- статусы (backend-спека 008) ---

  const changeStatus = (id: number, body: EvgaStatusChange) =>
    api<EvgaCard>(`/api/v1/evga/registry/${id}/status`, { method: 'POST', body })

  const bulkStatus = (ids: number[], body: EvgaStatusChange) =>
    api<EvgaBulkReport>('/api/v1/evga/registry/status/bulk', { method: 'POST', body: { ids, ...body } })

  const history = (id: number) =>
    api<{ items: EvgaHistoryEntry[] }>(`/api/v1/evga/registry/${id}/history`)

  return { registry, card, references, exportRegistry, changeStatus, bulkStatus, history }
}
