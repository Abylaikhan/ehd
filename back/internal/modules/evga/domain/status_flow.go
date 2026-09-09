package domain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Статусы записи витрины по id (ТЗ §7.1, EVGA-BR-010: работать по id, не по code).
const (
	StatusInWork       int64 = 1  // В работе у ДВГА (стартовый, system)
	StatusNoViolations int64 = 2  // Не подтверждено (терминальный)
	StatusNoticeSent   int64 = 4  // Уведомление направлено (только system)
	StatusWaitingDocs  int64 = 6  // Ожидание документов (от ГУ)
	StatusRefundDue    int64 = 7  // Подлежит возмещению
	StatusRefunded     int64 = 8  // Возмещено (терминальный)
	StatusConfirmed    int64 = 11 // Подтверждено
	StatusAudit        int64 = 12 // Аудит
)

// statusNames — человекочитаемые наименования статусов (ТЗ §7.1) для сообщений об
// отклонении переходов (EVGA-BR-014): в тексте показываем название, а не числовой id.
var statusNames = map[int64]string{
	StatusInWork:       "В работе у ДВГА",
	StatusNoViolations: "Не подтверждено",
	StatusNoticeSent:   "Уведомление направлено",
	StatusWaitingDocs:  "Ожидание документов",
	StatusRefundDue:    "Подлежит возмещению",
	StatusRefunded:     "Возмещено",
	StatusConfirmed:    "Подтверждено",
	StatusAudit:        "Аудит",
}

// statusName — наименование статуса для сообщений; неизвестный id → «№N».
func statusName(id int64) string {
	if t, ok := statusNames[id]; ok {
		return t
	}
	return fmt.Sprintf("№%d", id)
}

// Источники изменения статуса (журнал EVGA-DB-003).
const (
	ChangeSourceManual = "manual"
	ChangeSourceBulk   = "bulk"
	ChangeSourceSystem = "system"
)

// transitions — матрица разрешённых РУЧНЫХ переходов (ТЗ §7.2, спека 008 FR-3).
// Статус 4 в целях нет: его выставляет только система при регистрации исходящего (BR-012).
var transitions = map[int64][]int64{
	StatusInWork:       {StatusNoViolations},
	StatusNoticeSent:   {StatusWaitingDocs, StatusConfirmed, StatusNoViolations},
	StatusWaitingDocs:  {StatusConfirmed, StatusNoViolations},
	StatusConfirmed:    {StatusRefundDue, StatusAudit},
	StatusRefundDue:    {StatusRefunded, StatusAudit},
	StatusAudit:        {StatusRefunded},
	StatusNoViolations: {}, // терминальный
	StatusRefunded:     {}, // терминальный
}

// TransitionAttrs — атрибуты перехода (спека 008 FR-5/10).
// Денежные значения передаются строками (numeric без потери точности), парсятся при проверке.
type TransitionAttrs struct {
	Note             string
	AmountForVozvrat string // требуется при → 7; ≤ AmountPart
	Refund           string // требуется при → 8; ≥ AmountForVozvrat записи (если та задана)
	ActivityID       *int64 // опционально при → 12 (ответ аналитика 07.09.2026)
}

// Ошибки переходов.
var (
	// ErrTransitionNotAllowed — переход вне матрицы (422 TRANSITION_NOT_ALLOWED).
	ErrTransitionNotAllowed = errors.New("переход не предусмотрен")
	// ErrTransitionCondition — не выполнено условие входа (422 TRANSITION_CONDITION_FAILED).
	ErrTransitionCondition = errors.New("не выполнено условие перехода")
)

// TransitionError — отказ перехода с человекочитаемой причиной (для отчёта bulk, BR-014).
type TransitionError struct {
	Base   error  // ErrTransitionNotAllowed | ErrTransitionCondition
	Field  string // атрибут, которого не хватает (для CONDITION)
	Reason string // текст причины по ТЗ
}

func (e *TransitionError) Error() string { return e.Reason }
func (e *TransitionError) Unwrap() error { return e.Base }

func notAllowed(from, to int64) *TransitionError {
	return &TransitionError{
		Base:   ErrTransitionNotAllowed,
		Reason: fmt.Sprintf("Переход из статуса «%s» в статус «%s» не предусмотрен", statusName(from), statusName(to)),
	}
}

func condition(field, reason string) *TransitionError {
	return &TransitionError{Base: ErrTransitionCondition, Field: field, Reason: reason}
}

// parseAmount — строка → сумма; пустая строка = нет значения.
func parseAmount(s string) (float64, bool, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false, nil
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	if err != nil {
		return 0, false, err
	}
	return v, true, nil
}

// ValidateTransition проверяет ручной переход записи current→target с атрибутами attrs
// против матрицы §7.2 и условий входа (Clarifications спеки 008).
// recordAmountPart / recordAmountForVozvrat — текущие значения записи витрины.
func ValidateTransition(current, target int64, attrs TransitionAttrs, recordAmountPart, recordAmountForVozvrat string) error {
	if current == target {
		return condition("status_id", "Запись уже находится в этом статусе")
	}
	if target == StatusNoticeSent {
		// BR-012: статус 4 вручную не выставляется
		return notAllowed(current, target)
	}
	allowed, ok := transitions[current]
	if !ok {
		return notAllowed(current, target)
	}
	found := false
	for _, t := range allowed {
		if t == target {
			found = true
			break
		}
	}
	if !found {
		return notAllowed(current, target)
	}

	switch target {
	case StatusNoViolations: // → 2: обязателен комментарий (AT-04)
		if strings.TrimSpace(attrs.Note) == "" {
			return condition("risk_status_note", "Для перевода в «Не подтверждено» обязателен комментарий")
		}
	case StatusRefundDue: // → 7: обязательна сумма возмещения ≤ суммы записи
		amount, has, err := parseAmount(attrs.AmountForVozvrat)
		if err != nil || !has || amount <= 0 {
			return condition("amount_for_vozvrat", "Для перевода в «Подлежит возмещению» обязательна сумма к возмещению")
		}
		part, hasPart, err := parseAmount(recordAmountPart)
		if err == nil && hasPart && amount > part {
			return condition("amount_for_vozvrat", "Сумма к возмещению не может превышать сумму записи")
		}
	case StatusAudit:
		// → 12: по ответу аналитика 07.09.2026 мероприятие НЕ обязательно
		// («если ставят Аудит — ничего не происходит»); activity_id принимается опционально.
	case StatusRefunded: // → 8: обязателен refund; если меньше суммы к возмещению — отказ (AT-05)
		refund, has, err := parseAmount(attrs.Refund)
		if err != nil || !has || refund <= 0 {
			return condition("refund", "Для перевода в «Возмещено» обязательна сумма возмещения")
		}
		due, hasDue, err := parseAmount(firstNonEmpty(attrs.AmountForVozvrat, recordAmountForVozvrat))
		if err == nil && hasDue && refund < due {
			return condition("refund", "Сумма возмещения меньше суммы к возмещению — переход отклонён")
		}
	}
	return nil
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
