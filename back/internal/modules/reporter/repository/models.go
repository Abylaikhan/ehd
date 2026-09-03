// Package repository — GORM-модели и доступ к данным Reporter Module (схема reporter).
package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditLogModel — журнал административных действий, запросов и экспортов.
// Срок хранения по ТЗ — 3 месяца; чистка регламентным заданием.
type AuditLogModel struct {
	ID         uint64     `gorm:"primaryKey;autoIncrement"`
	ActorID    *uuid.UUID `gorm:"type:uuid"`
	ActorType  string     `gorm:"size:16;not null;default:user"`
	Action     string     `gorm:"size:64;not null"`
	EntityType string     `gorm:"size:64;not null;index:idx_audit_entity"`
	EntityID   *string    `gorm:"size:64;index:idx_audit_entity"`
	RequestID  *string    `gorm:"size:64"`
	Metadata   string     `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt  time.Time  `gorm:"index"`
}

func (AuditLogModel) TableName() string { return "audit_logs" }

// DataSourceModel — единственный read-only источник ClickHouse (REP-FR-010..014).
// PasswordEnc хранит пароль в зашифрованном виде (AES-256-GCM); ключ вне БД.
type DataSourceModel struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	Code          string    `gorm:"size:64;not null;uniqueIndex"`
	Name          string    `gorm:"size:255;not null"`
	Host          string    `gorm:"size:255;not null"`
	Port          int       `gorm:"not null"`
	Protocol      string    `gorm:"size:16;not null;default:native"`
	TLSEnabled    bool      `gorm:"not null;default:false"`
	TLSSkipVerify bool      `gorm:"not null;default:false"`
	Username      string    `gorm:"size:128;not null"`
	PasswordEnc   []byte    `gorm:"type:bytea;not null"`
	Status        string    `gorm:"size:16;not null;default:inactive"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (DataSourceModel) TableName() string { return "data_sources" }

// DataViewModel — конфигурация представления таблицы (REP-FR-040..044).
// PublishedSnapshot — неизменяемая опубликованная конфигурация (jsonb) для Query Engine.
type DataViewModel struct {
	ID                       uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name                     string    `gorm:"size:255;not null"`
	Slug                     string    `gorm:"size:128;not null;uniqueIndex"`
	Description              string    `gorm:"size:1024"`
	DataSourceID             uuid.UUID `gorm:"type:uuid;not null;index"`
	DatabaseName             string    `gorm:"size:255;not null"`
	SourceTable              string    `gorm:"column:table_name;size:255;not null"`
	SourceMode               string    `gorm:"size:32;not null;default:physical_object"`
	Status                   string    `gorm:"size:16;not null;default:draft;index"`
	Revision                 int64     `gorm:"not null;default:0"`
	SchemaHash               string    `gorm:"size:64"`
	PageSizeDefault          int       `gorm:"not null;default:50"`
	PageSizeMin              int       `gorm:"not null;default:20"`
	PageSizeMax              int       `gorm:"not null;default:200"`
	DefaultSortColumn        string    `gorm:"size:255"`
	DefaultSortDir           string    `gorm:"size:4"`
	ExportRowLimit           int       `gorm:"not null;default:100000"`
	RowScopeMode             string    `gorm:"size:16;not null;default:by_profile"`
	KeysetColumn             string    `gorm:"size:255"`
	KeysetColumns            string    `gorm:"size:512"`
	KeysetDir                string    `gorm:"size:4;not null;default:asc"`
	RowScopeRegionColumn     string    `gorm:"size:255"`
	RowScopeDepartmentColumn string    `gorm:"size:255"`
	PublishedSnapshot        *string   `gorm:"type:jsonb"` // NULL пока не опубликовано (пустая строка — невалидный jsonb)
	PublishedAt              *time.Time
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

func (DataViewModel) TableName() string { return "data_views" }

// ViewColumnModel — настройка колонки представления (ТЗ, «Настройка колонок»).
type ViewColumnModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	ViewID      uuid.UUID `gorm:"type:uuid;not null;index:idx_view_columns_view"`
	SourceName  string    `gorm:"size:255;not null"`
	SourceType  string    `gorm:"size:255;not null"`
	Label       string    `gorm:"size:255"`
	DisplayType string    `gorm:"size:16"`
	Position    int       `gorm:"not null;default:0"`
	Visible     bool      `gorm:"not null;default:false"`
	Searchable  bool      `gorm:"not null;default:false"`
	Filterable  bool      `gorm:"not null;default:false"`
	Sortable    bool      `gorm:"not null;default:false"`
	Exportable  bool      `gorm:"not null;default:false"`
	Format      string    `gorm:"type:jsonb;not null;default:'{}'"`
	MaskRule    string    `gorm:"size:16;not null;default:none"`
	Width       int       `gorm:"not null;default:0"`
	NullLabel   string    `gorm:"size:255"`
}

func (ViewColumnModel) TableName() string { return "view_columns" }

// ViewPermissionModel — доступ роли к представлению (ТЗ, Права).
type ViewPermissionModel struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	ViewID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_view_perm,priority:1"`
	RoleCode string    `gorm:"size:64;not null;uniqueIndex:idx_view_perm,priority:2"`
}

func (ViewPermissionModel) TableName() string { return "view_permissions" }

// MenuItemModel — пункт навигации Reporter (ТЗ, menu_items).
type MenuItemModel struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
	ParentID     *uuid.UUID `gorm:"type:uuid;index"`
	DataViewID   *uuid.UUID `gorm:"type:uuid;index"`
	NameRu       string     `gorm:"size:255;not null"`
	NameKk       string     `gorm:"size:255"`
	IconKey      string     `gorm:"size:64"`
	Position     int        `gorm:"not null;default:0"`
	IsDisabled   bool       `gorm:"not null;default:false"`
	PublicAccess bool       `gorm:"not null;default:false"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (MenuItemModel) TableName() string { return "menu_items" }

// MenuItemRoleModel — роли, которым виден пункт при public_access=false.
type MenuItemRoleModel struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	MenuItemID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_menu_role,priority:1"`
	RoleCode   string    `gorm:"size:64;not null;uniqueIndex:idx_menu_role,priority:2"`
}

func (MenuItemRoleModel) TableName() string { return "menu_item_roles" }

// Migrate создаёт/обновляет таблицы Reporter через GORM AutoMigrate.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&AuditLogModel{},
		&DataSourceModel{},
		&DataViewModel{},
		&ViewColumnModel{},
		&ViewPermissionModel{},
		&MenuItemModel{},
		&MenuItemRoleModel{},
	)
}
