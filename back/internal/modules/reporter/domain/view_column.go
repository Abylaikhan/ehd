package domain

// Правила маскирования значения колонки (ТЗ, mask_rule).
const (
	MaskNone    = "none"
	MaskPartial = "partial"
	MaskFull    = "full"
)

// ViewColumn — настройка колонки представления (ТЗ, «Настройка колонок»).
// SourceName/SourceType берутся из интроспекции и не редактируются.
type ViewColumn struct {
	ID          string
	ViewID      string
	SourceName  string
	SourceType  string
	Label       string
	DisplayType string
	Position    int
	Visible     bool
	Searchable  bool
	Filterable  bool
	Sortable    bool
	Exportable  bool
	Format      string // JSON-конфигурация формата
	MaskRule    string
	Width       int
	NullLabel   string
}
