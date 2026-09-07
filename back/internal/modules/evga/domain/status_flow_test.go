package domain

import (
	"errors"
	"strings"
	"testing"
)

// полный перебор матрицы: явный список разрешённых ручных переходов; всё прочее — отказ.
func TestTransitionMatrixExhaustive(t *testing.T) {
	statuses := []int64{1, 2, 4, 6, 7, 8, 11, 12}
	allowed := map[[2]int64]bool{
		{1, 2}: true,
		{4, 6}: true, {4, 11}: true, {4, 2}: true,
		{6, 11}: true, {6, 2}: true,
		{11, 7}: true, {11, 12}: true,
		{7, 8}: true, {7, 12}: true,
		{12, 8}: true,
	}
	// валидные атрибуты, чтобы условия входа не мешали проверке самой матрицы
	attrs := TransitionAttrs{Note: "причина", AmountForVozvrat: "100.00", Refund: "100.00", ActivityID: ptr(int64(5))}

	for _, from := range statuses {
		for _, to := range statuses {
			if from == to {
				continue
			}
			err := ValidateTransition(from, to, attrs, "500.00", "100.00")
			if allowed[[2]int64{from, to}] {
				if err != nil {
					t.Errorf("%d→%d: ожидали успех, получили %v", from, to, err)
				}
			} else if !errors.Is(err, ErrTransitionNotAllowed) {
				t.Errorf("%d→%d: ожидали ErrTransitionNotAllowed, получили %v", from, to, err)
			}
		}
	}
}

func ptr[T any](v T) *T { return &v }

func TestManualStatus4Forbidden(t *testing.T) {
	// AT-03-подобное: 4 вручную не выставляется ни из одного статуса (BR-012)
	for _, from := range []int64{1, 2, 6, 7, 8, 11, 12} {
		if err := ValidateTransition(from, StatusNoticeSent, TransitionAttrs{}, "", ""); !errors.Is(err, ErrTransitionNotAllowed) {
			t.Errorf("%d→4: ожидали ErrTransitionNotAllowed, получили %v", from, err)
		}
	}
}

func TestBackToInWorkForbidden(t *testing.T) {
	// AT-03 / BR-013: из 4 и 6 возврат в 1 невозможен, с сообщением по BR-014
	err := ValidateTransition(4, 1, TransitionAttrs{}, "", "")
	if !errors.Is(err, ErrTransitionNotAllowed) {
		t.Fatalf("4→1: %v", err)
	}
	var te *TransitionError
	if !errors.As(err, &te) || te.Reason != "Переход из статуса 4 в статус 1 не предусмотрен" {
		t.Fatalf("текст причины: %q", err.Error())
	}
	if err := ValidateTransition(6, 1, TransitionAttrs{}, "", ""); !errors.Is(err, ErrTransitionNotAllowed) {
		t.Fatalf("6→1: %v", err)
	}
}

func TestConditionNoteRequired(t *testing.T) {
	// AT-04: → 2 без комментария блокируется, с ним — проходит
	err := ValidateTransition(1, 2, TransitionAttrs{}, "", "")
	if !errors.Is(err, ErrTransitionCondition) {
		t.Fatalf("1→2 без note: %v", err)
	}
	var te *TransitionError
	if !errors.As(err, &te) || te.Field != "risk_status_note" {
		t.Fatalf("field: %+v", te)
	}
	if err := ValidateTransition(1, 2, TransitionAttrs{Note: "ложное срабатывание"}, "", ""); err != nil {
		t.Fatalf("1→2 с note: %v", err)
	}
}

func TestConditionAmountForVozvrat(t *testing.T) {
	// → 7: сумма обязательна, > 0 и ≤ amount_part
	if err := ValidateTransition(11, 7, TransitionAttrs{}, "500.00", ""); !errors.Is(err, ErrTransitionCondition) {
		t.Fatalf("без суммы: %v", err)
	}
	if err := ValidateTransition(11, 7, TransitionAttrs{AmountForVozvrat: "600.00"}, "500.00", ""); !errors.Is(err, ErrTransitionCondition) {
		t.Fatalf("сумма больше amount_part: %v", err)
	}
	if err := ValidateTransition(11, 7, TransitionAttrs{AmountForVozvrat: "500.00"}, "500.00", ""); err != nil {
		t.Fatalf("валидная сумма: %v", err)
	}
}

func TestAuditNoConditions(t *testing.T) {
	// ответ аналитика 07.09.2026: переход в «Аудит» не требует атрибутов
	if err := ValidateTransition(11, 12, TransitionAttrs{}, "", ""); err != nil {
		t.Fatalf("11→12 без атрибутов должен проходить: %v", err)
	}
	// мероприятие опционально принимается
	if err := ValidateTransition(7, 12, TransitionAttrs{ActivityID: ptr(int64(3))}, "", "100.00"); err != nil {
		t.Fatalf("с мероприятием: %v", err)
	}
}

func TestConditionRefund(t *testing.T) {
	// AT-05: refund < amount_for_vozvrat → отказ; refund ≥ — успех; refund обязателен
	if err := ValidateTransition(7, 8, TransitionAttrs{}, "", "300.00"); !errors.Is(err, ErrTransitionCondition) {
		t.Fatalf("без refund: %v", err)
	}
	err := ValidateTransition(7, 8, TransitionAttrs{Refund: "200.00"}, "", "300.00")
	if !errors.Is(err, ErrTransitionCondition) || !strings.Contains(err.Error(), "меньше суммы к возмещению") {
		t.Fatalf("refund < due: %v", err)
	}
	if err := ValidateTransition(7, 8, TransitionAttrs{Refund: "300.00"}, "", "300.00"); err != nil {
		t.Fatalf("refund == due: %v", err)
	}
	// 12→8: amount_for_vozvrat у записи может отсутствовать — достаточно refund
	if err := ValidateTransition(12, 8, TransitionAttrs{Refund: "50.00"}, "", ""); err != nil {
		t.Fatalf("12→8: %v", err)
	}
}

func TestSameStatusRejected(t *testing.T) {
	if err := ValidateTransition(6, 6, TransitionAttrs{}, "", ""); !errors.Is(err, ErrTransitionCondition) {
		t.Fatalf("same status: %v", err)
	}
}
