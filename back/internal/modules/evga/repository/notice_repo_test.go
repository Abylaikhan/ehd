package repository

import (
	"testing"

	"ehd-api/internal/modules/evga/domain"
)

func s1() *int64 { v := domain.StatusInWork; return &v }

func TestClassifyReasons(t *testing.T) {
	busyNotice := int64(77)
	busyNum := "11/2026/000777"
	rows := []previewRow{
		{ID: 1, GU: "111", StatusID: s1(), AmountPart: "10.00"},                           // валидная
		{ID: 2, GU: "111", StatusID: func() *int64 { v := int64(4); return &v }()},        // статус ≠ 1
		{ID: 3, GU: "", StatusID: s1()},                                                   // пустой ГУ
		{ID: 4, GU: "222", StatusID: s1(), BusyNotice: &busyNotice, BusyDocnum: &busyNum}, // занята
		{ID: 6, GU: "222", StatusID: nil},                                                 // статус NULL
	}
	ids := []int64{1, 2, 3, 4, 5, 6} // 5 — отсутствует в выборке

	valid, rejected := classify(ids, rows)
	if len(valid) != 1 || valid[0].ID != 1 {
		t.Fatalf("valid: %+v", valid)
	}
	byID := map[int64]domain.RejectedRecord{}
	for _, r := range rejected {
		byID[r.ID] = r
	}
	if byID[2].Code != domain.RejectBadStatus {
		t.Errorf("id2: %+v", byID[2])
	}
	if byID[3].Code != domain.RejectEmptyGU {
		t.Errorf("id3: %+v", byID[3])
	}
	if byID[4].Code != domain.RejectInNotice || byID[4].NoticeNum != busyNum {
		t.Errorf("id4: %+v", byID[4])
	}
	if byID[5].Code != domain.RejectNotFound {
		t.Errorf("id5: %+v", byID[5])
	}
	if byID[6].Code != domain.RejectBadStatus {
		t.Errorf("id6: %+v", byID[6])
	}
}

func TestGroupByGU(t *testing.T) {
	rows := []previewRow{
		{ID: 1, GU: "111", GUBIN: "b1", SenderName: "Школа №1", StatusID: s1(), AmountPart: "100.50", FM: "А", NM: "Б"},
		{ID: 2, GU: "222", SenderName: "Больница", StatusID: s1(), AmountPart: "1.00"},
		{ID: 3, GU: "111", StatusID: s1(), AmountPart: "0.50"},
	}
	groups := groupByGU(rows)
	if len(groups) != 2 {
		t.Fatalf("groups: %d", len(groups))
	}
	// порядок появления сохраняется; каждая группа = отдельное уведомление (EVGA-FR-031)
	g := groups[0]
	if g.GU != "111" || g.Count != 2 || g.TotalSum != "101.00" || g.SenderName != "Школа №1" || g.GUBIN != "b1" {
		t.Fatalf("g111: %+v", g)
	}
	if g.Records[0].FIO != "А Б" {
		t.Fatalf("fio: %q", g.Records[0].FIO)
	}
	if groups[1].GU != "222" || groups[1].Count != 1 {
		t.Fatalf("g222: %+v", groups[1])
	}
}
