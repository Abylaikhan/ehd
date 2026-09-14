package domain

import "time"

// DefaultDeadlineWarnDays — окно «истекающий» в РАБОЧИХ днях (EVGA-FR-080; подтверждено
// аналитиком, ответ В4-2 от 10.09.2026: «3 рабочих дня»). Конфигурируется EVGA_DEADLINE_WARN_DAYS.
const DefaultDeadlineWarnDays = 3

// Состояния контрольного срока записи (EVGA-FR-080).
const (
	DeadlineNone     = ""         // срок неактуален / не задан / до порога
	DeadlineExpiring = "expiring" // срок скоро истекает (в пределах warnDays)
	DeadlineExpired  = "expired"  // срок истёк
)

// deadlineStatuses — статусы, для которых контрольный срок актуален: ответ ГУ ещё ожидается
// (спека 013 §2). Для решённых статусов состояние всегда пустое.
var deadlineStatuses = map[int64]bool{
	StatusNoticeSent:  true, // 4 «Уведомление направлено»
	StatusWaitingDocs: true, // 6 «Ожидание документов»
}

// DeadlineState вычисляет состояние контрольного срока записи по its_out.exec_due_time.
// execDue — строка "YYYY-MM-DD" (пусто = срок не задан); warnDays<=0 → берётся дефолт.
// «Истекающий» — срок наступает в пределах warnDays РАБОЧИХ дней от сегодня (ответ В4-2);
// сб/вс не сокращают окно (порог сдвигается через AddBusinessDays).
func DeadlineState(execDue string, statusID *int64, warnDays int, now time.Time) string {
	if statusID == nil || !deadlineStatuses[*statusID] {
		return DeadlineNone
	}
	due, err := time.ParseInLocation("2006-01-02", execDue, now.Location())
	if err != nil {
		return DeadlineNone
	}
	if warnDays <= 0 {
		warnDays = DefaultDeadlineWarnDays
	}
	// нормализуем к началу суток
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dueDay := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, now.Location())

	if dueDay.Before(today) {
		return DeadlineExpired
	}
	// порог — дата через warnDays рабочих дней; срок на неё или раньше → «истекающий».
	if !dueDay.After(AddBusinessDays(today, warnDays)) {
		return DeadlineExpiring
	}
	return DeadlineNone
}
