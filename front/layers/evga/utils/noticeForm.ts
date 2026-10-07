// Логика экрана формирования уведомлений (спека 007-evga-notices-ui).

export interface GroupState {
  gu: string
  excluded: boolean
  cliIds: number[]
}

/** Кнопка «Создать» активна: остаётся ≥1 включённая группа и у каждой выбран
 * хотя бы один адресат (EVGA-FR-034). На уведомление можно выбрать несколько адресатов. */
export function canCreate(groups: GroupState[]): boolean {
  const active = groups.filter((g) => !g.excluded)
  if (active.length === 0) return false
  return active.every((g) => g.cliIds.length > 0 && g.cliIds.every((id) => id > 0))
}
