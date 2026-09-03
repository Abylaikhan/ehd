package clickhouse

import (
	"context"
	"database/sql"
	"regexp"
	"strings"
)

var identOnly = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// SortingKey возвращает ведущие колонки сортировочного ключа таблицы (для keyset-пагинации).
// Берётся префикс из простых идентификаторов; выражение в ключе (напр. toYYYYMM(dt))
// останавливает разбор — используется валидный префикс.
func SortingKey(ctx context.Context, db *sql.DB, database, table string) ([]string, error) {
	var key string
	err := db.QueryRowContext(ctx,
		"SELECT sorting_key FROM system.tables WHERE database = ? AND name = ?", database, table).Scan(&key)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0)
	for _, p := range strings.Split(key, ",") {
		p = strings.TrimSpace(p)
		if !identOnly.MatchString(p) {
			break
		}
		out = append(out, p)
	}
	return out, nil
}

// TableInfo — строка интроспекции таблиц (system.tables).
type TableInfo struct {
	Name   string
	Engine string
	Kind   string // table | view | materialized_view
}

// ColumnInfo — строка интроспекции колонок (system.columns).
type ColumnInfo struct {
	Name         string
	Type         string
	Position     uint64
	Nullable     bool
	Comment      string
	InPrimaryKey bool
	InSortingKey bool
}

// PingSelect1 выполняет безопасный SELECT 1 (REP-FR-011).
func PingSelect1(ctx context.Context, db *sql.DB) error {
	var one uint8
	return db.QueryRowContext(ctx, "SELECT 1").Scan(&one)
}

// ListDatabases возвращает имена всех баз (фильтрация системных — на уровне сервиса).
func ListDatabases(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, "SELECT name FROM system.databases ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// ListTables возвращает таблицы/VIEW/MATERIALIZED VIEW выбранной базы.
// database передаётся типизированным параметром (без конкатенации в SQL).
func ListTables(ctx context.Context, db *sql.DB, database string) ([]TableInfo, error) {
	rows, err := db.QueryContext(ctx,
		"SELECT name, engine FROM system.tables WHERE database = ? ORDER BY name", database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TableInfo
	for rows.Next() {
		var name, engine string
		if err := rows.Scan(&name, &engine); err != nil {
			return nil, err
		}
		out = append(out, TableInfo{Name: name, Engine: engine, Kind: tableKind(engine)})
	}
	return out, rows.Err()
}

// ListColumns возвращает колонки таблицы из system.columns.
func ListColumns(ctx context.Context, db *sql.DB, database, table string) ([]ColumnInfo, error) {
	const q = `SELECT name, type, position, comment, is_in_primary_key, is_in_sorting_key
	           FROM system.columns
	           WHERE database = ? AND table = ?
	           ORDER BY position`
	rows, err := db.QueryContext(ctx, q, database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ColumnInfo
	for rows.Next() {
		var (
			name, chType, comment string
			position              uint64
			inPK, inSort          uint8
		)
		if err := rows.Scan(&name, &chType, &position, &comment, &inPK, &inSort); err != nil {
			return nil, err
		}
		out = append(out, ColumnInfo{
			Name:         name,
			Type:         chType,
			Position:     position,
			Nullable:     strings.HasPrefix(chType, "Nullable("),
			Comment:      comment,
			InPrimaryKey: inPK == 1,
			InSortingKey: inSort == 1,
		})
	}
	return out, rows.Err()
}

// tableKind классифицирует объект по движку ClickHouse.
func tableKind(engine string) string {
	switch engine {
	case "View":
		return "view"
	case "MaterializedView":
		return "materialized_view"
	default:
		return "table"
	}
}
