package domain

// Виды объектов ClickHouse (REP-FR «Просмотр структуры», п.3).
const (
	ObjectKindTable            = "table"
	ObjectKindView             = "view"
	ObjectKindMaterializedView = "materialized_view"
)

// Database — база данных ClickHouse из интроспекции.
type Database struct {
	Name string
}

// Table — таблица/VIEW/MATERIALIZED VIEW выбранной базы.
type Table struct {
	Name   string
	Engine string
	Kind   string // table | view | materialized_view
}

// Column — колонка таблицы (REP-FR «Просмотр структуры», п.5).
type Column struct {
	Name         string
	Type         string
	Position     uint64
	Nullable     bool
	Comment      string
	InPrimaryKey bool
	InSortingKey bool
}
