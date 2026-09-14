package esedo

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// sampleDoc — значения из примера вызова аналитика (voprosy-analitiku-4.md, 10.09.2026).
func sampleDoc() DocOutgoing {
	return DocOutgoing{
		Href:               "44acb82b-d546-4f3c-a793-ff10053723ae",
		SectionUUID:        "8d3fe9e2-efb3-4d48-aab1-943d2777fc7e",
		From:               "17481752",
		SenderOrg:          "2309569",
		Performers:         []string{"2301015"},
		DocNo:              "38219",
		DocDate:            "2026-05-04T17:33:03.294+05:00",
		OutTime:            "2025-09-25T21:14:49.000Z",
		DocLang:            "ru",
		DocKind:            "12560979",
		Character:          "12850581",
		Description:        "Тестовое письмо",
		Executor:           "Проектный офис",
		AuthorNameKz:       "Проектный офис",
		AuthorNameRu:       "Проектный офис",
		SignerNameKz:       "Проектный офис",
		SignerNameRu:       "Проектный офис",
		DocumentReceiverRu: "Акционерное общество «Центр электронных финансов»",
		DocumentReceiverKz: "«Электрондық қаржы орталығы» акционерлік қоғамы",
		Attachments:        []FileRef{{FileIdentifier: "3828910d-c197-4cee-bb1c-3537080b44ea"}},
	}
}

func TestStubSenderNoNetwork(t *testing.T) {
	s := NewStubSender(nil)
	fileID, err := s.UploadAttachment(context.Background(), Attachment{Filename: "u.pdf", Data: []byte("pdf")})
	if err != nil || fileID == "" {
		t.Fatalf("stub UploadAttachment: fileID=%q err=%v", fileID, err)
	}
	res, err := s.SendOutgoing(context.Background(), sampleDoc())
	if err != nil {
		t.Fatalf("stub SendOutgoing err=%v", err)
	}
	if !res.Accepted || !strings.HasPrefix(res.MessageID, "STUB-MSG-") {
		t.Fatalf("stub SendOutgoing result=%+v", res)
	}
}

func TestStubSigner(t *testing.T) {
	cms, err := StubSigner{}.SignCMS(context.Background(), []byte("data"))
	if err != nil || len(cms) == 0 {
		t.Fatalf("stub sign: cms=%q err=%v", cms, err)
	}
}

func TestSoapDisabledByDefault(t *testing.T) {
	s := NewSoapSender(Config{}, StubSigner{}, nil) // Enabled=false
	if _, err := s.SendOutgoing(context.Background(), sampleDoc()); !errors.Is(err, ErrDisabled) {
		t.Fatalf("ожидали ErrDisabled, получили %v", err)
	}
	if _, err := s.UploadAttachment(context.Background(), Attachment{Filename: "u.pdf"}); !errors.Is(err, ErrDisabled) {
		t.Fatalf("ожидали ErrDisabled для upload, получили %v", err)
	}
}

func TestSoapNotConfigured(t *testing.T) {
	s := NewSoapSender(Config{Enabled: true}, StubSigner{}, nil) // нет endpoint/senderId
	if _, err := s.SendOutgoing(context.Background(), sampleDoc()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("ожидали ErrNotConfigured, получили %v", err)
	}
}

// Даже полностью «сконфигурированный» отправитель НЕ уходит в сеть: собирает конверт и
// возвращает ErrTransportDisabled (гарантия «в ЕСЭДО не отправлять»).
func TestSoapTransportDisabled(t *testing.T) {
	s := NewSoapSender(Config{
		Enabled:  true,
		Endpoint: "https://vshep.example/SendMessage",
		SenderID: "sender-1",
		Password: "secret",
	}, StubSigner{}, nil)
	if _, err := s.SendOutgoing(context.Background(), sampleDoc()); !errors.Is(err, ErrTransportDisabled) {
		t.Fatalf("ожидали ErrTransportDisabled, получили %v", err)
	}
}

func TestBuildEnvelope(t *testing.T) {
	s := NewSoapSender(Config{Enabled: true, Endpoint: "x", SenderID: "sender-1", Password: "p"}, StubSigner{}, nil)
	s.nowFn = func() time.Time { return time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC) }
	s.newID = func() string { return "MSG-TEST" }

	doc := sampleDoc()
	if err := s.SignAndAttach(context.Background(), &doc, []byte("pdf-bytes")); err != nil {
		t.Fatalf("SignAndAttach: %v", err)
	}
	if !doc.SecondSignEnabled || doc.SecondSignData == "" {
		t.Fatalf("SignAndAttach не проставил подпись: enabled=%v data=%q", doc.SecondSignEnabled, doc.SecondSignData)
	}

	body, err := s.buildEnvelope(doc)
	if err != nil {
		t.Fatalf("buildEnvelope: %v", err)
	}
	xmlStr := string(body)
	for _, want := range []string{
		"ESEDO_UNIVERSAL_SERVICE",
		"<routeId>R_ESEDO</routeId>", // маршрут из шаблона (фикс.)
		`xsi:type="ns1:docOutgoing"`, // тип полезной нагрузки
		"MSG-TEST",                   // messageId
		"<sessionId>{}</sessionId>",  // sessionId в фигурных скобках (пустой)
		"<docNo>38219</docNo>",       // рег. номер
		"<from>17481752</from>",      // код отправителя
		"<performers>2301015</performers>",
		"<senderOrg>2309569</senderOrg>",
		"<secondSignEnabled>1</secondSignEnabled>",
		"<fileIdentifier>3828910d-c197-4cee-bb1c-3537080b44ea</fileIdentifier>",
		"Акционерное общество «Центр электронных финансов»", // получатель (рус)
	} {
		if !strings.Contains(xmlStr, want) {
			t.Fatalf("в конверте нет %q\n%s", want, xmlStr)
		}
	}
}
