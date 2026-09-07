// Клиентская копия матрицы переходов §7.2 (UX-подсказка для диалога смены статуса).
// Источник истины — backend (спека 008): расхождение даёт максимум 422 с причиной.

/** Разрешённые ручные переходы: из статуса → список целевых. Статус 4 назначает только система. */
const transitions: Record<number, number[]> = {
  1: [2],
  4: [6, 11, 2],
  6: [11, 2],
  11: [7, 12],
  7: [8, 12],
  12: [8],
  2: [],
  8: [],
}

/** Целевые статусы, доступные из текущего (для одиночной смены). */
export function allowedTargets(current: number | null): number[] {
  if (current == null) return []
  return transitions[current] ?? []
}

/** Все статусы, назначаемые вручную хотя бы из одного состояния (для bulk-диалога). */
export function manualTargets(): number[] {
  const set = new Set<number>()
  for (const targets of Object.values(transitions)) for (const t of targets) set.add(t)
  return [...set].sort((a, b) => a - b)
}

/** Какие поля обязательны для перехода в целевой статус.
 * «Аудит» (12) по ответу аналитика 07.09.2026 обязательных полей не имеет —
 * мероприятие показывается опционально (showActivity). */
export function requiredFields(target: number | null): { note: boolean; amount: boolean; refund: boolean; activity: boolean; showActivity: boolean } {
  return {
    note: target === 2,
    amount: target === 7,
    refund: target === 8,
    activity: false,
    showActivity: target === 12,
  }
}

/** Валидация формы диалога до отправки (сервер валидирует повторно). */
export function validateForm(
  target: number | null,
  form: { note: string; amount: string; refund: string; activityId: number | null },
): string | null {
  if (target == null) return 'Выберите целевой статус'
  const req = requiredFields(target)
  if (req.note && !form.note.trim()) return 'Укажите комментарий — для «Не подтверждено» он обязателен'
  if (req.amount && !(parseFloat(form.amount) > 0)) return 'Укажите сумму к возмещению'
  if (req.refund && !(parseFloat(form.refund) > 0)) return 'Укажите сумму возмещения'
  return null
}
