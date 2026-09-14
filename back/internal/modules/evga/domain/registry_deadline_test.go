package domain

import (
	"testing"
	"time"
)

func ptrI64(v int64) *int64 { return &v }

func TestDeadlineState(t *testing.T) {
	// фиксированное «сегодня» = среда 2026-09-09; окно 3 рабочих дня → порог пн 2026-09-14
	// (пропускаются сб 12 и вс 13).
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
		{"выходные внутри окна", "2026-09-12", ptrI64(StatusNoticeSent), DeadlineExpiring},
		{"граница окна — 3 рабочих дня (пн)", "2026-09-14", ptrI64(StatusWaitingDocs), DeadlineExpiring},
		{"сразу за окном (вт)", "2026-09-15", ptrI64(StatusNoticeSent), DeadlineNone},
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
	now := time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC) // среда
	// warnDays<=0 → дефолт (3 рабочих дня): порог пн 2026-09-14 = expiring, вт 09-15 = none
	if got := DeadlineState("2026-09-14", ptrI64(StatusNoticeSent), 0, now); got != DeadlineExpiring {
		t.Fatalf("дефолтный порог: 3 раб. дня (пн) = %q, ожидалось %q", got, DeadlineExpiring)
	}
	if got := DeadlineState("2026-09-15", ptrI64(StatusNoticeSent), 0, now); got != DeadlineNone {
		t.Fatalf("дефолтный порог: за окном (вт) = %q, ожидалось %q", got, DeadlineNone)
	}
}
