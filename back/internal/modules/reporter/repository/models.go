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

// Migrate создаёт/обновляет таблицы Reporter через GORM AutoMigrate.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&AuditLogModel{},
		&DataSourceModel{},
	)
}
