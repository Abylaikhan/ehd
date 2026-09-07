import { describe, expect, it } from 'vitest'
import { allowedTargets, manualTargets, requiredFields, validateForm } from './statusFlow'

describe('statusFlow', () => {
  it('матрица §7.2: разрешённые цели по каждому статусу', () => {
    expect(allowedTargets(1)).toEqual([2])
    expect(allowedTargets(4)).toEqual([6, 11, 2])
    expect(allowedTargets(6)).toEqual([11, 2])
    expect(allowedTargets(11)).toEqual([7, 12])
    expect(allowedTargets(7)).toEqual([8, 12])
    expect(allowedTargets(12)).toEqual([8])
    expect(allowedTargets(2)).toEqual([]) // терминальный
    expect(allowedTargets(8)).toEqual([]) // терминальный
    expect(allowedTargets(null)).toEqual([])
    expect(allowedTargets(99)).toEqual([]) // неизвестный статус
  })

  it('статус 4 не назначается вручную ни из одного состояния', () => {
    expect(manualTargets()).not.toContain(4)
    expect(manualTargets()).not.toContain(1)
  })

  it('обязательные поля по целевому статусу', () => {
    expect(requiredFields(2).note).toBe(true)
    expect(requiredFields(7).amount).toBe(true)
    expect(requiredFields(8).refund).toBe(true)
    // «Аудит»: мероприятие опционально (ответ аналитика 07.09.2026)
    expect(requiredFields(12).activity).toBe(false)
    expect(requiredFields(12).showActivity).toBe(true)
    expect(requiredFields(6)).toEqual({ note: false, amount: false, refund: false, activity: false, showActivity: false })
  })

  it('валидация формы', () => {
    const empty = { note: '', amount: '', refund: '', activityId: null }
    expect(validateForm(null, empty)).toContain('целевой статус')
    expect(validateForm(2, empty)).toContain('комментарий')
    expect(validateForm(2, { ...empty, note: 'ложное срабатывание' })).toBeNull()
    expect(validateForm(7, empty)).toContain('сумму к возмещению')
    expect(validateForm(7, { ...empty, amount: '100.50' })).toBeNull()
    expect(validateForm(8, empty)).toContain('сумму возмещения')
    expect(validateForm(12, empty)).toBeNull() // «Аудит» — без обязательных полей
    expect(validateForm(6, empty)).toBeNull()
  })
})
