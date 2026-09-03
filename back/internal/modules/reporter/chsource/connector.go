// Package chsource — driven-адаптер: реализует application.Connector поверх
// pkg/clickhouse (database/sql) и маппит результаты интроспекции в доменные типы Reporter.
package chsource

import (
	"context"
	"database/sql"
	"reflect"

	"ehd-api/internal/modules/reporter/application"
	"ehd-api/internal/modules/reporter/domain"
	"ehd-api/pkg/clickhouse"
)

// Connector открывает подключения к источникам ClickHouse.
type Connector struct{}

func New() Connector { return Connector{} }

// Open устанавливает подключение по параметрам источника.
func (Connector) Open(_ context.Context, p application.ConnParams) (application.SourceConn, error) {
	db, err := clickhouse.Connect(clickhouse.ConnParams{
		Host:          p.Host,
		Port:          p.Port,
		Protocol:      p.Protocol,
		TLSEnabled:    p.TLSEnabled,
		TLSSkipVerify: p.TLSSkipVerify,
		Username:      p.Username,
		Password:      p.Password,
		Database:      p.Database,
	})
	if err != nil {
		return nil, err
	}
	return &sourceConn{db: db}, nil
}

// sourceConn оборачивает *sql.DB и отдаёт доменные типы.
type sourceConn struct{ db *sql.DB }

func (c *sourceConn) Ping(ctx context.Context) error {
	return clickhouse.PingSelect1(ctx, c.db)
}

func (c *sourceConn) Databases(ctx context.Context) ([]string, error) {
	return clickhouse.ListDatabases(ctx, c.db)
}

func (c *sourceConn) Tables(ctx context.Context, db string) ([]domain.Table, error) {
	rows, err := clickhouse.ListTables(ctx, c.db, db)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Table, len(rows))
	for i, t := range rows {
		out[i] = domain.Table{Name: t.Name, Engine: t.Engine, Kind: t.Kind}
	}
	return out, nil
}

func (c *sourceConn) Columns(ctx context.Context, db, table string) ([]domain.Column, error) {
	rows, err := clickhouse.ListColumns(ctx, c.db, db, table)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Column, len(rows))
	for i, col := range rows {
		out[i] = domain.Column{
			Name:         col.Name,
			Type:         col.Type,
			Position:     col.Position,
			Nullable:     col.Nullable,
			Comment:      col.Comment,
			InPrimaryKey: col.InPrimaryKey,
			InSortingKey: col.InSortingKey,
		}
	}
	return out, nil
}

func (c *sourceConn) SortingKey(ctx context.Context, db, table string) ([]string, error) {
	return clickhouse.SortingKey(ctx, c.db, db, table)
}

// Query выполняет параметризованный SELECT и возвращает строки map[колонка]значение.
// Скан динамический по ScanType драйвера (clickhouse-go не поддерживает *interface{}).
func (c *sourceConn) Query(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	types, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	var out []map[string]any
	for rows.Next() {
		ptrs := make([]any, len(cols))
		for i := range ptrs {
			st := types[i].ScanType()
			if st == nil {
				var v any
				ptrs[i] = &v
			} else {
				ptrs[i] = reflect.New(st).Interface()
			}
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		m := make(map[string]any, len(cols))
		for i, name := range cols {
			m[name] = reflect.ValueOf(ptrs[i]).Elem().Interface()
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ScalarUint64 выполняет запрос, возвращающий одно целое (например, count()).
func (c *sourceConn) ScalarUint64(ctx context.Context, query string, args ...any) (uint64, error) {
	var n uint64
	if err := c.db.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (c *sourceConn) Close() error { return c.db.Close() }
