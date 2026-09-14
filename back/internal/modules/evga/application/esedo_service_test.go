package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"ehd-api/internal/modules/auth/contract"
	"ehd-api/internal/modules/evga/domain"
	"ehd-api/internal/modules/evga/esedo"
	"ehd-api/internal/modules/evga/pdf"
)

type fakeScope struct{ scope domain.DepartmentScope }

func (f fakeScope) Scope(context.Context, contract.Identity) (domain.DepartmentScope, error) {
	return f.scope, nil
}

type fakePDFRepo struct{ data pdf.Data }

func (f fakePDFRepo) PDFData(context.Context, int64, *int64) (pdf.Data, error) { return f.data, nil }

// spySender — записывает вызовы и НЕ ходит в сеть (роль StubSender в тесте оркестрации).
type spySender struct {
	uploaded esedo.Attachment
	sent     esedo.DocOutgoing
	fileID   string
}

func (s *spySender) UploadAttachment(_ context.Context, a esedo.Attachment) (string, error) {
	s.uploaded = a
	return s.fileID, nil
}

func (s *spySender) SendOutgoing(_ context.Context, d esedo.DocOutgoing) (esedo.SendResult, error) {
	s.sent = d
	return esedo.SendResult{MessageID: "SPY", Accepted: true, Note: "spy"}, nil
}

func sampleData() pdf.Data {
	return pdf.Data{
		DocNum:        "05/2026/000583",
		DocDate:       time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC),
		RecipientRU:   "Управление образования Алматинской области",
		RecipientKZ:   "Алматы облысының білім басқармасы",
		RecipientCode: "2295028",
		DeptTitle:     "ДВГА по Алматинской области",
		SignerName:    "Ботанов Нариман",
		ExecName:      "Елдесбаев Дарман Келгенбаевич",
		ExecPhone:     "+7...",
		Rows: []pdf.Row{
			{GU: "101240010295", PpoPp: "1", FM: "Иванов", NM: "Иван", IIN: "900101300000", LA1: "KZ...", AmountPart: "100.00"},
		},
	}
}

func TestESEDOOrchestrationStub(t *testing.T) {
	spy := &spySender{fileID: "HED-FILE-XYZ"}
	svc := NewESEDOOutgoingService(
		fakePDFRepo{sampleData()},
		fakeScope{domain.DepartmentScope{AllDepartments: true}},
		spy, esedo.StubSigner{},
		esedo.Config{FromOrg: "2309569"},
		true, zap.NewNop(),
	)

	res, err := svc.SendToESEDO(context.Background(), contract.Identity{IsAdmin: true}, 42)
	if err != nil {
		t.Fatalf("SendToESEDO: %v", err)
	}
	if !res.Accepted {
		t.Fatalf("ожидали Accepted, got %+v", res)
	}

	// вложение: PDF загружен, имя из номера
	if len(spy.uploaded.Data) == 0 || spy.uploaded.Filename != "uvedomlenie-05-2026-000583.pdf" {
		t.Fatalf("вложение неверно: name=%q bytes=%d", spy.uploaded.Filename, len(spy.uploaded.Data))
	}

	// docOutgoing из данных уведомления
	d := spy.sent
	if d.DocNo != "05/2026/000583" {
		t.Fatalf("DocNo=%q", d.DocNo)
	}
	if len(d.Performers) != 1 || d.Performers[0] != "2295028" {
		t.Fatalf("Performers=%v (ожидали [2295028])", d.Performers)
	}
	if d.DocumentReceiverRu != "Управление образования Алматинской области" {
		t.Fatalf("получатель RU=%q", d.DocumentReceiverRu)
	}
	if d.From != "2309569" || d.SenderOrg != "2309569" {
		t.Fatalf("from/senderOrg=%q/%q", d.From, d.SenderOrg)
	}
	if len(d.Attachments) != 1 || d.Attachments[0].FileIdentifier != "HED-FILE-XYZ" {
		t.Fatalf("вложение в docOutgoing=%v", d.Attachments)
	}
	if !d.SecondSignEnabled || d.SecondSignData == "" {
		t.Fatalf("подпись не проставлена: enabled=%v data=%q", d.SecondSignEnabled, d.SecondSignData)
	}
}

func TestESEDOOrchestrationReadOnly(t *testing.T) {
	svc := NewESEDOOutgoingService(
		fakePDFRepo{sampleData()},
		fakeScope{domain.DepartmentScope{AllDepartments: true}},
		&spySender{}, esedo.StubSigner{}, esedo.Config{}, false, zap.NewNop(),
	)
	if _, err := svc.SendToESEDO(context.Background(), contract.Identity{IsAdmin: true}, 42); !errors.Is(err, ErrReadOnlyMode) {
		t.Fatalf("ожидали ErrReadOnlyMode, got %v", err)
	}
}
