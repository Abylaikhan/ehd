import type {
  EvgaBulkReport,
  EvgaCard,
  EvgaCliOrg,
  EvgaHistoryEntry,
  EvgaNoticeCard,
  EvgaNoticeCreateResult,
  EvgaNoticeListResponse,
  EvgaNoticePreview,
  EvgaParticipant,
  EvgaReferences,
  EvgaRouteView,
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

  // --- уведомления (backend-спека 009) ---

  const noticePreview = (ids: number[]) =>
    api<EvgaNoticePreview>('/api/v1/evga/notices/preview', { method: 'POST', body: { ids } })

  const noticeCreate = (groups: { record_ids: number[]; its_cli_id: number }[]) =>
    api<EvgaNoticeCreateResult>('/api/v1/evga/notices', { method: 'POST', body: { groups } })

  const noticeList = (p: { page?: number; page_size?: number; status_id?: number | null; docnum?: string; department_id?: number | null }) =>
    api<EvgaNoticeListResponse>('/api/v1/evga/notices', { query: toQuery(p as EvgaRegistryParams) })

  const noticeGet = (id: number) => api<EvgaNoticeCard>(`/api/v1/evga/notices/${id}`)

  const noticeDelete = (id: number) =>
    api<{ deleted: boolean }>(`/api/v1/evga/notices/${id}`, { method: 'DELETE' })

  const cliSearch = (q: string) =>
    api<{ items: EvgaCliOrg[] }>('/api/v1/evga/cli', { query: { q } })

  // --- согласование (backend-спека 010) ---

  const routeGet = (noticeId: number) =>
    api<EvgaRouteView>(`/api/v1/evga/notices/${noticeId}/route`)

  const routePut = (noticeId: number, approverIds: number[], outgoingUserId: number) =>
    api<EvgaRouteView>(`/api/v1/evga/notices/${noticeId}/route`, {
      method: 'PUT',
      body: { approver_ids: approverIds, outgoing_user_id: outgoingUserId },
    })

  const participants = (noticeId: number, q: string) =>
    api<{ items: EvgaParticipant[] }>(`/api/v1/evga/notices/${noticeId}/participants`, { query: { q } })

  const noticeSubmit = (noticeId: number) =>
    api<{ docnum: string; status_id: number }>(`/api/v1/evga/notices/${noticeId}/submit`, { method: 'POST' })

  const noticeApprove = (noticeId: number, comment?: string) =>
    api<{ approved: boolean }>(`/api/v1/evga/notices/${noticeId}/approve`, { method: 'POST', body: { comment } })

  const noticeReject = (noticeId: number, comment: string) =>
    api<{ rejected: boolean }>(`/api/v1/evga/notices/${noticeId}/reject`, { method: 'POST', body: { comment } })

  return {
    registry, card, references, exportRegistry, changeStatus, bulkStatus, history,
    noticePreview, noticeCreate, noticeList, noticeGet, noticeDelete, cliSearch,
    routeGet, routePut, participants, noticeSubmit, noticeApprove, noticeReject,
  }
}

// Черновик формирования: выбранные в реестре записи (передача между страницами).
export const useEvgaNoticeDraft = () => useState<number[]>('evga-notice-draft', () => [])
