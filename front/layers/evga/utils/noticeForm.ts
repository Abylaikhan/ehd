// Логика экрана формирования уведомлений (спека 007-evga-notices-ui).

export interface GroupState {
  gu: string
  excluded: boolean
  cliId: number | null
}

/** Кнопка «Создать» активна: остаётся ≥1 включённая группа и у каждой выбран адресат
 * (EVGA-FR-034: неактивна, пока хотя бы у одной группы не выбран адресат). */
export function canCreate(groups: GroupState[]): boolean {
  const active = groups.filter((g) => !g.excluded)
  if (active.length === 0) return false
  return active.every((g) => g.cliId != null && g.cliId > 0)
}
