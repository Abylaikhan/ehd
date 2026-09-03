import type {
  ItemsResponse,
  DataSourceSummary,
  DataSourcePayload,
  IntrospectDatabase,
  IntrospectTable,
  IntrospectColumn,
  ViewSummary,
  ViewDetail,
  CreateViewPayload,
  UpdateViewMetaPayload,
  ColumnConfigPayload,
  Role,
  QuerySpec,
  QueryResult,
  MenuItem,
  MenuItemPayload,
} from '~~/shared/api/types'

// Админ-клиент Reporter (backend slices 003–005): источники, интроспекция, витрины, роли.
export function useReporterAdmin() {
  const api = useApi()
  const enc = encodeURIComponent
  const base = '/api/v1/reporter/admin'

  const sources = {
    list: () => api<ItemsResponse<DataSourceSummary>>(`${base}/sources`),
    get: (id: string) => api<DataSourceSummary>(`${base}/sources/${enc(id)}`),
    create: (payload: DataSourcePayload) =>
      api<DataSourceSummary>(`${base}/sources`, { method: 'POST', body: payload }),
    update: (id: string, payload: DataSourcePayload) =>
      api<DataSourceSummary>(`${base}/sources/${enc(id)}`, { method: 'PATCH', body: payload }),
    // Проверка связи без сохранения (по параметрам формы) и по сохранённому источнику.
    testParams: (payload: DataSourcePayload) =>
      api<{ ok: boolean }>(`${base}/sources/test`, { method: 'POST', body: payload }),
    test: (id: string) => api<{ ok: boolean }>(`${base}/sources/${enc(id)}/test`, { method: 'POST' }),
    activate: (id: string) => api<{ status: string }>(`${base}/sources/${enc(id)}/activate`, { method: 'POST' }),
    deactivate: (id: string) =>
      api<{ status: string }>(`${base}/sources/${enc(id)}/deactivate`, { method: 'POST' }),
    databases: (id: string) => api<ItemsResponse<IntrospectDatabase>>(`${base}/sources/${enc(id)}/databases`),
    tables: (id: string, db: string) =>
      api<ItemsResponse<IntrospectTable>>(`${base}/sources/${enc(id)}/databases/${enc(db)}/tables`),
    columns: (id: string, db: string, table: string) =>
      api<ItemsResponse<IntrospectColumn>>(`${base}/sources/${enc(id)}/databases/${enc(db)}/tables/${enc(table)}/columns`),
  }

  const views = {
    list: () => api<ItemsResponse<ViewSummary>>(`${base}/views`),
    get: (id: string) => api<ViewDetail>(`${base}/views/${enc(id)}`),
    create: (payload: CreateViewPayload) => api<ViewSummary>(`${base}/views`, { method: 'POST', body: payload }),
    updateMeta: (id: string, payload: UpdateViewMetaPayload) =>
      api<ViewSummary>(`${base}/views/${enc(id)}`, { method: 'PATCH', body: payload }),
    updateColumns: (id: string, columns: ColumnConfigPayload[]) =>
      api(`${base}/views/${enc(id)}/columns`, { method: 'PUT', body: { columns } }),
    setPermissions: (id: string, role_codes: string[]) =>
      api(`${base}/views/${enc(id)}/permissions`, { method: 'PUT', body: { role_codes } }),
    preview: (id: string, spec: QuerySpec) =>
      api<QueryResult>(`${base}/views/${enc(id)}/preview`, { method: 'POST', body: spec }),
    publish: (id: string) => api<ViewSummary>(`${base}/views/${enc(id)}/publish`, { method: 'POST' }),
    disable: (id: string) => api(`${base}/views/${enc(id)}/disable`, { method: 'POST' }),
    remove: (id: string) => api(`${base}/views/${enc(id)}`, { method: 'DELETE' }),
  }

  const menu = {
    list: () => api<ItemsResponse<MenuItem>>(`${base}/menu`),
    create: (payload: MenuItemPayload) => api<MenuItem>(`${base}/menu`, { method: 'POST', body: payload }),
    update: (id: string, payload: MenuItemPayload) =>
      api<MenuItem>(`${base}/menu/${enc(id)}`, { method: 'PATCH', body: payload }),
    remove: (id: string) => api(`${base}/menu/${enc(id)}`, { method: 'DELETE' }),
  }

  const roles = () => api<ItemsResponse<Role>>('/api/v1/auth/admin/roles')

  return { sources, views, roles, menu }
}
