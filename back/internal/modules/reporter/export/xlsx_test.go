package export

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestWriteXLSX_HeadersAndRows(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"ID", "ФИО", "Сумма"}
	rows := [][]any{
		{uint64(1), "Клиент 1", "10.50"},
		{uint64(2), "Клиент 2", "20.00"},
	}
	if err := WriteXLSX(&buf, "Данные", headers, rows); err != nil {
		t.Fatalf("WriteXLSX: %v", err)
	}

	f, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatalf("невалидный XLSX: %v", err)
	}
	defer f.Close()

	got, err := f.GetRows("Данные")
	if err != nil {
		t.Fatalf("GetRows: %v", err)
	}
	if len(got) != 3 { // заголовок + 2 строки
		t.Fatalf("ожидалось 3 строки, получено %d", len(got))
	}
	if got[0][0] != "ID" || got[0][1] != "ФИО" || got[0][2] != "Сумма" {
		t.Errorf("заголовки должны быть label: %v", got[0])
	}
	if got[1][0] != "1" || got[1][1] != "Клиент 1" {
		t.Errorf("данные строки 1: %v", got[1])
	}
}
