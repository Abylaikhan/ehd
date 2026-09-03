import type {
  ItemsResponse,
  UserViewListItem,
  ViewMeta,
  QuerySpec,
  QueryResult,
  CountResult,
  MenuNode,
} from '~~/shared/api/types'
import { filenameFromDisposition } from '../utils/format'

// Клиент пользовательского Reporter API (backend slices 005/006).
export function useReporterViews() {
  const api = useApi()
  const enc = (s: string) => encodeURIComponent(s)

  const list = () => api<ItemsResponse<UserViewListItem>>('/api/v1/reporter/views')

  const navigation = () => api<ItemsResponse<MenuNode>>('/api/v1/reporter/navigation')

  const meta = (slug: string) => api<ViewMeta>(`/api/v1/reporter/views/${enc(slug)}`)

  const query = (slug: string, spec: QuerySpec) =>
    api<QueryResult>(`/api/v1/reporter/views/${enc(slug)}/query`, { method: 'POST', body: spec })

  const count = (slug: string, spec: QuerySpec) =>
    api<CountResult>(`/api/v1/reporter/views/${enc(slug)}/count`, { method: 'POST', body: spec })

  // Экспорт: получаем бинарный blob + имя из Content-Disposition и запускаем скачивание браузером.
  const exportView = async (slug: string, spec: QuerySpec) => {
    const res = await api.raw(`/api/v1/reporter/views/${enc(slug)}/export`, {
      method: 'POST',
      body: spec,
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

  return { list, navigation, meta, query, count, exportView }
}
