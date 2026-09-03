// Типы API ЕХД — зеркало back/openapi/openapi.yaml (Auth Module).
// Можно заменить на сгенерированные openapi-typescript; сейчас поддерживаются вручную.

export type UserStatus = 'pending' | 'active' | 'blocked'

export interface ApiError {
  request_id: string
  error: {
    code: string
    message: string
    details?: { field: string; reason: string }[]
  }
}

export interface Me {
  user_id: string
  login: string
  is_admin: boolean
  has_password: boolean
  roles: string[]
  region_codes: string[]
  department_codes: string[]
}

export interface LoginResponse {
  user_id: string
  login: string
  expires_at: string
  password_change_required: boolean
}

export interface UserView {
  id: string
  login: string
  email: string
  full_name: string
  iin_masked: string
  iin_verified: boolean
  status: UserStatus
  failed_attempts: number
  roles: string[]
  region_codes: string[]
  department_codes: string[]
  created_at: string
}

export interface UserList {
  items: UserView[]
  total_count: number
  page: number
  page_size: number
}

export interface Role {
  id: string
  code: string
  name_ru: string
  name_kk: string
  status: string
}

export interface Reference {
  id: string
  code: string
  name_ru: string
  name_kk: string
  status: string
}

export interface RegisterRequest {
  login: string
  password: string
  iin: string
  full_name: string
  email: string
  phone?: string
}

export interface UpdateUserRequest {
  iin_verified?: boolean
  status?: UserStatus
  role_ids?: string[]
  region_ids?: string[]
  department_ids?: string[]
}

export interface CreateRoleRequest {
  code: string
  name_ru: string
  name_kk?: string
}

export interface ItemsResponse<T> {
  items: T[]
}

// --- Reporter: пользовательский просмотр витрин (backend slices 005/006) ---

export interface UserViewListItem {
  slug: string
  name: string
  description: string
}

export interface ColumnMeta {
  source_name: string
  label: string
  display_type: string
  searchable: boolean
  filterable: boolean
  sortable: boolean
  operators: string[]
}

export interface ViewMeta {
  slug: string
  name: string
  description: string
  page_size_default: number
  page_size_min: number
  page_size_max: number
  columns: ColumnMeta[]
}

export interface FilterSpec {
  column: string
  operator: string
  value?: unknown
  values?: unknown[]
}

export interface QuerySpec {
  filters?: FilterSpec[]
  search?: string
  sort?: { column?: string; dir?: 'asc' | 'desc' }
  page_size?: number
  cursor?: string
}

export interface ResultColumn {
  source_name: string
  label: string
  display_type: string
}

export interface QueryResult {
  columns: ResultColumn[]
  rows: Record<string, unknown>[]
  page: { page_size: number; next_cursor: string }
}

export interface CountResult {
  total_count: number
}

// --- Reporter admin: конструктор витрин (backend slices 003–005) ---

export interface DataSourceSummary {
  id: string
  code: string
  name: string
  host: string
  port: number
  protocol: string
  username: string
  status: string
}

export interface IntrospectDatabase {
  name: string
}

export interface IntrospectTable {
  name: string
  engine: string
  kind: string
}

export interface IntrospectColumn {
  name: string
  type: string
  position: number
  nullable: boolean
  comment: string
  in_primary_key: boolean
  in_sorting_key: boolean
}

// Полная запись представления (мета + параметры), общая для списка и карточки.
export interface ViewSummary {
  id: string
  name: string
  slug: string
  description: string
  data_source_id: string
  database: string
  table: string
  source_mode: string
  status: string
  revision: number
  schema_hash: string
  page_size_default: number
  page_size_min: number
  page_size_max: number
  default_sort_column: string
  default_sort_dir: string
  export_row_limit: number
  row_scope_mode: string
  keyset_column: string
  keyset_dir: string
  row_scope_region_column: string
  row_scope_department_column: string
  published_at: string | null
  created_at: string
  updated_at: string
}

export interface ViewColumnDetail {
  source_name: string
  source_type: string
  label: string
  display_type: string
  position: number
  visible: boolean
  searchable: boolean
  filterable: boolean
  sortable: boolean
  exportable: boolean
  format: string
  mask_rule: string
  width: number
  null_label: string
}

export interface ViewDetail extends ViewSummary {
  columns: ViewColumnDetail[]
  role_codes: string[]
}

export interface CreateViewPayload {
  name: string
  slug: string
  description?: string
  data_source_id: string
  database: string
  table: string
}

export interface UpdateViewMetaPayload {
  name: string
  slug: string
  description?: string
  page_size_default: number
  page_size_min: number
  page_size_max: number
  default_sort_column?: string
  default_sort_dir?: string
  export_row_limit: number
  row_scope_mode: string
  keyset_column?: string
  keyset_dir?: string
  row_scope_region_column?: string
  row_scope_department_column?: string
}

export interface ColumnConfigPayload {
  source_name: string
  label: string
  display_type: string
  position: number
  visible: boolean
  searchable: boolean
  filterable: boolean
  sortable: boolean
  exportable: boolean
  format?: string
  mask_rule?: string
  width?: number
  null_label?: string
}
