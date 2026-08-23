package domain

// SnapshotColumn — колонка в опубликованном snapshot (только видимые попадают сюда).
type SnapshotColumn struct {
	SourceName  string `json:"source_name"`
	SourceType  string `json:"source_type"`
	Label       string `json:"label"`
	DisplayType string `json:"display_type"`
	Position    int    `json:"position"`
	Searchable  bool   `json:"searchable"`
	Filterable  bool   `json:"filterable"`
	Sortable    bool   `json:"sortable"`
	Exportable  bool   `json:"exportable"`
	Format      string `json:"format,omitempty"`
	MaskRule    string `json:"mask_rule,omitempty"`
	NullLabel   string `json:"null_label,omitempty"`
}

// PublishedSnapshot — неизменяемая конфигурация, зафиксированная при публикации.
// Именно её читает Query Engine и конечный пользователь (REP-FR-042, REP-BR-007).
type PublishedSnapshot struct {
	DatabaseName      string           `json:"database_name"`
	TableName         string           `json:"table_name"`
	SchemaHash        string           `json:"schema_hash"`
	PageSizeDefault   int              `json:"page_size_default"`
	PageSizeMin       int              `json:"page_size_min"`
	PageSizeMax       int              `json:"page_size_max"`
	DefaultSortColumn string           `json:"default_sort_column,omitempty"`
	DefaultSortDir    string           `json:"default_sort_dir,omitempty"`
	ExportRowLimit    int              `json:"export_row_limit"`
	RowScopeMode      string           `json:"row_scope_mode"`
	RoleCodes         []string         `json:"role_codes"`
	Columns           []SnapshotColumn `json:"columns"`
}
