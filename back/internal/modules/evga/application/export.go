package application

import (
	"context"
	"io"

	"ehd-api/internal/modules/auth/contract"
	"ehd-api/internal/modules/evga/domain"
	"ehd-api/pkg/xlsx"
)

// exportHeaders — колонки выгрузки = колонкам грида (spec FR-11, EVGA-FR-012/017).
var exportHeaders = []string{
	"№", "Профиль риска", "Код ГУ отправителя", "Наименование отправителя",
	"Номер платежа", "Дата платежа", "ИИН получателя", "Фамилия", "Имя", "Отчество",
	"Карт-счет", "Сумма", "Статус", "Комментарий", "Уведомление", "Исходящее письмо",
}

// ExportResult — результат экспорта.
type ExportResult struct {
	Truncated bool
	Rows      int
}

// Export пишет XLSX той же выборки, что и реестр; лимит domain.ExportLimit строк (FR-11).
func (s *Service) Export(ctx context.Context, id contract.Identity, f domain.Filter, sort, order string, w io.Writer) (ExportResult, error) {
	scope, err := s.Scope(ctx, id)
	if err != nil {
		return ExportResult{}, err
	}
	var items []domain.RiskRecord
	if !scope.Unmapped {
		if !scope.AllDepartments {
			f.DepartmentID = nil
		}
		// limit+1 — чтобы отличить «ровно лимит» от «обрезано»
		items, err = s.repo.ListAll(ctx, f, deptFilter(scope), sort, order, domain.ExportLimit+1)
		if err != nil {
			return ExportResult{}, s.wrapSourceErr(ctx, err)
		}
	}
	res := ExportResult{}
	if len(items) > domain.ExportLimit {
		items = items[:domain.ExportLimit]
		res.Truncated = true
	}
	res.Rows = len(items)

	rows := make([][]any, len(items))
	for i, r := range items {
		date := ""
		if r.PaymentDate != nil {
			date = r.PaymentDate.Format("02.01.2006")
		}
		notice, out := r.NoticeNum, r.OutNum
		if notice == "" {
			notice = "—"
		}
		if out == "" {
			out = "—"
		}
		rows[i] = []any{
			i + 1, r.ProfileTitle, r.GU, r.SenderName,
			r.PpoPp, date, r.IIN, r.FM, r.NM, r.FT,
			r.LA1, r.AmountPart, r.StatusTitle, r.StatusNote, notice, out,
		}
	}
	if err := xlsx.Write(w, "Реестр рисков 5-15а", exportHeaders, rows); err != nil {
		return ExportResult{}, err
	}
	return res, nil
}
