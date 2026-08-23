package http

import (
	"github.com/gofiber/fiber/v2"

	"ehd-api/internal/modules/auth/contract"
	"ehd-api/internal/modules/reporter/application"
	"ehd-api/internal/modules/reporter/domain"
)

// --- запросы ---

type filterReq struct {
	Column   string `json:"column"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
	Values   []any  `json:"values"`
}

type querySpecReq struct {
	Filters  []filterReq `json:"filters"`
	Search   string      `json:"search"`
	Sort     sortReq     `json:"sort"`
	PageSize int         `json:"page_size"`
	Cursor   string      `json:"cursor"`
}

type sortReq struct {
	Dir string `json:"dir"`
}

func (q querySpecReq) toDomain() domain.QuerySpec {
	filters := make([]domain.Filter, len(q.Filters))
	for i, f := range q.Filters {
		filters[i] = domain.Filter{Column: f.Column, Operator: f.Operator, Value: f.Value, Values: f.Values}
	}
	return domain.QuerySpec{
		Filters:  filters,
		Search:   q.Search,
		SortDir:  q.Sort.Dir,
		PageSize: q.PageSize,
		Cursor:   q.Cursor,
	}
}

// requesterFrom извлекает доверенный контекст пользователя из сессии (проставлен guard'ом).
func requesterFrom(c *fiber.Ctx) application.Requester {
	id, _ := c.Locals(identityKey).(contract.Identity)
	return application.Requester{
		UserID: id.UserID, IsAdmin: id.IsAdmin, RoleCodes: id.RoleCodes,
		RegionCodes: id.RegionCodes, DepartmentCodes: id.DepartmentCodes,
	}
}

// --- ответы ---

type columnMetaResp struct {
	SourceName  string   `json:"source_name"`
	Label       string   `json:"label"`
	DisplayType string   `json:"display_type"`
	Searchable  bool     `json:"searchable"`
	Filterable  bool     `json:"filterable"`
	Sortable    bool     `json:"sortable"`
	Operators   []string `json:"operators"`
}

type viewMetaResp struct {
	Slug            string           `json:"slug"`
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	PageSizeDefault int              `json:"page_size_default"`
	PageSizeMin     int              `json:"page_size_min"`
	PageSizeMax     int              `json:"page_size_max"`
	Columns         []columnMetaResp `json:"columns"`
}

func toViewMetaResp(m application.ViewMeta) viewMetaResp {
	cols := make([]columnMetaResp, len(m.Columns))
	for i, c := range m.Columns {
		ops := c.Operators
		if ops == nil {
			ops = []string{}
		}
		cols[i] = columnMetaResp{
			SourceName: c.SourceName, Label: c.Label, DisplayType: c.DisplayType,
			Searchable: c.Searchable, Filterable: c.Filterable, Sortable: c.Sortable, Operators: ops,
		}
	}
	return viewMetaResp{
		Slug: m.Slug, Name: m.Name, Description: m.Description,
		PageSizeDefault: m.PageSizeDefault, PageSizeMin: m.PageSizeMin, PageSizeMax: m.PageSizeMax,
		Columns: cols,
	}
}

func toQueryResultResp(r domain.QueryResult) fiber.Map {
	rows := r.Rows
	if rows == nil {
		rows = []map[string]any{}
	}
	cols := make([]fiber.Map, len(r.Columns))
	for i, c := range r.Columns {
		cols[i] = fiber.Map{"source_name": c.SourceName, "label": c.Label, "display_type": c.DisplayType}
	}
	return fiber.Map{
		"columns": cols,
		"rows":    rows,
		"page":    fiber.Map{"page_size": r.PageSize, "next_cursor": r.NextCursor},
	}
}
