import { describe, it, expect } from 'vitest'
import { buildFilterSpec, operatorNeedsValue, operatorLabel, filterLabel } from './filters'

describe('buildFilterSpec', () => {
  it('одиночное значение', () => {
    expect(buildFilterSpec('name', 'eq', 'Иван')).toEqual({ column: 'name', operator: 'eq', value: 'Иван' })
  })
  it('is_null — без значения', () => {
    expect(buildFilterSpec('name', 'is_null', '')).toEqual({ column: 'name', operator: 'is_null' })
  })
  it('in — по запятой в values', () => {
    expect(buildFilterSpec('code', 'in', '01, 02 ,03')).toEqual({
      column: 'code',
      operator: 'in',
      values: ['01', '02', '03'],
    })
  })
  it('between — ровно два значения', () => {
    expect(buildFilterSpec('amount', 'between', '10,20')).toEqual({
      column: 'amount',
      operator: 'between',
      values: ['10', '20'],
    })
    expect(buildFilterSpec('amount', 'between', '10')).toBeNull()
  })
  it('неполный фильтр → null', () => {
    expect(buildFilterSpec('', 'eq', 'x')).toBeNull()
    expect(buildFilterSpec('name', 'eq', '  ')).toBeNull()
  })
})

describe('operatorNeedsValue / label', () => {
  it('is_null/is_not_null не требуют значения', () => {
    expect(operatorNeedsValue('is_null')).toBe(false)
    expect(operatorNeedsValue('eq')).toBe(true)
  })
  it('метки на русском', () => {
    expect(operatorLabel('contains')).toBe('содержит')
    expect(operatorLabel('unknown')).toBe('unknown')
  })
  it('filterLabel', () => {
    expect(filterLabel({ column: 'name', operator: 'eq', value: 'Иван' }, 'ФИО')).toBe('ФИО равно Иван')
    expect(filterLabel({ column: 'name', operator: 'is_null' }, 'ФИО')).toBe('ФИО: пусто')
  })
})
