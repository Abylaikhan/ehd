import { describe, it, expect } from 'vitest'
import { columnsToPayload, statusMeta } from './viewForm'
import type { ViewColumnDetail } from '~~/shared/api/types'

function col(over: Partial<ViewColumnDetail>): ViewColumnDetail {
  return {
    source_name: 'id',
    source_type: 'UInt64',
    label: 'ID',
    display_type: 'number',
    position: 1,
    visible: true,
    searchable: false,
    filterable: false,
    sortable: false,
    exportable: false,
    format: '{}',
    mask_rule: 'none',
    width: 0,
    null_label: '',
    ...over,
  }
}

describe('columnsToPayload', () => {
  it('переносит флаги и типы, пустые опции → undefined', () => {
    const out = columnsToPayload([col({ visible: true, filterable: true, width: 0, null_label: '' })])
    expect(out[0]).toMatchObject({ source_name: 'id', label: 'ID', display_type: 'number', visible: true, filterable: true })
    expect(out[0].width).toBeUndefined()
    expect(out[0].null_label).toBeUndefined()
  })
  it('не теряет source_type-независимые правки', () => {
    const out = columnsToPayload([col({ label: 'Идентификатор', display_type: 'text', position: 5, exportable: true })])
    expect(out[0].label).toBe('Идентификатор')
    expect(out[0].display_type).toBe('text')
    expect(out[0].position).toBe(5)
    expect(out[0].exportable).toBe(true)
  })
})

describe('statusMeta', () => {
  it('известные статусы', () => {
    expect(statusMeta('published').severity).toBe('success')
    expect(statusMeta('draft').label).toBe('Черновик')
    expect(statusMeta('disabled').label).toBe('Отключена')
  })
  it('неизвестный статус → сам код', () => {
    expect(statusMeta('weird').label).toBe('weird')
  })
})
