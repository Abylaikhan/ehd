// Логика формы маршрута (спека 008-evga-approval-ui).

/** Маршрут заполнен для отправки: ≥1 согласующий без дублей и ответственный за исходящее
 * (EVGA-FR-050/051; сервер валидирует повторно — AT-09). */
export function routeComplete(approverIds: number[], outgoingId: number | null): boolean {
  if (approverIds.length === 0 || outgoingId == null || outgoingId <= 0) return false
  if (approverIds.some((id) => id <= 0)) return false
  return new Set(approverIds).size === approverIds.length
}
