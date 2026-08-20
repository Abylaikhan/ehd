// Package repository — GORM-модели и доступ к данным Auth Module (схема auth).
// Модели живут в repository, доменные сущности (domain) остаются чистыми.
package repository

import (
	"time"

	"github.com/google/uuid"
)

type UserModel struct {
	ID                     uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Login                  string    `gorm:"size:64;uniqueIndex;not null"`
	IINEnc                 []byte
	IINHmac                *string `gorm:"size:128;uniqueIndex"`
	FullNameEnc            []byte
	PhoneEnc               []byte
	Email                  string  `gorm:"size:255"`
	IINVerified            bool    `gorm:"not null;default:false"`
	Status                 string  `gorm:"size:16;not null;default:pending"`
	PasswordHash           *string `gorm:"size:255"`
	FailedAttempts         int16   `gorm:"not null;default:0"`
	CertificateBIN         *string `gorm:"size:12"`
	PasswordChangeRequired bool    `gorm:"not null;default:false"`
	TempPasswordExpiresAt  *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (UserModel) TableName() string { return "auth.users" }

type RoleModel struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Code      string    `gorm:"size:64;uniqueIndex;not null"`
	NameRu    string    `gorm:"size:255;not null"`
	NameKk    string    `gorm:"size:255"`
	Status    string    `gorm:"size:16;not null;default:active"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (RoleModel) TableName() string { return "auth.roles" }

type UserRoleModel struct {
	UserID uuid.UUID `gorm:"type:uuid;primaryKey"`
	RoleID uuid.UUID `gorm:"type:uuid;primaryKey"`
}

func (UserRoleModel) TableName() string { return "auth.user_roles" }

type SessionModel struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;index;not null"`
	TokenHash string    `gorm:"size:64;uniqueIndex;not null"`
	CreatedAt time.Time
	// максимум пользовательской сессии по ТЗ — 3 часа
	ExpiresAt time.Time `gorm:"not null;index"`
	RevokedAt *time.Time
}

func (SessionModel) TableName() string { return "auth.sessions" }

type RegionModel struct {
	ID     uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Code   string    `gorm:"size:32;uniqueIndex;not null"`
	NameRu string    `gorm:"size:255;not null"`
	NameKk string    `gorm:"size:255"`
	Status string    `gorm:"size:16;not null;default:active"`
}

func (RegionModel) TableName() string { return "auth.regions" }

type DepartmentModel struct {
	ID     uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Code   string    `gorm:"size:32;uniqueIndex;not null"`
	NameRu string    `gorm:"size:255;not null"`
	NameKk string    `gorm:"size:255"`
	Status string    `gorm:"size:16;not null;default:active"`
}

func (DepartmentModel) TableName() string { return "auth.departments" }

type UserRegionModel struct {
	UserID   uuid.UUID `gorm:"type:uuid;primaryKey"`
	RegionID uuid.UUID `gorm:"type:uuid;primaryKey"`
}

func (UserRegionModel) TableName() string { return "auth.user_regions" }

type UserDepartmentModel struct {
	UserID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	DepartmentID uuid.UUID `gorm:"type:uuid;primaryKey"`
}

func (UserDepartmentModel) TableName() string { return "auth.user_departments" }
