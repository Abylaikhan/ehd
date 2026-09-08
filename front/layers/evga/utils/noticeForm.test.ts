import { describe, expect, it } from 'vitest'
import { canCreate } from './noticeForm'

describe('noticeForm.canCreate', () => {
  it('неактивна без групп и когда все исключены', () => {
    expect(canCreate([])).toBe(false)
    expect(canCreate([{ gu: '1', excluded: true, cliId: 5 }])).toBe(false)
  })

  it('неактивна, пока хотя бы у одной включённой группы нет адресата (EVGA-FR-034)', () => {
    expect(
      canCreate([
        { gu: '1', excluded: false, cliId: 5 },
        { gu: '2', excluded: false, cliId: null },
      ]),
    ).toBe(false)
  })

  it('активна, когда у всех включённых есть адресат; исключённые не мешают', () => {
    expect(
      canCreate([
        { gu: '1', excluded: false, cliId: 5 },
        { gu: '2', excluded: true, cliId: null },
      ]),
    ).toBe(true)
  })
})
