package pdf

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRenderProducesValidPDF(t *testing.T) {
	d1 := time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC)
	rows := make([]Row, 0, 14)
	for i := 0; i < 14; i++ { // >1 страницы таблицы — проверка переноса с повтором шапки
		rows = append(rows, Row{
			GU: "2617832", PpoPp: "488727711-2422915061", PaymentDate: &d1,
			FM: "ӘБДІҚАРІМОВА", NM: "ҰЛБОЛҒАН", FT: "ҚАЙЫРЖАНҚЫЗЫ", // казахские глифы (приёмка 2)
			IIN: "010928600155", LA1: "KZ916010002047787804", AmountPart: "70000.00",
		})
	}
	data := Data{
		DocNum: "11/2026/001097", DocDate: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		RecipientRU:    "Коммунальное государственное учреждение «Управление образования Кызылординской области»",
		RecipientKZ:    "«Қызылорда облысының білім басқармасы» коммуналдық мемлекеттік мекемесі",
		RecipientCode:  "2294440",
		DeptTitle:      "ДВГА по Кызылординской области КВГА МФ РК",
		SignerPosition: "Руководитель ДВГА", SignerPosKZ: "ІМАД басшысы", SignerName: "Кузембаев Ербол",
		ExecName: "Қарсақбаев Нұрлан Әнуарбекұлы", ExecPhone: "87242233809", ExecEmail: "n.qarsaqbaev@minfin.gov.kz",
		Rows: rows,
	}

	var buf bytes.Buffer
	if err := Render(&buf, data); err != nil {
		t.Fatalf("Render: %v", err)
	}
	out := buf.Bytes()
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Fatalf("не PDF: %q", out[:8])
	}
	if len(out) < 100_000 {
		t.Fatalf("подозрительно маленький PDF: %d байт (шрифты/картинки не вшились?)", len(out))
	}
	// 4 секции + перенос таблиц → минимум 6 страниц
	if pages := bytes.Count(out, []byte("/Type /Page")) - bytes.Count(out, []byte("/Type /Pages")); pages < 6 {
		t.Fatalf("страниц %d, ожидали ≥6", pages)
	}
}

func TestRenderDraftWithoutSigner(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, Data{
		DocNum: "", DocDate: time.Now(),
		RecipientRU: "Тест", RecipientKZ: "Тест", DeptTitle: "ДВГА",
		ExecName: "Исп", ExecPhone: "1", ExecEmail: "a@b",
		Rows: []Row{{GU: "1", PpoPp: "x", AmountPart: "1.00"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(buf.String(), "%PDF-") {
		t.Fatal("не PDF")
	}
}
