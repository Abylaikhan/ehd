// Шаги исходящего уведомления — сближение с экраном платформы (спека 009-evga-outgoing-ui).
// Истина по статусам — на бэке; здесь только визуальный прогресс.

export const NOTICE_STEP_LABELS = ['Создан', 'Подписание', 'Канцелярия', 'Выполнен'] as const

export interface NoticeStepOut {
  exists?: boolean
  doc_num?: string | null
}

// currentNoticeStep — индекс текущего шага (0..3) по статусу заявки (its_req.stat_id)
// и состоянию исходящего.
export function currentNoticeStep(statusId: number | null | undefined, out?: NoticeStepOut | null): number {
  const registered = Boolean(out?.exists && out?.doc_num)
  if (statusId === 3 || registered) return 3 // «Исполнен» либо исходящее зарегистрировано
  if (out?.exists) return 2 // исходящее создано, ожидает канцелярию/регистрацию
  if (statusId === 2 || statusId === 5 || statusId === 12) return 1 // согласование / подписание / доработка
  return 0 // «Проект создан»
}
