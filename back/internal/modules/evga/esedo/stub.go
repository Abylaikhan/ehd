package esedo

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// StubSender — безопасная заглушка (реализация по умолчанию): НИЧЕГО не шлёт по сети,
// логирует намерение и возвращает детерминированный результат. Позволяет прогонять
// сценарий «создать→отправить» локально и на dev-реплике без доступа к ШЭП.
type StubSender struct{ log *zap.Logger }

// NewStubSender — заглушка отправителя. log может быть nil (тогда логи глушатся).
func NewStubSender(log *zap.Logger) *StubSender {
	if log == nil {
		log = zap.NewNop()
	}
	return &StubSender{log: log}
}

func (s *StubSender) UploadAttachment(_ context.Context, a Attachment) (string, error) {
	fileID := "STUB-FILE-" + a.Filename
	s.log.Info("esedo(stub): вложение НЕ загружено (заглушка)",
		zap.String("file", a.Filename), zap.Int("bytes", len(a.Data)), zap.String("file_id", fileID))
	return fileID, nil
}

func (s *StubSender) SendOutgoing(_ context.Context, doc DocOutgoing) (SendResult, error) {
	s.log.Info("esedo(stub): docOutgoing НЕ отправлен (заглушка)",
		zap.String("doc_no", doc.DocNo),
		zap.Strings("performers", doc.Performers),
		zap.Int("attachments", len(doc.Attachments)),
		zap.Bool("second_sign", doc.SecondSignEnabled))
	return SendResult{
		MessageID: "STUB-MSG-" + doc.Href,
		Accepted:  true,
		Note:      "stub: not sent to ESEDO",
	}, nil
}

// StubSigner — заглушка подписи: возвращает псевдо-CMS, по сети не ходит.
type StubSigner struct{}

func (StubSigner) SignCMS(_ context.Context, data []byte) ([]byte, error) {
	return []byte(fmt.Sprintf("STUB-CMS(%d)", len(data))), nil
}
