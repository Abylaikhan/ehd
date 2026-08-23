package application

import (
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"ehd-api/internal/modules/reporter/domain"
)

// TestExport_Busy: занятый семафор → ErrExportBusy (проверка до обращения к зависимостям).
func TestExport_Busy(t *testing.T) {
	s := NewQueryService(nil, nil, zap.NewNop())
	s.exportSem <- struct{}{} // экспорт уже «выполняется»
	if _, err := s.Export(ctx, Requester{}, "demo", domain.QuerySpec{}); !errors.Is(err, domain.ErrExportBusy) {
		t.Fatalf("ожидалась ErrExportBusy, получено %v", err)
	}
}

type stringerT struct{ s string }

func (x stringerT) String() string { return x.s }

func TestCellValue(t *testing.T) {
	s := "текст"
	var nilPtr *string
	tm := time.Date(2026, 3, 23, 16, 37, 13, 0, time.UTC)

	cases := []struct {
		in   any
		want any
	}{
		{nil, ""},
		{nilPtr, ""},
		{&s, "текст"},
		{"строка", "строка"},
		{uint64(42), uint64(42)},
		{true, true},
		{stringerT{"9.99"}, "9.99"}, // decimal и т.п.
		{tm, tm},
	}
	for _, c := range cases {
		if got := cellValue(c.in); got != c.want {
			t.Errorf("cellValue(%v) = %v (%T), want %v", c.in, got, got, c.want)
		}
	}
}
