import { describe, expect, it } from 'vitest'
import { canCreate } from './noticeForm'

describe('noticeForm.canCreate', () => {
  it('неактивна без групп и когда все исключены', () => {
    expect(canCreate([])).toBe(false)
    expect(canCreate([{ gu: '1', excluded: true, cliIds: [5] }])).toBe(false)
  })

  it('неактивна, пока хотя бы у одной включённой группы нет адресата (EVGA-FR-034)', () => {
    expect(
      canCreate([
        { gu: '1', excluded: false, cliIds: [5] },
        { gu: '2', excluded: false, cliIds: [] },
      ]),
    ).toBe(false)
  })

  it('активна с несколькими адресатами; исключённые не мешают', () => {
    expect(
      canCreate([
        { gu: '1', excluded: false, cliIds: [5, 7] },
        { gu: '2', excluded: true, cliIds: [] },
      ]),
    ).toBe(true)
  })
})
