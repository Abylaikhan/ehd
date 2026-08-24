import type { ViewColumnDetail, ColumnConfigPayload } from '~~/shared/api/types'

// Допустимые типы отображения (совпадают с backend display_type).
export const DISPLAY_TYPES = [
  'text',
  'number',
  'money',
  'percent',
  'date',
  'datetime',
  'boolean',
  'enum',
  'json',
  'uuid',
] as const

export const ROW_SCOPE_MODES = [
  { label: 'По профилю пользователя', value: 'by_profile' },
  { label: 'Без ограничения строк', value: 'unrestricted' },
] as const

// Маппинг строк редактора колонок в payload для PUT .../columns.
export function columnsToPayload(rows: ViewColumnDetail[]): ColumnConfigPayload[] {
  return rows.map((c) => ({
    source_name: c.source_name,
    label: c.label,
    display_type: c.display_type,
    position: c.position,
    visible: c.visible,
    searchable: c.searchable,
    filterable: c.filterable,
    sortable: c.sortable,
    exportable: c.exportable,
    format: c.format || undefined,
    mask_rule: c.mask_rule || undefined,
    width: c.width || undefined,
    null_label: c.null_label || undefined,
  }))
}

// Метка и severity статуса витрины для Tag.
export function statusMeta(status: string): { label: string; severity: string } {
  switch (status) {
    case 'published':
      return { label: 'Опубликована', severity: 'success' }
    case 'draft':
      return { label: 'Черновик', severity: 'warn' }
    case 'disabled':
      return { label: 'Отключена', severity: 'secondary' }
    case 'schema_error':
      return { label: 'Ошибка схемы', severity: 'danger' }
    case 'archived':
      return { label: 'Архив', severity: 'contrast' }
    default:
      return { label: status, severity: 'secondary' }
  }
}
