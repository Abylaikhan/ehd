// Package xlsx — потоковая генерация XLSX (общая для модулей).
// Копия internal/modules/reporter/export (перевод reporter на этот пакет — техдолг).
package xlsx

import (
	"io"

	"github.com/xuri/excelize/v2"
)

// Write формирует XLSX с одним листом: первая строка — заголовки, далее — данные.
// StreamWriter ограничивает память при большом числе строк.
func Write(w io.Writer, sheet string, headers []string, rows [][]any) error {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return err
	}
	sw, err := f.NewStreamWriter(sheet)
	if err != nil {
		return err
	}

	hdr := make([]any, len(headers))
	for i, h := range headers {
		hdr[i] = excelize.Cell{Value: h}
	}
	if err := sw.SetRow("A1", hdr); err != nil {
		return err
	}
	for i, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, i+2)
		if err != nil {
			return err
		}
		if err := sw.SetRow(cell, row); err != nil {
			return err
		}
	}
	if err := sw.Flush(); err != nil {
		return err
	}
	_, err = f.WriteTo(w)
	return err
}
