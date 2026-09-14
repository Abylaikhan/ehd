package esedo

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"go.uber.org/zap"
)

// SoapSender — СКЕЛЕТ клиента ШЭП/ВШЭП (ESEDO_UNIVERSAL_SERVICE + EDS_TEMP_FILES).
// Собирает конверт docOutgoing по шаблону аналитика (10.09.2026), НО реальный сетевой вызов
// намеренно отключён (ErrTransportDisabled) плюс рубильник Config.Enabled. Активация — spec 014.
//
// Конверт, который собирает buildEnvelope, ещё НЕ подписан WSSE: в бою его оборачивает подпись
// госслужащего (signWSSE с серверным сертификатом ГОСТ) — это отдельный шаг при активации.
type SoapSender struct {
	cfg    Config
	signer Signer
	http   *http.Client // будет использован при активации (сейчас Do не вызывается)
	log    *zap.Logger
	nowFn  func() time.Time
	newID  func() string
}

// NewSoapSender — скелет реального отправителя. log/signer могут быть nil.
func NewSoapSender(cfg Config, signer Signer, log *zap.Logger) *SoapSender {
	if log == nil {
		log = zap.NewNop()
	}
	if cfg.ServiceID == "" {
		cfg.ServiceID = DefaultServiceID
	}
	if cfg.RouteID == "" {
		cfg.RouteID = DefaultRouteID
	}
	s := &SoapSender{
		cfg: cfg, signer: signer,
		http:  &http.Client{Timeout: 60 * time.Second},
		log:   log,
		nowFn: time.Now,
	}
	// TODO(ШЭП): заменить на UUID v4 для messageId.
	s.newID = func() string { return "MSG-" + strconv.FormatInt(s.nowFn().UnixNano(), 10) }
	return s
}

// UploadAttachment — загрузка файла в EDS_TEMP_FILES (MIND-S-0048). Транспорт отключён.
func (s *SoapSender) UploadAttachment(_ context.Context, a Attachment) (string, error) {
	if !s.cfg.Enabled {
		return "", ErrDisabled
	}
	if s.cfg.Endpoint == "" {
		return "", ErrNotConfigured
	}
	// TODO(ШЭП): SendMessage(serviceId=EDS_TEMP_FILES) TempStorageRequest/UPLOAD → fileIdentifier.
	s.log.Warn("esedo(soap): загрузка в EDS_TEMP_FILES отключена (заготовка)",
		zap.String("file", a.Filename), zap.Int("bytes", len(a.Data)))
	return "", ErrTransportDisabled
}

// SendOutgoing — отправка docOutgoing через ESEDO_UNIVERSAL_SERVICE. Собирает конверт, но НЕ шлёт.
func (s *SoapSender) SendOutgoing(ctx context.Context, doc DocOutgoing) (SendResult, error) {
	if !s.cfg.Enabled {
		return SendResult{}, ErrDisabled
	}
	if s.cfg.Endpoint == "" || s.cfg.SenderID == "" {
		return SendResult{}, ErrNotConfigured
	}
	body, err := s.buildEnvelope(doc)
	if err != nil {
		return SendResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.Endpoint, bytes.NewReader(body))
	if err != nil {
		return SendResult{}, err
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", "SendMessage")

	// TODO(ШЭП): после доступов/сертификата и снятия ограничения —
	//   1) подписать конверт WSSE подписью госслужащего (signWSSE, серверный сертификат);
	//   2) s.http.Do(req); разобрать квитанцию SendMessage → SendResult.
	// Сейчас транспорт НАМЕРЕННО отключён (ограничение «в ЕСЭДО запросы не отправлять»).
	s.log.Warn("esedo(soap): транспорт отключён — конверт собран, но НЕ отправлен",
		zap.String("endpoint", s.cfg.Endpoint), zap.String("method", req.Method), zap.Int("bytes", len(body)))
	return SendResult{}, ErrTransportDisabled
}

// SignAndAttach ставит подпись документа (secondSignData, base64 CMS) через Signer — шов подписи.
// payload — байты подписываемого документа (обычно PDF/тело). Подпись КОНВЕРТА (WSSE) — отдельно.
func (s *SoapSender) SignAndAttach(ctx context.Context, doc *DocOutgoing, payload []byte) error {
	if s.signer == nil {
		return errors.New("esedo: signer не задан")
	}
	cms, err := s.signer.SignCMS(ctx, payload)
	if err != nil {
		return err
	}
	doc.SecondSignEnabled = true
	doc.SecondSignData = base64.StdEncoding.EncodeToString(cms)
	return nil
}

// --- сборка конверта docOutgoing по шаблону аналитика (voprosy-analitiku-4.md, 10.09.2026) ---

func (s *SoapSender) buildEnvelope(doc DocOutgoing) ([]byte, error) {
	var performers strings.Builder
	for _, p := range doc.Performers {
		performers.WriteString("<performers>" + xmlEsc(p) + "</performers>")
	}
	var attachments strings.Builder
	for _, a := range doc.Attachments {
		attachments.WriteString("<attachments><fileIdentifier>" + xmlEsc(a.FileIdentifier) + "</fileIdentifier></attachments>")
	}
	secondSignEnabled := "0"
	if doc.SecondSignEnabled {
		secondSignEnabled = "1"
	}
	data := envData{
		BodyID:             "id-ehd-" + s.newID(),
		MessageID:          s.newID(),
		ServiceID:          xmlEsc(s.cfg.ServiceID),
		MessageDate:        s.nowFn().Format("2006-01-02T15:04:05.000-07:00"),
		RouteID:            xmlEsc(s.cfg.RouteID),
		SenderID:           xmlEsc(s.cfg.SenderID),
		Password:           xmlEsc(s.cfg.Password),
		SessionID:          xmlEsc(doc.SessionID),
		Attachments:        attachments.String(),
		Performers:         performers.String(),
		SectionUUID:        xmlEsc(doc.SectionUUID),
		From:               xmlEsc(doc.From),
		Href:               xmlEsc(doc.Href),
		SenderOrg:          xmlEsc(doc.SenderOrg),
		AppendCount:        xmlEsc(doc.AppendCount),
		AuthorNameKz:       xmlEsc(doc.AuthorNameKz),
		AuthorNameRu:       xmlEsc(doc.AuthorNameRu),
		Character:          xmlEsc(doc.Character),
		Description:        xmlEsc(doc.Description),
		DocDate:            xmlEsc(doc.DocDate),
		DocKind:            xmlEsc(doc.DocKind),
		DocLang:            xmlEsc(doc.DocLang),
		DocNo:              xmlEsc(doc.DocNo),
		DocumentReceiverKz: xmlEsc(doc.DocumentReceiverKz),
		DocumentReceiverRu: xmlEsc(doc.DocumentReceiverRu),
		EmployeePhone:      xmlEsc(doc.EmployeePhone),
		Executor:           xmlEsc(doc.Executor),
		OutTime:            xmlEsc(doc.OutTime),
		SecondSignData:     xmlEsc(doc.SecondSignData),
		SecondSignEnabled:  secondSignEnabled,
		SheetCount:         xmlEsc(doc.SheetCount),
		SignerNameKz:       xmlEsc(doc.SignerNameKz),
		SignerNameRu:       xmlEsc(doc.SignerNameRu),
	}
	var buf bytes.Buffer
	if err := docOutgoingTmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// envData — значения для подстановки в шаблон (все уже XML-экранированы; Attachments/Performers —
// готовые XML-фрагменты).
type envData struct {
	BodyID, MessageID, ServiceID, MessageDate, RouteID, SenderID, Password, SessionID string
	Attachments, Performers                                                           string
	SectionUUID, From, Href, SenderOrg                                                string
	AppendCount, AuthorNameKz, AuthorNameRu, Character, Description                   string
	DocDate, DocKind, DocLang, DocNo                                                  string
	DocumentReceiverKz, DocumentReceiverRu                                            string
	EmployeePhone, Executor, OutTime                                                  string
	SecondSignData, SecondSignEnabled, SheetCount, SignerNameKz, SignerNameRu         string
}

var xmlReplacer = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")

func xmlEsc(s string) string { return xmlReplacer.Replace(s) }

// docOutgoingTmpl — 1:1 воспроизведение шаблона аналитика ESEDO_doc_outgoing (без WSSE-обёртки:
// её накладывает signWSSE при активации). preparedDate = outTime (как в шаблоне).
var docOutgoingTmpl = template.Must(template.New("docOutgoing").Parse(
	`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <soap:Header xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"/>
 <soap:Body xmlns:wsu="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd" wsu:Id="{{.BodyID}}">
  <SendMessage xmlns="http://bip.bee.kz/SyncChannel/v10/Types">
<request xmlns="">
    <requestInfo>
        <messageId>{{.MessageID}}</messageId>
        <correlationId/>
        <serviceId>{{.ServiceID}}</serviceId>
        <messageDate>{{.MessageDate}}</messageDate>
        <routeId>{{.RouteID}}</routeId>
        <sender>
            <senderId>{{.SenderID}}</senderId>
            <password>{{.Password}}</password>
        </sender>
        <sessionId>{{"{"}}{{.SessionID}}{{"}"}}</sessionId>
    </requestInfo>
    <requestData>
        <data xsi:type="ns1:docOutgoing" xmlns:ns1="http://esedo.nitec.kz/service/model/document">
        {{.Attachments}}
            <metadataSystem>
                <activityId>{{.SectionUUID}}</activityId>
                <from>{{.From}}</from>
                <href>{{.Href}}</href>
                {{.Performers}}
                <senderOrg>{{.SenderOrg}}</senderOrg>
            </metadataSystem>
            <appendCount>{{.AppendCount}}</appendCount>
            <authorNameKz>{{.AuthorNameKz}}</authorNameKz>
            <authorNameRu>{{.AuthorNameRu}}</authorNameRu>
            <carrierType>1</carrierType>
            <character>{{.Character}}</character>
            <controlTypeOuterCode/>
            <controlTypeOuterNameKz/>
            <controlTypeOuterNameRu/>
            <description>{{.Description}}</description>
            <docDate>{{.DocDate}}</docDate>
            <docKind>{{.DocKind}}</docKind>
            <docLang>{{.DocLang}}</docLang>
            <docNo>{{.DocNo}}</docNo>
            <docNoR/>
            <docRecPostKz/>
            <docRecPostRu/>
            <docToNumber/>
            <documentReceiverKz>{{.DocumentReceiverKz}}</documentReceiverKz>
            <documentReceiverRu>{{.DocumentReceiverRu}}</documentReceiverRu>
            <documentSectionId>{{.SectionUUID}}</documentSectionId>
            <employeePhone>{{.EmployeePhone}}</employeePhone>
            <executor>{{.Executor}}</executor>
            <idPortalInternal>0</idPortalInternal>
            <outTime>{{.OutTime}}</outTime>
            <portalSign/>
            <preparedDate>{{.OutTime}}</preparedDate>
            <resolutionText/>
            <secondSignData>{{.SecondSignData}}</secondSignData>
            <secondSignEnabled>{{.SecondSignEnabled}}</secondSignEnabled>
            <sectionId/>
            <sheetCount>{{.SheetCount}}</sheetCount>
            <signerNameKz>{{.SignerNameKz}}</signerNameKz>
            <signerNameRu>{{.SignerNameRu}}</signerNameRu>
            <userUin/>
        </data>
    </requestData>
</request>
</SendMessage>
 </soap:Body>
</soap:Envelope>`))
