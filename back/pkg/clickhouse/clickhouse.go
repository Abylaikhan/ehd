package clickhouse

import (
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"ehd-api/config"
)

// New открывает read-only подключение к источнику ClickHouse.
// Соединение ленивое: реальный dial происходит при первом запросе/Ping.
func New(cfg config.CH) (driver.Conn, error) {
	return clickhouse.Open(&clickhouse.Options{
		Addr: []string{cfg.Addr},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.User,
			Password: cfg.Password,
		},
		DialTimeout: 5 * time.Second,
		Settings: clickhouse.Settings{
			"max_execution_time": 15,
		},
	})
}
