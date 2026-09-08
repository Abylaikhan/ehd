package domain

import "time"

// Статусы заявки уведомления в платформенном справочнике its_req_stat
// (снято с прода, 14-shema-bd.md; «Исполнен» из ТЗ = «Отправлен объекту аудита»).
const (
	ReqStatDraft     int64 = 1  // Проект создан
	ReqStatApproving int64 = 2  // На согласовании
	ReqStatSent      int64 = 3  // Отправлен объекту аудита (финал, «Исполнен»)
	ReqStatSigning   int64 = 5  // На подписании
	ReqStatRework    int64 = 12 // Отправлен на доработку
	ReqStatRevoked   int64 = 15 // Отозван (официальное аннулирование, ответ В4 07.09.2026)
)

// DefaultNoticeText — утверждённый текст уведомления (ТЗ §9.2, русская версия).
// Подмена на присланный заказчиком пример и админ-редактирование — фаза 5 (spec 009 FR-11).
const DefaultNoticeText = "Онлайн бюджетным мониторингом идентифицированы высокие риски " +
	"в расходной части бюджета по случаям, указанным в приложении к настоящему уведомлению. " +
	"На момент уведомления наблюдаются признаки неправомерного перечисления на карт-счета " +
	"физических лиц. С целью пресечения необоснованного использования бюджетных средств " +
	"рекомендуется самостоятельное устранение рисков в срок не позднее 10 (десяти) рабочих " +
	"дней. В противном случае будет рассмотрен вопрос о принятии директивных мер реагирования."

// Причины отклонения записи из формирования (spec 009 FR-2).
const (
	RejectNotFound  = "not_found"         // не найдена / чужой департамент / удалена
	RejectBadStatus = "bad_status"        // статус ≠ 1 «В работе у ДВГА»
	RejectInNotice  = "already_in_notice" // уже в действующем уведомлении
	RejectEmptyGU   = "gu_empty"          // не заполнен код ГУ
)

// RejectedRecord — запись, не попавшая в формирование, с причиной (EVGA-FR-030/036).
type RejectedRecord struct {
	ID        int64
	Code      string // Reject*-код
	Reason    string // человекочитаемо
	NoticeNum string // для already_in_notice — номер занявшего уведомления (может быть пустым)
}

// GroupRecord — запись внутри группы предпросмотра.
type GroupRecord struct {
	ID          int64
	PpoPp       string
	PaymentDate *time.Time
	IIN         string
	FIO         string
	AmountPart  string
}

// NoticeGroup — группа записей одного ГУ = будущее уведомление (EVGA-FR-031/032).
type NoticeGroup struct {
	GU         string
	GUBIN      string
	SenderName string
	Count      int
	TotalSum   string
	Records    []GroupRecord
}

// CreateGroupInput — подтверждённая группа для создания (spec 009 FR-4).
type CreateGroupInput struct {
	RecordIDs []int64
	CliID     int64
}

// CreatedNotice — результат создания одной группы.
type CreatedNotice struct {
	NoticeID int64
	GU       string
	Records  int
}

// CliOrg — организация ЕСЭДО для выбора адресата (FR-3).
type CliOrg struct {
	ID     int64
	Code   string
	BinIIN string
	Title  string
}

// NoticeListItem — строка списка уведомлений (FR-7).
type NoticeListItem struct {
	ID          int64
	Docnum      string
	StatusID    *int64
	StatusTitle string
	GU          string
	SenderName  string
	Recipient   string
	RowsCount   int64
	TotalSum    string
	CreatedAt   *time.Time
	Department  string
}

// NoticeCard — карточка уведомления (FR-8).
type NoticeCard struct {
	NoticeListItem
	NoticeTxt string
	Rows      []NoticeRow
}

// NoticeRow — строка снимка (EVGA-FR-038).
type NoticeRow struct {
	ID          int64
	TB515AID    *int64
	PpoPp       string
	PaymentDate *time.Time
	IIN         string
	FM          string
	NM          string
	FT          string
	LA1         string
	AmountPart  string
	GU          string
	GUBIN       string
	SenderName  string
}
