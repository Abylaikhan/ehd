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

// Migrate создаёт/обновляет таблицы схемы reporter через GORM AutoMigrate.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&AuditLogModel{},
	)
}
