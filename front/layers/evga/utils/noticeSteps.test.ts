import { describe, it, expect } from 'vitest'
import { currentNoticeStep, NOTICE_STEP_LABELS } from './noticeSteps'

describe('currentNoticeStep', () => {
  it('проект без исходящего → 0', () => {
    expect(currentNoticeStep(1, { exists: false })).toBe(0)
    expect(currentNoticeStep(null, null)).toBe(0)
  })
  it('на согласовании → 1', () => {
    expect(currentNoticeStep(2, { exists: false })).toBe(1)
  })
  it('на подписании → 1', () => {
    expect(currentNoticeStep(5, null)).toBe(1)
  })
  it('на доработке → 1', () => {
    expect(currentNoticeStep(12, null)).toBe(1)
  })
  it('исходящее создано без номера → 2 (канцелярия)', () => {
    expect(currentNoticeStep(5, { exists: true, doc_num: '' })).toBe(2)
    expect(currentNoticeStep(5, { exists: true, doc_num: null })).toBe(2)
  })
  it('исходящее зарегистрировано → 3', () => {
    expect(currentNoticeStep(5, { exists: true, doc_num: '05/2026/000583' })).toBe(3)
  })
  it('заявка «Исполнен» → 3', () => {
    expect(currentNoticeStep(3, { exists: true, doc_num: '05/2026/000583' })).toBe(3)
  })
  it('меток ровно 4', () => {
    expect(NOTICE_STEP_LABELS).toHaveLength(4)
  })
})
