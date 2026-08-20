package domain

import "time"

// Статусы пользователя по ТЗ (раздел 9, users.status).
const (
	UserStatusPending = "pending"
	UserStatusActive  = "active"
	UserStatusBlocked = "blocked"
)

const RoleAdminCode = "admin"

// Статусы записей справочников/ролей.
const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

// User — доменная сущность пользователя ЕХД (чистые данные).
// Персональные поля (ИИН, ФИО, телефон) хранятся зашифрованными; ИИН имеет HMAC для поиска.
type User struct {
	ID                     string
	Login                  string
	Email                  string
	IINEnc                 []byte
	IINHmac                string
	FullNameEnc            []byte
	PhoneEnc               []byte
	IINVerified            bool
	Status                 string
	PasswordHash           *string
	FailedAttempts         int
	CertificateBIN         *string
	PasswordChangeRequired bool
	TempPasswordExpiresAt  *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// Role — пользовательская или системная роль.
type Role struct {
	ID     string
	Code   string
	NameRu string
	NameKk string
	Status string
}

// Session — серверная сессия; в БД хранится sha256(token), а не сам токен.
type Session struct {
	ID        string
	UserID    string
	TokenHash string
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// Reference — запись справочника (регион или подразделение).
type Reference struct {
	ID     string
	Code   string
	NameRu string
	NameKk string
	Status string
}
