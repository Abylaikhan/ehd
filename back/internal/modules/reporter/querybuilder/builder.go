package querybuilder

import (
	"strconv"
	"strings"
)

// Search — глобальный поиск по строковым searchable-колонкам.
type Search struct {
	Columns []string // только searchable=true строковые колонки
	Term    string
}

// RowScope — ограничение строк по профилю (RLS). Пустые поля не порождают предикатов.
type RowScope struct {
	RegionColumn     string
	Regions          []string
	DepartmentColumn string
	Departments      []string
}

// Keyset — стабильный ключ для cursor-пагинации.
type Keyset struct {
	Column string
	Dir    string // asc|desc
	Cursor any    // nil на первой странице
}

// Plan — полностью проверенный план запроса (колонки уже из whitelist).
type Plan struct {
	Database   string
	Table      string
	SelectCols []string
	Filters    []Filter
	Search     *Search
	RowScope   RowScope
	Keyset     Keyset
	Limit      int
}

// conditions собирает предикаты фильтров, поиска и RLS (без keyset — общие для SELECT и COUNT).
func (p Plan) conditions() ([]string, []any, error) {
	var conds []string
	var args []any

	for _, f := range p.Filters {
		frag, fargs, err := f.fragment()
		if err != nil {
			return nil, nil, err
		}
		conds = append(conds, frag)
		args = append(args, fargs...)
	}

	if p.Search != nil && strings.TrimSpace(p.Search.Term) != "" && len(p.Search.Columns) > 0 {
		var ors []string
		for _, c := range p.Search.Columns {
			if !SafeIdent(c) {
				return nil, nil, ErrUnsafeIdentifier
			}
			ors = append(ors, quoteIdent(c)+" ILIKE ?")
			args = append(args, "%"+p.Search.Term+"%")
		}
		conds = append(conds, "("+strings.Join(ors, " OR ")+")")
	}

	// RLS: предикат добавляется только для измерения с заданной колонкой и непустым списком (REP-FR-11,12).
	if p.RowScope.RegionColumn != "" && len(p.RowScope.Regions) > 0 {
		if !SafeIdent(p.RowScope.RegionColumn) {
			return nil, nil, ErrUnsafeIdentifier
		}
		rc := quoteIdent(p.RowScope.RegionColumn)
		conds = append(conds, rc+" IN ? AND "+rc+" IS NOT NULL")
		args = append(args, p.RowScope.Regions)
	}
	if p.RowScope.DepartmentColumn != "" && len(p.RowScope.Departments) > 0 {
		if !SafeIdent(p.RowScope.DepartmentColumn) {
			return nil, nil, ErrUnsafeIdentifier
		}
		dc := quoteIdent(p.RowScope.DepartmentColumn)
		conds = append(conds, dc+" IN ? AND "+dc+" IS NOT NULL")
		args = append(args, p.RowScope.Departments)
	}

	return conds, args, nil
}

func (p Plan) from() (string, error) {
	if !SafeIdent(p.Database) || !SafeIdent(p.Table) {
		return "", ErrUnsafeIdentifier
	}
	return quoteIdent(p.Database) + "." + quoteIdent(p.Table), nil
}

// Build формирует SELECT видимых колонок с WHERE (фильтры+поиск+RLS+keyset), ORDER BY и LIMIT.
func (p Plan) Build() (string, []any, error) {
	if len(p.SelectCols) == 0 {
		return "", nil, ErrNoColumns
	}
	from, err := p.from()
	if err != nil {
		return "", nil, err
	}
	sel := make([]string, len(p.SelectCols))
	for i, c := range p.SelectCols {
		if !SafeIdent(c) {
			return "", nil, ErrUnsafeIdentifier
		}
		sel[i] = quoteIdent(c)
	}

	conds, args, err := p.conditions()
	if err != nil {
		return "", nil, err
	}

	// keyset-предикат
	if p.Keyset.Column != "" && p.Keyset.Cursor != nil {
		if !SafeIdent(p.Keyset.Column) {
			return "", nil, ErrUnsafeIdentifier
		}
		op := ">"
		if p.Keyset.Dir == "desc" {
			op = "<"
		}
		conds = append(conds, quoteIdent(p.Keyset.Column)+" "+op+" ?")
		args = append(args, p.Keyset.Cursor)
	}

	var sb strings.Builder
	sb.WriteString("SELECT ")
	sb.WriteString(strings.Join(sel, ", "))
	sb.WriteString(" FROM ")
	sb.WriteString(from)
	if len(conds) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(conds, " AND "))
	}
	if p.Keyset.Column != "" {
		if !SafeIdent(p.Keyset.Column) {
			return "", nil, ErrUnsafeIdentifier
		}
		dir := "ASC"
		if p.Keyset.Dir == "desc" {
			dir = "DESC"
		}
		sb.WriteString(" ORDER BY ")
		sb.WriteString(quoteIdent(p.Keyset.Column))
		sb.WriteString(" ")
		sb.WriteString(dir)
	}
	sb.WriteString(" LIMIT ")
	sb.WriteString(strconv.Itoa(p.Limit))

	return sb.String(), args, nil
}

// BuildCount формирует точный COUNT с теми же WHERE (фильтры+поиск+RLS), без keyset/limit (FR-15).
func (p Plan) BuildCount() (string, []any, error) {
	from, err := p.from()
	if err != nil {
		return "", nil, err
	}
	conds, args, err := p.conditions()
	if err != nil {
		return "", nil, err
	}
	sql := "SELECT count() FROM " + from
	if len(conds) > 0 {
		sql += " WHERE " + strings.Join(conds, " AND ")
	}
	return sql, args, nil
}
