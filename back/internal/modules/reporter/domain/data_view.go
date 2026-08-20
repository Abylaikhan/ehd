package domain

import "time"

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

// DataView — конфигурация опубликованного представления таблицы ClickHouse.
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
	PublishedAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
