package domain

import (
	"testing"
	"time"
)

func d(y int, m time.Month, day int) time.Time {
	return time.Date(y, m, day, 12, 0, 0, 0, time.UTC)
}

func TestAddBusinessDays(t *testing.T) {
	cases := []struct {
		from time.Time
		n    int
		want time.Time
	}{
		// понедельник + 10 будних = понедельник через 2 недели
		{d(2026, time.September, 7), 10, d(2026, time.September, 21)},
		// пятница + 1 будний = понедельник
		{d(2026, time.September, 11), 1, d(2026, time.September, 14)},
		// суббота + 1 будний = понедельник
		{d(2026, time.September, 12), 1, d(2026, time.September, 14)},
		// среда + 3 будних: чт, пт, пн
		{d(2026, time.September, 9), 3, d(2026, time.September, 14)},
		// ноль дней — та же дата
		{d(2026, time.September, 9), 0, d(2026, time.September, 9)},
	}
	for i, c := range cases {
		got := AddBusinessDays(c.from, c.n)
		if !got.Equal(c.want) {
			t.Errorf("case %d: %v + %d = %v, want %v", i, c.from.Format("Mon 02.01"), c.n, got.Format("Mon 02.01"), c.want.Format("Mon 02.01"))
		}
	}
}
