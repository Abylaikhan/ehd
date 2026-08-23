package clickhouse

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ConnParams — параметры подключения к конкретному источнику ClickHouse.
// В отличие от New(config.CH), параметры приходят из сохранённого источника
// (host/port/protocol/TLS/user/pass), а не из окружения.
type ConnParams struct {
	Host          string
	Port          int
	Protocol      string // "native" | "http"
	TLSEnabled    bool
	TLSSkipVerify bool
	Username      string
	Password      string
	Database      string
}

// Connect открывает read-only подключение к источнику по его параметрам.
// Соединение ленивое: реальный dial происходит при первом Ping/запросе.
func Connect(p ConnParams) (driver.Conn, error) {
	proto := clickhouse.Native
	if p.Protocol == "http" {
		proto = clickhouse.HTTP
	}

	opts := &clickhouse.Options{
		Protocol: proto,
		Addr:     []string{fmt.Sprintf("%s:%d", p.Host, p.Port)},
		Auth: clickhouse.Auth{
			Database: p.Database,
			Username: p.Username,
			Password: p.Password,
		},
		DialTimeout: 5 * time.Second,
		Settings: clickhouse.Settings{
			"max_execution_time": 15,
		},
	}
	if p.TLSEnabled {
		// InsecureSkipVerify управляется явным флагом источника (self-signed в sandbox).
		opts.TLS = &tls.Config{InsecureSkipVerify: p.TLSSkipVerify} //nolint:gosec
	}

	return clickhouse.Open(opts)
}
