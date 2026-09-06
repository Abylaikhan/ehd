// Package domain — предметная область модуля ОБМ ЕВГА (спека 007):
// реестр записей витрины рисков 5-15а внешней БД obm_evga, строго read-only.
package domain

import "time"

// Роли модуля (spec 007 FR-4). Работаем по кодам ролей ЕХД.
const (
	RoleAuditor = "evga_auditor"
	RoleCurator = "evga_curator"
)

// RiskRecord — запись витрины its_tb_5_15a в объёме грида и карточки (EVGA-FR-012, FR-9).
// Суммы — строками: numeric(20,2) не должен терять точность в float.
type RiskRecord struct {
	ID          int64
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
	God         *int64
	Mes         *int64

	ProfileID    int64
	ProfileTitle string
	StatusID     *int64
	StatusTitle  string
	StatusNote   string

	DepartmentID    *int64
	DepartmentTitle string

	NoticeID  *int64
	NoticeNum string
	OutNum    string

	// Поля карточки (FR-9)
	Refund           string
	AmountForVozvrat string
	ActivityID       *int64
	IsGBDFL          *int16
	FlFM             string
	FlNM             string
	FlFT             string
	CreatedAt        *time.Time
	UpdatedAt        *time.Time
}

// Filter — фильтры реестра (EVGA-FR-013 / spec FR-6). Nil/пусто — фильтр не применяется.
type Filter struct {
	ProfileID    *int64
	StatusID     *int64
	God          *int64
	Mes          *int64
	PaymentFrom  *time.Time
	PaymentTo    *time.Time
	GU           string
	SenderName   string
	IIN          string
	AmountFrom   *float64
	AmountTo     *float64
	InNotice     *bool
	NoticeNum    string
	DepartmentID *int64 // применяется только при scope «все департаменты»
}

// Page — запрос страницы (spec FR-7).
type Page struct {
	Page  int
	Size  int
	Sort  string
	Order string
}

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
	ExportLimit     = 50000
)

// sortWhitelist — допустимые поля сортировки → SQL-выражения (Принцип 3: только whitelist).
var sortWhitelist = map[string]string{
	"paymentdate": "t.paymentdate",
	"amount_part": "t.amount_part",
	"god_mes":     "t.god",
	"id":          "t.id",
}

// SortExpr возвращает SQL-выражение сортировки или ErrInvalidSort.
// Пустой sort — сортировка по умолчанию: свежие платежи сверху.
func SortExpr(sort, order string) (string, error) {
	if sort == "" {
		sort = "paymentdate"
	}
	col, ok := sortWhitelist[sort]
	if !ok {
		return "", ErrInvalidSort
	}
	dir := "desc"
	switch order {
	case "", "desc":
	case "asc":
		dir = "asc"
	default:
		return "", ErrInvalidSort
	}
	expr := col + " " + dir
	if sort == "god_mes" {
		expr += ", t.mes " + dir
	}
	return expr + ", t.id " + dir, nil
}

// Normalize приводит страницу к допустимым границам (spec FR-7).
func (p Page) Normalize() Page {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Size < 1 {
		p.Size = DefaultPageSize
	}
	if p.Size > MaxPageSize {
		p.Size = MaxPageSize
	}
	return p
}

// DepartmentScope — доверенная видимость пользователя (spec FR-3/4).
// Ровно одно из состояний: департамент | все департаменты | не сопоставлен.
type DepartmentScope struct {
	DepartmentID    *int64
	DepartmentTitle string
	AllDepartments  bool
	Unmapped        bool
}

// Reference — элемент справочника модуля (статус/профиль/департамент).
type Reference struct {
	ID    int64
	Code  string
	Title string
}

// RecordPage — результат листинга.
type RecordPage struct {
	Items []RiskRecord
	Total int64
}
