import type { FilterSpec } from '~~/shared/api/types'

// Русские подписи операторов (набор из backend AllowedOperators).
export const OPERATOR_LABELS: Record<string, string> = {
  eq: 'равно',
  neq: 'не равно',
  contains: 'содержит',
  starts_with: 'начинается с',
  in: 'в списке',
  gt: 'больше',
  gte: 'больше или равно',
  lt: 'меньше',
  lte: 'меньше или равно',
  between: 'между',
  before: 'до',
  after: 'после',
  is_null: 'пусто',
  is_not_null: 'не пусто',
}

export function operatorLabel(op: string): string {
  return OPERATOR_LABELS[op] ?? op
}

export function operatorNeedsValue(op: string): boolean {
  return op !== 'is_null' && op !== 'is_not_null'
}

// Оператор по умолчанию для inline-фильтра колонки (одно поле «Все»):
// текст ищем по вхождению, остальные типы — по равенству.
export function defaultOperator(displayType: string): string {
  return displayType === 'text' ? 'contains' : 'eq'
}

export function operatorMultiValue(op: string): boolean {
  return op === 'in' || op === 'between'
}

// Собирает FilterSpec из черновика фильтра.
// in/between принимают значения через запятую; is_null/is_not_null — без значения.
// Возвращает null, если фильтр неполный (не применяем).
export function buildFilterSpec(column: string, operator: string, raw: string): FilterSpec | null {
  if (!column || !operator) return null
  if (!operatorNeedsValue(operator)) return { column, operator }

  const v = (raw ?? '').trim()
  if (v === '') return null

  if (operatorMultiValue(operator)) {
    const values = v
      .split(',')
      .map((x) => x.trim())
      .filter((x) => x !== '')
    if (values.length === 0) return null
    if (operator === 'between' && values.length !== 2) return null
    return { column, operator, values }
  }
  return { column, operator, value: v }
}

// Краткое человекочитаемое описание фильтра для чипа.
export function filterLabel(f: FilterSpec, columnLabel: string): string {
  const op = operatorLabel(f.operator)
  if (!operatorNeedsValue(f.operator)) return `${columnLabel}: ${op}`
  const val = f.values ? f.values.join(', ') : String(f.value ?? '')
  return `${columnLabel} ${op} ${val}`
}
