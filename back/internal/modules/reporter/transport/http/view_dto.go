package http

import (
	"time"

	"ehd-api/internal/modules/reporter/application"
	"ehd-api/internal/modules/reporter/domain"
)

// --- запросы ---

type createViewReq struct {
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Description  string `json:"description"`
	DataSourceID string `json:"data_source_id"`
	Database     string `json:"database"`
	Table        string `json:"table"`
}

type updateViewReq struct {
	Name                     string `json:"name"`
	Slug                     string `json:"slug"`
	Description              string `json:"description"`
	PageSizeDefault          int    `json:"page_size_default"`
	PageSizeMin              int    `json:"page_size_min"`
	PageSizeMax              int    `json:"page_size_max"`
	DefaultSortColumn        string `json:"default_sort_column"`
	DefaultSortDir           string `json:"default_sort_dir"`
	ExportRowLimit           int    `json:"export_row_limit"`
	RowScopeMode             string `json:"row_scope_mode"`
	KeysetColumn             string `json:"keyset_column"`
	KeysetDir                string `json:"keyset_dir"`
	RowScopeRegionColumn     string `json:"row_scope_region_column"`
	RowScopeDepartmentColumn string `json:"row_scope_department_column"`
}

type columnConfigReq struct {
	SourceName  string `json:"source_name"`
	Label       string `json:"label"`
	DisplayType string `json:"display_type"`
	Position    int    `json:"position"`
	Visible     bool   `json:"visible"`
	Searchable  bool   `json:"searchable"`
	Filterable  bool   `json:"filterable"`
	Sortable    bool   `json:"sortable"`
	Exportable  bool   `json:"exportable"`
	Format      string `json:"format"`
	MaskRule    string `json:"mask_rule"`
	Width       int    `json:"width"`
	NullLabel   string `json:"null_label"`
}

type columnsReq struct {
	Columns []columnConfigReq `json:"columns"`
}

type permissionsReq struct {
	RoleCodes []string `json:"role_codes"`
}

func (r columnsReq) toInput() []application.ColumnConfigInput {
	out := make([]application.ColumnConfigInput, len(r.Columns))
	for i, c := range r.Columns {
		out[i] = application.ColumnConfigInput{
			SourceName: c.SourceName, Label: c.Label, DisplayType: c.DisplayType,
			Position: c.Position, Visible: c.Visible, Searchable: c.Searchable,
			Filterable: c.Filterable, Sortable: c.Sortable, Exportable: c.Exportable,
			Format: c.Format, MaskRule: c.MaskRule, Width: c.Width, NullLabel: c.NullLabel,
		}
	}
	return out
}

// --- ответы ---

type viewResp struct {
	ID                       string     `json:"id"`
	Name                     string     `json:"name"`
	Slug                     string     `json:"slug"`
	Description              string     `json:"description"`
	DataSourceID             string     `json:"data_source_id"`
	DatabaseName             string     `json:"database"`
	TableName                string     `json:"table"`
	SourceMode               string     `json:"source_mode"`
	Status                   string     `json:"status"`
	Revision                 int64      `json:"revision"`
	SchemaHash               string     `json:"schema_hash"`
	PageSizeDefault          int        `json:"page_size_default"`
	PageSizeMin              int        `json:"page_size_min"`
	PageSizeMax              int        `json:"page_size_max"`
	DefaultSortColumn        string     `json:"default_sort_column"`
	DefaultSortDir           string     `json:"default_sort_dir"`
	ExportRowLimit           int        `json:"export_row_limit"`
	RowScopeMode             string     `json:"row_scope_mode"`
	KeysetColumn             string     `json:"keyset_column"`
	KeysetDir                string     `json:"keyset_dir"`
	RowScopeRegionColumn     string     `json:"row_scope_region_column"`
	RowScopeDepartmentColumn string     `json:"row_scope_department_column"`
	PublishedAt              *time.Time `json:"published_at"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

func toViewResp(v domain.DataView) viewResp {
	return viewResp{
		ID: v.ID, Name: v.Name, Slug: v.Slug, Description: v.Description,
		DataSourceID: v.DataSourceID, DatabaseName: v.DatabaseName, TableName: v.TableName,
		SourceMode: v.SourceMode, Status: v.Status, Revision: v.Revision, SchemaHash: v.SchemaHash,
		PageSizeDefault: v.PageSizeDefault, PageSizeMin: v.PageSizeMin, PageSizeMax: v.PageSizeMax,
		DefaultSortColumn: v.DefaultSortColumn, DefaultSortDir: v.DefaultSortDir,
		ExportRowLimit: v.ExportRowLimit, RowScopeMode: v.RowScopeMode,
		KeysetColumn: v.KeysetColumn, KeysetDir: v.KeysetDir,
		RowScopeRegionColumn: v.RowScopeRegionColumn, RowScopeDepartmentColumn: v.RowScopeDepartmentColumn,
		PublishedAt: v.PublishedAt, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt,
	}
}

type viewColumnResp struct {
	SourceName  string `json:"source_name"`
	SourceType  string `json:"source_type"`
	Label       string `json:"label"`
	DisplayType string `json:"display_type"`
	Position    int    `json:"position"`
	Visible     bool   `json:"visible"`
	Searchable  bool   `json:"searchable"`
	Filterable  bool   `json:"filterable"`
	Sortable    bool   `json:"sortable"`
	Exportable  bool   `json:"exportable"`
	Format      string `json:"format"`
	MaskRule    string `json:"mask_rule"`
	Width       int    `json:"width"`
	NullLabel   string `json:"null_label"`
}

func toColumnResp(c domain.ViewColumn) viewColumnResp {
	return viewColumnResp{
		SourceName: c.SourceName, SourceType: c.SourceType, Label: c.Label, DisplayType: c.DisplayType,
		Position: c.Position, Visible: c.Visible, Searchable: c.Searchable, Filterable: c.Filterable,
		Sortable: c.Sortable, Exportable: c.Exportable, Format: c.Format, MaskRule: c.MaskRule,
		Width: c.Width, NullLabel: c.NullLabel,
	}
}

type viewDetailResp struct {
	viewResp
	Columns   []viewColumnResp `json:"columns"`
	RoleCodes []string         `json:"role_codes"`
}

func toViewDetailResp(d application.ViewDetail) viewDetailResp {
	cols := make([]viewColumnResp, len(d.Columns))
	for i, c := range d.Columns {
		cols[i] = toColumnResp(c)
	}
	roles := d.RoleCodes
	if roles == nil {
		roles = []string{}
	}
	return viewDetailResp{viewResp: toViewResp(d.View), Columns: cols, RoleCodes: roles}
}
