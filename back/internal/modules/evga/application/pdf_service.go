package application

import (
	"context"
	"io"

	"ehd-api/internal/modules/auth/contract"
	"ehd-api/internal/modules/evga/pdf"
)

// PDFRepoPort — сбор данных документа (спека 012 FR-3).
type PDFRepoPort interface {
	PDFData(ctx context.Context, noticeID int64, deptID *int64) (pdf.Data, error)
}

// PDFService — генерация PDF уведомления по образцу (спека 012).
type PDFService struct {
	repo   PDFRepoPort
	scopes ScopeProvider
}

func NewPDFService(repo PDFRepoPort, scopes ScopeProvider) *PDFService {
	return &PDFService{repo: repo, scopes: scopes}
}

// Render — PDF уведомления в пределах видимости пользователя (FR-1).
// Возвращает номер документа для имени файла.
func (s *PDFService) Render(ctx context.Context, id contract.Identity, noticeID int64, w io.Writer) (string, error) {
	scope, err := s.scopes.Scope(ctx, id)
	if err != nil {
		return "", err
	}
	data, err := s.repo.PDFData(ctx, noticeID, deptFilter(scope))
	if err != nil {
		return "", err
	}
	return data.DocNum, pdf.Render(w, data)
}
