import { describe, it, expect } from 'vitest'
import { formatCell, filenameFromDisposition } from './format'

describe('formatCell', () => {
  it('NULL/undefined/пусто → прочерк', () => {
    expect(formatCell(null, 'text')).toBe('—')
    expect(formatCell(undefined, 'number')).toBe('—')
    expect(formatCell('', 'text')).toBe('—')
  })
  it('текст и число', () => {
    expect(formatCell('Клиент 1', 'text')).toBe('Клиент 1')
    expect(formatCell(0, 'number')).toBe('0')
    expect(formatCell('2629.35', 'money')).toBe('2629.35')
  })
  it('boolean → Да/Нет', () => {
    expect(formatCell(true, 'boolean')).toBe('Да')
    expect(formatCell(0, 'boolean')).toBe('Нет')
  })
  it('datetime форматируется (не прочерк, не ISO)', () => {
    const out = formatCell('2026-03-23T16:37:13Z', 'datetime')
    expect(out).not.toBe('—')
    expect(out).not.toContain('T')
  })
})

describe('filenameFromDisposition', () => {
  it('filename в кавычках', () => {
    expect(filenameFromDisposition('attachment; filename="demo-transactions_2026-08-23.xlsx"')).toBe(
      'demo-transactions_2026-08-23.xlsx',
    )
  })
  it('filename* (RFC 5987)', () => {
    expect(filenameFromDisposition("attachment; filename*=UTF-8''%D0%94%D0%B5%D0%BC%D0%BE.xlsx")).toBe('Демо.xlsx')
  })
  it('нет заголовка → fallback', () => {
    expect(filenameFromDisposition(null)).toBe('export.xlsx')
    expect(filenameFromDisposition(undefined, 'x.xlsx')).toBe('x.xlsx')
  })
})
