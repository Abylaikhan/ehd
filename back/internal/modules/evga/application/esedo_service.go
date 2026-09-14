package application

import (
	"bytes"
	"context"
	"encoding/base64"
	"strings"
	"time"

	"go.uber.org/zap"

	"ehd-api/internal/modules/auth/contract"
	"ehd-api/internal/modules/evga/domain"
	"ehd-api/internal/modules/evga/esedo"
	"ehd-api/internal/modules/evga/pdf"
)

// scopeReader — минимальный порт видимости (реализуется *Service.Scope).
type scopeReader interface {
	Scope(ctx context.Context, id contract.Identity) (domain.DepartmentScope, error)
}

// ESEDOOutgoingService — оркестрация отправки исходящего в ЕСЭДО (спека 014, вариант B:
// «отправляем мы»). Пайплайн: PDF → загрузка в EDS_TEMP_FILES → docOutgoing → подпись →
// SendMessage. По умолчанию собран на StubSender/StubSigner — реально в ЕСЭДО НИЧЕГО не уходит.
// Реальный клиент (SoapSender) даже при включении возвращает ErrTransportDisabled до активации.
type ESEDOOutgoingService struct {
	data         PDFRepoPort
	scopes       scopeReader
	sender       esedo.Sender
	signer       esedo.Signer
	cfg          esedo.Config
	writeEnabled bool
	nowFn        func() time.Time
	log          *zap.Logger
}

func NewESEDOOutgoingService(data PDFRepoPort, scopes scopeReader, sender esedo.Sender, signer esedo.Signer, cfg esedo.Config, writeEnabled bool, log *zap.Logger) *ESEDOOutgoingService {
	return &ESEDOOutgoingService{
		data: data, scopes: scopes, sender: sender, signer: signer,
		cfg: cfg, writeEnabled: writeEnabled, nowFn: time.Now, log: log,
	}
}

// SendToESEDO собирает и «отправляет» исходящее уведомление (на заглушке — без сети).
// Шаги 1–5 из спеки 014 §5.0; реальные транспорт/подпись подключаются при активации.
func (s *ESEDOOutgoingService) SendToESEDO(ctx context.Context, id contract.Identity, noticeID int64) (esedo.SendResult, error) {
	if !s.writeEnabled {
		return esedo.SendResult{}, ErrReadOnlyMode
	}
	scope, err := s.scopes.Scope(ctx, id)
	if err != nil {
		return esedo.SendResult{}, err
	}
	if scope.Unmapped && !id.IsAdmin {
		return esedo.SendResult{}, domain.ErrNotFound
	}
	data, err := s.data.PDFData(ctx, noticeID, deptFilter(scope))
	if err != nil {
		return esedo.SendResult{}, err
	}

	// 1) PDF уведомления (уже умеем)
	var buf bytes.Buffer
	if err := pdf.Render(&buf, data); err != nil {
		return esedo.SendResult{}, err
	}

	// 2) загрузка вложения в EDS_TEMP_FILES → fileIdentifier
	fileName := "uvedomlenie-" + strings.ReplaceAll(data.DocNum, "/", "-") + ".pdf"
	fileID, err := s.sender.UploadAttachment(ctx, esedo.Attachment{
		Filename: fileName, ContentType: "application/pdf", Data: buf.Bytes(),
	})
	if err != nil {
		return esedo.SendResult{}, err
	}

	// 3) сборка docOutgoing из данных уведомления
	doc := s.buildDoc(data, fileID)

	// 4) подпись документа (secondSignData) через Signer (заглушка — псевдо-CMS)
	cms, err := s.signer.SignCMS(ctx, buf.Bytes())
	if err != nil {
		return esedo.SendResult{}, err
	}
	doc.SecondSignEnabled = true
	doc.SecondSignData = base64.StdEncoding.EncodeToString(cms)

	// 5) отправка (на заглушке — без сети; реальный клиент до активации → ErrTransportDisabled)
	res, err := s.sender.SendOutgoing(ctx, doc)
	if err != nil {
		return res, err
	}
	s.log.Info("evga: esedo orchestration (заглушка)",
		zap.Int64("notice_id", noticeID), zap.String("doc_no", doc.DocNo),
		zap.String("file_id", fileID), zap.Int("attachments", len(doc.Attachments)),
		zap.Bool("accepted", res.Accepted), zap.String("note", res.Note))
	return res, nil
}

// buildDoc маппит данные уведомления в docOutgoing. Поля, которых пока нет со стороны
// заказчика (коды НСИ docKind/character, коды организаций from/senderOrg, href, sectionUUID),
// берутся из конфигурации или пусты — заполнить при активации (spec 014 §5, коды организаций).
func (s *ESEDOOutgoingService) buildDoc(d pdf.Data, fileID string) esedo.DocOutgoing {
	var performers []string
	if code := strings.TrimSpace(d.RecipientCode); code != "" {
		performers = []string{code} // ID участника ЕСЭДО = its_cli.code
	}
	return esedo.DocOutgoing{
		DocNo:              d.DocNum,
		DocDate:            d.DocDate.Format(time.RFC3339),
		OutTime:            s.nowFn().Format(time.RFC3339),
		DocLang:            "ru",
		Description:        "Уведомление об устранении нарушения",
		DocumentReceiverRu: d.RecipientRU,
		DocumentReceiverKz: d.RecipientKZ,
		Performers:         performers,
		SignerNameRu:       d.SignerName,
		SignerNameKz:       d.SignerName,
		AuthorNameRu:       d.SignerName,
		AuthorNameKz:       d.SignerName,
		Executor:           d.ExecName,
		EmployeePhone:      d.ExecPhone,
		From:               s.cfg.FromOrg,
		SenderOrg:          s.cfg.FromOrg,
		Attachments:        []esedo.FileRef{{FileIdentifier: fileID}},
		// TODO(активация): Href, SectionUUID, DocKind, Character (коды НСИ), SheetCount —
		// от заказчика/платформы.
	}
}
