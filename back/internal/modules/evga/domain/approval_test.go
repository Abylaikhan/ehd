package domain

import (
	"errors"
	"testing"
)

func TestFormatNoticeNumber(t *testing.T) {
	got, err := FormatNoticeNumber("11", 2026, 1097)
	if err != nil || got != "11/2026/001097" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := FormatNoticeNumber("  ", 2026, 1); !errors.Is(err, ErrNumberingUnavailable) {
		t.Fatalf("empty region: %v", err)
	}
	if _, err := FormatNoticeNumber("11", 2026, 0); err == nil {
		t.Fatal("seq 0 must fail")
	}
	if _, err := FormatNoticeNumber("11", 2026, 1000000); err == nil {
		t.Fatal("seq overflow must fail")
	}
}

func TestNextNoticeSeqYearReset(t *testing.T) {
	y2025 := int32(2025)
	y2026 := int32(2026)
	// первый номер вообще (год не заполнен) — AT-11
	if got := NextNoticeSeq(541, nil, 2026); got != 1 {
		t.Fatalf("nil year: %d", got)
	}
	// сброс при смене года — AT-11
	if got := NextNoticeSeq(541, &y2025, 2026); got != 1 {
		t.Fatalf("year reset: %d", got)
	}
	// обычный инкремент — AT-10
	if got := NextNoticeSeq(541, &y2026, 2026); got != 542 {
		t.Fatalf("increment: %d", got)
	}
}

func TestRouteInputValidate(t *testing.T) {
	ok := RouteInput{ApproverIDs: []int64{5, 7}, OutgoingUserID: 3}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	cases := []RouteInput{
		{OutgoingUserID: 3},                             // нет согласующих (AT-09)
		{ApproverIDs: []int64{5}},                       // нет ответственного (AT-09)
		{ApproverIDs: []int64{5, 5}, OutgoingUserID: 3}, // дубль согласующего
		{ApproverIDs: []int64{0}, OutgoingUserID: 3},    // некорректный id
	}
	for i, c := range cases {
		if err := c.Validate(); !errors.Is(err, ErrRouteIncomplete) {
			t.Errorf("case %d: want ErrRouteIncomplete, got %v", i, err)
		}
	}
}
