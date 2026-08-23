package domain

import "time"

// Протоколы подключения к ClickHouse (REP-FR-010).
const (
	ProtocolNative = "native"
	ProtocolHTTP   = "http"
)

// Статусы источника (REP-FR-012).
const (
	SourceStatusActive   = "active"
	SourceStatusInactive = "inactive"
)

// DataSource — настроенное read-only подключение Reporter к ClickHouse.
// Пароль не входит в доменную сущность: секрет хранится зашифрованным в repository
// и никогда не покидает сервер (REP-FR-013).
type DataSource struct {
	ID            string
	Code          string
	Name          string
	Host          string
	Port          int
	Protocol      string
	TLSEnabled    bool
	TLSSkipVerify bool
	Username      string
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ValidProtocol проверяет допустимость протокола подключения.
func ValidProtocol(p string) bool {
	return p == ProtocolNative || p == ProtocolHTTP
}

// ValidStatus проверяет допустимость статуса источника.
func ValidStatus(s string) bool {
	return s == SourceStatusActive || s == SourceStatusInactive
}
