package domain

import (
	"regexp"
	"time"
)

// Статусы представления по ТЗ (раздел 5, статусная модель Data View).
const (
	ViewStatusDraft       = "draft"
	ViewStatusPublished   = "published"
	ViewStatusDisabled    = "disabled"
	ViewStatusSchemaError = "schema_error"
	ViewStatusArchived    = "archived"
)

// Режимы источника запроса.
const (
	SourceModePhysicalObject = "physical_object"
	SourceModeManagedQuery   = "managed_query" // Phase 2, за feature flag
)

// Режимы ограничения строк (ТЗ, Row scope).
const (
	RowScopeByProfile    = "by_profile"
	RowScopeUnrestricted = "unrestricted"
)

// Направление сортировки.
const (
	SortAsc  = "asc"
	SortDesc = "desc"
)

// Границы пагинации и экспорта из ТЗ (раздел «Создание Data View»).
const (
	DefaultPageSize = 50
	MinPageSize     = 20
	MaxPageSize     = 200
	MaxExportRows   = 100000
)

// DataView — конфигурация представления таблицы ClickHouse.
type DataView struct {
	ID           string
	Name         string
	Slug         string
	Description  string
	DataSourceID string
	DatabaseName string
	TableName    string
	SourceMode   string
	Status       string
	Revision     int64
	SchemaHash   string

	// Параметры поведения (ТЗ «Создание Data View»).
	PageSizeDefault   int
	PageSizeMin       int
	PageSizeMax       int
	DefaultSortColumn string
	DefaultSortDir    string
	ExportRowLimit    int
	RowScopeMode      string

	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

var slugRe = regexp.MustCompile(`^[a-z0-9-]+$`)

// ValidSlug проверяет slug: lowercase, латиница/цифры/дефис (ТЗ, Slug).
func ValidSlug(s string) bool { return s != "" && slugRe.MatchString(s) }

// ValidRowScope проверяет режим ограничения строк.
func ValidRowScope(m string) bool {
	return m == RowScopeByProfile || m == RowScopeUnrestricted
}
