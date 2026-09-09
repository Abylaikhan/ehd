package domain

import (
	"testing"
	"time"
)

func ptrI64(v int64) *int64 { return &v }

func TestDeadlineState(t *testing.T) {
	// фиксированное «сегодня» для детерминизма
	now := time.Date(2026, 9, 9, 14, 0, 0, 0, time.UTC)
	warn := 3

	cases := []struct {
		name    string
		execDue string
		status  *int64
		want    string
	}{
		{"истёк вчера", "2026-09-08", ptrI64(StatusNoticeSent), DeadlineExpired},
		{"истекает сегодня", "2026-09-09", ptrI64(StatusNoticeSent), DeadlineExpiring},
		{"истекает в пределах порога", "2026-09-12", ptrI64(StatusNoticeSent), DeadlineExpiring},
		{"на границе порога", "2026-09-12", ptrI64(StatusWaitingDocs), DeadlineExpiring},
		{"за порогом", "2026-09-13", ptrI64(StatusNoticeSent), DeadlineNone},
		{"далеко", "2026-10-30", ptrI64(StatusNoticeSent), DeadlineNone},
		{"статус 6 тоже считается", "2026-09-08", ptrI64(StatusWaitingDocs), DeadlineExpired},
		{"решённый статус 11 — не считается", "2026-09-08", ptrI64(StatusConfirmed), DeadlineNone},
		{"стартовый статус 1 — не считается", "2026-09-08", ptrI64(StatusInWork), DeadlineNone},
		{"нет статуса", "2026-09-08", nil, DeadlineNone},
		{"пустой срок", "", ptrI64(StatusNoticeSent), DeadlineNone},
		{"мусор в сроке", "не-дата", ptrI64(StatusNoticeSent), DeadlineNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := DeadlineState(c.execDue, c.status, warn, now)
			if got != c.want {
				t.Fatalf("DeadlineState(%q, %v) = %q, ожидалось %q", c.execDue, c.status, got, c.want)
			}
		})
	}
}

func TestDeadlineStateDefaultWarn(t *testing.T) {
	now := time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC)
	// warnDays<=0 → дефолт (3): сегодня+3 = expiring, +4 = none
	if got := DeadlineState("2026-09-12", ptrI64(StatusNoticeSent), 0, now); got != DeadlineExpiring {
		t.Fatalf("дефолтный порог: +3 дня = %q, ожидалось %q", got, DeadlineExpiring)
	}
	if got := DeadlineState("2026-09-13", ptrI64(StatusNoticeSent), 0, now); got != DeadlineNone {
		t.Fatalf("дефолтный порог: +4 дня = %q, ожидалось %q", got, DeadlineNone)
	}
}
