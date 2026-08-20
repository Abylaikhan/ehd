package domain

import "time"

// Статусы пользователя по ТЗ (раздел 9, users.status).
const (
	UserStatusPending = "pending"
	UserStatusActive  = "active"
	UserStatusBlocked = "blocked"
)

// User — доменная сущность пользователя ЕХД.
// Персональные поля (ИИН, ФИО, телефон) хранятся зашифрованными на уровне приложения.
type User struct {
	ID             string
	Login          string
	Email          string
	IINVerified    bool
	Status         string
	FailedAttempts int
	CertificateBIN string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

const RoleAdminCode = "admin"

type Role struct {
	ID     string
	Code   string
	NameRu string
	NameKk string
	Status string
}
