// Package chsource — driven-адаптер: реализует application.Connector поверх
// pkg/clickhouse и маппит результаты интроспекции в доменные типы Reporter.
package chsource

import (
	"context"
	"reflect"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"ehd-api/internal/modules/reporter/application"
	"ehd-api/internal/modules/reporter/domain"
	"ehd-api/pkg/clickhouse"
)

// Connector открывает подключения к источникам ClickHouse.
type Connector struct{}

func New() Connector { return Connector{} }

// Open устанавливает подключение по параметрам источника.
func (Connector) Open(_ context.Context, p application.ConnParams) (application.SourceConn, error) {
	conn, err := clickhouse.Connect(clickhouse.ConnParams{
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
	return &sourceConn{conn: conn}, nil
}

// sourceConn оборачивает driver.Conn и отдаёт доменные типы.
type sourceConn struct{ conn driver.Conn }

func (c *sourceConn) Ping(ctx context.Context) error {
	return clickhouse.PingSelect1(ctx, c.conn)
}

func (c *sourceConn) Databases(ctx context.Context) ([]string, error) {
	return clickhouse.ListDatabases(ctx, c.conn)
}

func (c *sourceConn) Tables(ctx context.Context, db string) ([]domain.Table, error) {
	rows, err := clickhouse.ListTables(ctx, c.conn, db)
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
	rows, err := clickhouse.ListColumns(ctx, c.conn, db, table)
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

// Query выполняет параметризованный SELECT и возвращает строки как map[колонка]значение.
// Значения сканируются динамически по типам ответа ClickHouse.
func (c *sourceConn) Query(ctx context.Context, sql string, args ...any) ([]map[string]any, error) {
	rows, err := c.conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols := rows.Columns()
	types := rows.ColumnTypes()
	var out []map[string]any
	for rows.Next() {
		// типизированный скан: clickhouse-go не поддерживает *interface{},
		// поэтому под каждую колонку выделяем указатель её ScanType.
		ptrs := make([]any, len(cols))
		for i := range ptrs {
			ptrs[i] = reflect.New(types[i].ScanType()).Interface()
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
func (c *sourceConn) ScalarUint64(ctx context.Context, sql string, args ...any) (uint64, error) {
	var n uint64
	if err := c.conn.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (c *sourceConn) Close() error { return c.conn.Close() }
