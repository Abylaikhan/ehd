// Package esedo — ЗАГОТОВКА отправки исходящего в ЕСЭДО (спека 014, не активирована).
//
// ⚠ Ничего не отправляется: StubSender (по умолчанию) не ходит по сети, а SoapSender —
// скелет с намеренно отключённым транспортом (ErrTransportDisabled) и рубильником
// Config.Enabled=false. Активация — после доступов ШЭП/сертификата и снятия ограничения
// «в ЕСЭДО запросы не отправлять» (см. spec 014 §5).
//
// Контракт docOutgoing/ESEDO_UNIVERSAL_SERVICE — по шаблону аналитика от 10.09.2026
// (voprosy-analitiku-4.md; конспект 13-esedo-integratsiya.md). Имена и порядок полей
// воспроизводят присланный шаблон 1:1.
package esedo

// DocOutgoing — исходящий документ (тип ns1:docOutgoing,
// http://esedo.nitec.kz/service/model/document). Поля/маппинг — по шаблону
// ESEDO_doc_outgoing (см. пример вызова в voprosy-analitiku-4.md).
// Даты передаются строками ISO-8601 с зоной (как в примере: 2026-05-04T17:33:03.294+05:00).
type DocOutgoing struct {
	// Идентификаторы
	Href        string // metadataSystem.href — id документа в нашей системе
	SectionUUID string // activityId и documentSectionId (в шаблоне — один uuid)
	SessionID   string // requestInfo.sessionId (оборачивается в {})

	// Адресация
	From       string   // metadataSystem.from — код орг-отправителя
	SenderOrg  string   // metadataSystem.senderOrg
	Performers []string // metadataSystem.performers — коды получателей (объект аудита)

	// Реквизиты документа
	DocNo              string // docNo — рег. номер
	DocDate            string // docDate
	OutTime            string // outTime и preparedDate
	DocLang            string // docLang (ru/kz)
	DocKind            string // docKind — код НСИ вида документа
	Character          string // character — код НСИ характера
	Description        string // description
	AppendCount        string // appendCount
	SheetCount         string // sheetCount
	EmployeePhone      string // employeePhone
	Executor           string // executor (текст)
	AuthorNameKz       string
	AuthorNameRu       string
	SignerNameKz       string
	SignerNameRu       string
	DocumentReceiverKz string // documentReceiverKz — наименование получателя (каз)
	DocumentReceiverRu string // documentReceiverRu — наименование получателя (рус)

	// Вложения: fileIdentifier'ы, полученные из EDS_TEMP_FILES
	Attachments []FileRef

	// Подпись документа: secondSignData — base64 CMS ГОСТ (сертификат исполнителя),
	// secondSignEnabled=1. Подпись конверта (WSSE, сертификат госслужащего) — отдельно, signWSSE.
	SecondSignEnabled bool
	SecondSignData    string // base64 (значение .input.secondsign)
}

// FileRef — ссылка на файл, ранее загруженный в ХЭД (EDS_TEMP_FILES / MIND-S-0048).
type FileRef struct{ FileIdentifier string }

// Attachment — файл для загрузки в EDS_TEMP_FILES до отправки docOutgoing.
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// SendResult — синхронная квитанция ЕСЭДО на SendMessage. Это НЕ регистрация: рег. номер
// приходит позже извещением stateRegistered (приём — на стороне платформы ОБМ, ответ №3-7).
type SendResult struct {
	MessageID string
	Accepted  bool
	Note      string
}

// RegisteredNotice — разбор извещения stateRegistered (regNo+date+href). Оставлено на случай,
// если понадобится собственный приёмник извещений; сейчас извещения принимает платформа ОБМ.
type RegisteredNotice struct {
	RegNo string
	Date  string
	Href  string
}
