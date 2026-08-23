package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"ehd-api/internal/modules/reporter/domain"
	"ehd-api/internal/modules/reporter/querybuilder"
)

// Requester — доверенный контекст пользователя (маппинг auth/contract.Identity на границе транспорта).
type Requester struct {
	UserID          string
	IsAdmin         bool
	RoleCodes       []string
	RegionCodes     []string
	DepartmentCodes []string
}

// QueryService — безопасное выполнение запросов к данным по опубликованному представлению.
type QueryService struct {
	views   ViewRepo
	sources *Service
	log     *zap.Logger
}

func NewQueryService(views ViewRepo, sources *Service, log *zap.Logger) *QueryService {
	return &QueryService{views: views, sources: sources, log: log}
}

// ViewColumnMeta — колонка представления для рендера (с разрешёнными операциями).
type ViewColumnMeta struct {
	SourceName  string
	Label       string
	DisplayType string
	Searchable  bool
	Filterable  bool
	Sortable    bool
	Operators   []string
}

// ViewMeta — метаданные представления для пользователя.
type ViewMeta struct {
	Slug            string
	Name            string
	Description     string
	PageSizeDefault int
	PageSizeMin     int
	PageSizeMax     int
	Columns         []ViewColumnMeta
}

// UserViewItem — элемент списка доступных пользователю представлений.
type UserViewItem struct {
	Slug        string
	Name        string
	Description string
}

// ListForUser возвращает опубликованные представления, доступные пользователю по ролям (REP-FR-050).
func (s *QueryService) ListForUser(ctx context.Context, req Requester) ([]UserViewItem, error) {
	views, err := s.views.ListViews(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]UserViewItem, 0, len(views))
	for _, v := range views {
		switch v.Status {
		case domain.ViewStatusDisabled, domain.ViewStatusSchemaError, domain.ViewStatusArchived:
			continue
		}
		raw, err := s.views.GetSnapshot(ctx, v.ID)
		if err != nil || strings.TrimSpace(raw) == "" {
			continue // не опубликовано
		}
		var snap domain.PublishedSnapshot
		if err := json.Unmarshal([]byte(raw), &snap); err != nil {
			continue
		}
		if !req.IsAdmin && !intersects(req.RoleCodes, snap.RoleCodes) {
			continue
		}
		out = append(out, UserViewItem{Slug: v.Slug, Name: v.Name, Description: v.Description})
	}
	return out, nil
}

// Meta возвращает метаданные опубликованного представления (REP-FR-051/052).
func (s *QueryService) Meta(ctx context.Context, req Requester, slug string) (ViewMeta, error) {
	view, snap, err := s.resolve(ctx, req, slug)
	if err != nil {
		return ViewMeta{}, err
	}
	cols := make([]ViewColumnMeta, len(snap.Columns))
	for i, c := range snap.Columns {
		cols[i] = ViewColumnMeta{
			SourceName: c.SourceName, Label: c.Label, DisplayType: c.DisplayType,
			Searchable: c.Searchable, Filterable: c.Filterable, Sortable: c.Sortable,
			Operators: domain.AllowedOperators(c.DisplayType),
		}
	}
	return ViewMeta{
		Slug: view.Slug, Name: view.Name, Description: view.Description,
		PageSizeDefault: snap.PageSizeDefault, PageSizeMin: snap.PageSizeMin, PageSizeMax: snap.PageSizeMax,
		Columns: cols,
	}, nil
}

// Query выполняет запрос данных с фильтрами/поиском/keyset и применяет RLS (REP-FR-050+).
func (s *QueryService) Query(ctx context.Context, req Requester, slug string, spec domain.QuerySpec) (domain.QueryResult, error) {
	view, snap, err := s.resolve(ctx, req, slug)
	if err != nil {
		return domain.QueryResult{}, err
	}
	plan, pm, err := buildPlan(snap, spec, req, false)
	if err != nil {
		return domain.QueryResult{}, err
	}
	sql, args, err := plan.Build()
	if err != nil {
		return domain.QueryResult{}, domain.ErrQueryValidation
	}
	rows, err := s.sources.RunQuery(ctx, view.DataSourceID, snap.DatabaseName, sql, args)
	if err != nil {
		s.log.Warn("query execution failed", zap.String("slug", slug), zap.Error(err))
		return domain.QueryResult{}, domain.ErrSourceUnavailable
	}
	return buildResult(snap, rows, pm), nil
}

// Count возвращает точный total_count с теми же фильтрами и RLS (REP-FR total_count).
func (s *QueryService) Count(ctx context.Context, req Requester, slug string, spec domain.QuerySpec) (uint64, error) {
	view, snap, err := s.resolve(ctx, req, slug)
	if err != nil {
		return 0, err
	}
	plan, _, err := buildPlan(snap, spec, req, false)
	if err != nil {
		return 0, err
	}
	sql, args, err := plan.BuildCount()
	if err != nil {
		return 0, domain.ErrQueryValidation
	}
	n, err := s.sources.ScalarCount(ctx, view.DataSourceID, snap.DatabaseName, sql, args)
	if err != nil {
		return 0, domain.ErrSourceUnavailable
	}
	return n, nil
}

// PreviewDraft — админ-предпросмотр текущей конфигурации черновика без RLS (REP-FR-040).
func (s *QueryService) PreviewDraft(ctx context.Context, id string, spec domain.QuerySpec) (domain.QueryResult, error) {
	view, err := s.views.GetView(ctx, id)
	if err != nil {
		return domain.QueryResult{}, err
	}
	src, err := s.sources.GetSource(ctx, view.DataSourceID)
	if err != nil {
		return domain.QueryResult{}, err
	}
	if src.Status != domain.SourceStatusActive {
		return domain.QueryResult{}, domain.ErrSourceUnavailable
	}
	cols, err := s.views.GetColumns(ctx, id)
	if err != nil {
		return domain.QueryResult{}, err
	}
	visible := visibleColumns(cols)
	if len(visible) == 0 {
		return domain.QueryResult{}, domain.ErrQueryValidation
	}
	keysetType := ""
	for _, c := range cols {
		if c.SourceName == view.KeysetColumn {
			keysetType = c.SourceType
			break
		}
	}
	snap := buildSnapshot(view, visible, nil, keysetType)

	plan, pm, err := buildPlan(snap, spec, Requester{IsAdmin: true}, true)
	if err != nil {
		return domain.QueryResult{}, err
	}
	sql, args, err := plan.Build()
	if err != nil {
		return domain.QueryResult{}, domain.ErrQueryValidation
	}
	rows, err := s.sources.RunQuery(ctx, view.DataSourceID, snap.DatabaseName, sql, args)
	if err != nil {
		return domain.QueryResult{}, domain.ErrSourceUnavailable
	}
	return buildResult(snap, rows, pm), nil
}

// resolve находит опубликованное представление по slug и проверяет доступ и источник.
func (s *QueryService) resolve(ctx context.Context, req Requester, slug string) (domain.DataView, domain.PublishedSnapshot, error) {
	view, err := s.views.GetViewBySlug(ctx, slug)
	if err != nil {
		return domain.DataView{}, domain.PublishedSnapshot{}, err
	}
	// отключённое/архивное/со schema_error — не обслуживаем (REP-FR-044)
	switch view.Status {
	case domain.ViewStatusDisabled, domain.ViewStatusSchemaError, domain.ViewStatusArchived:
		return domain.DataView{}, domain.PublishedSnapshot{}, domain.ErrViewNotFound
	}
	raw, err := s.views.GetSnapshot(ctx, view.ID)
	if err != nil {
		return domain.DataView{}, domain.PublishedSnapshot{}, err
	}
	if strings.TrimSpace(raw) == "" {
		return domain.DataView{}, domain.PublishedSnapshot{}, domain.ErrViewNotFound // ещё не публиковалось
	}
	var snap domain.PublishedSnapshot
	if err := json.Unmarshal([]byte(raw), &snap); err != nil {
		return domain.DataView{}, domain.PublishedSnapshot{}, err
	}
	// RBAC: роль пользователя должна давать доступ (или админ) — REP-FR-053
	if !req.IsAdmin && !intersects(req.RoleCodes, snap.RoleCodes) {
		return domain.DataView{}, domain.PublishedSnapshot{}, domain.ErrAccessDenied
	}
	// источник должен быть активен
	src, err := s.sources.GetSource(ctx, view.DataSourceID)
	if err != nil {
		return domain.DataView{}, domain.PublishedSnapshot{}, err
	}
	if src.Status != domain.SourceStatusActive {
		return domain.DataView{}, domain.PublishedSnapshot{}, domain.ErrSourceUnavailable
	}
	return view, snap, nil
}

// planMeta — вспомогательные данные для формирования ответа.
type planMeta struct {
	keysetColumn string
	pageSize     int
	visibleOrder []string
}

// buildPlan валидирует QuerySpec по snapshot и строит план запроса (Принцип 3).
func buildPlan(snap domain.PublishedSnapshot, spec domain.QuerySpec, req Requester, preview bool) (querybuilder.Plan, planMeta, error) {
	if snap.KeysetColumn == "" {
		return querybuilder.Plan{}, planMeta{}, domain.ErrViewNotConfigured
	}

	// индекс видимых колонок snapshot
	byName := make(map[string]domain.SnapshotColumn, len(snap.Columns))
	visibleOrder := make([]string, 0, len(snap.Columns))
	selectCols := make([]string, 0, len(snap.Columns)+1)
	for _, c := range snap.Columns {
		byName[c.SourceName] = c
		visibleOrder = append(visibleOrder, c.SourceName)
		selectCols = append(selectCols, c.SourceName)
	}
	// keyset-колонка нужна в SELECT для чтения курсора (даже если скрыта)
	if _, ok := byName[snap.KeysetColumn]; !ok {
		selectCols = append(selectCols, snap.KeysetColumn)
	}

	// фильтры: только whitelist + filterable + допустимый оператор
	filters := make([]querybuilder.Filter, 0, len(spec.Filters))
	for _, f := range spec.Filters {
		col, ok := byName[f.Column]
		if !ok || !col.Filterable {
			return querybuilder.Plan{}, planMeta{}, domain.ErrQueryValidation
		}
		if !domain.OperatorAllowed(col.DisplayType, f.Operator) {
			return querybuilder.Plan{}, planMeta{}, domain.ErrQueryValidation
		}
		if domain.OperatorNeedsValue(f.Operator) && f.Value == nil {
			return querybuilder.Plan{}, planMeta{}, domain.ErrQueryValidation
		}
		if f.Operator == domain.OpIn && len(f.Values) == 0 {
			return querybuilder.Plan{}, planMeta{}, domain.ErrQueryValidation
		}
		if f.Operator == domain.OpBetween && len(f.Values) != 2 {
			return querybuilder.Plan{}, planMeta{}, domain.ErrQueryValidation
		}
		filters = append(filters, querybuilder.Filter{
			Column: col.SourceName, DisplayType: col.DisplayType, Operator: f.Operator,
			Value: f.Value, Values: f.Values,
		})
	}

	// поиск: только searchable строковые колонки
	var search *querybuilder.Search
	if strings.TrimSpace(spec.Search) != "" {
		var scols []string
		for _, c := range snap.Columns {
			if c.Searchable && (c.DisplayType == domain.DisplayText || c.DisplayType == domain.DisplayEnum) {
				scols = append(scols, c.SourceName)
			}
		}
		if len(scols) > 0 {
			search = &querybuilder.Search{Columns: scols, Term: spec.Search}
		}
	}

	// RLS: только by_profile и не админ и не preview (REP-FR-11..14)
	var rs querybuilder.RowScope
	if !preview && !req.IsAdmin && snap.RowScopeMode == domain.RowScopeByProfile {
		rs = querybuilder.RowScope{
			RegionColumn:     snap.RowScopeRegionColumn,
			Regions:          req.RegionCodes,
			DepartmentColumn: snap.RowScopeDepartmentColumn,
			Departments:      req.DepartmentCodes,
		}
	}

	// keyset
	dir := snap.KeysetDir
	if spec.SortDir == domain.SortAsc || spec.SortDir == domain.SortDesc {
		dir = spec.SortDir
	}
	if dir == "" {
		dir = domain.SortAsc
	}
	cursor, err := decodeCursor(spec.Cursor, snap.KeysetType)
	if err != nil {
		return querybuilder.Plan{}, planMeta{}, domain.ErrQueryValidation
	}

	pageSize := clampPageSize(spec.PageSize, snap.PageSizeMin, snap.PageSizeDefault, snap.PageSizeMax)

	plan := querybuilder.Plan{
		Database:   snap.DatabaseName,
		Table:      snap.TableName,
		SelectCols: selectCols,
		Filters:    filters,
		Search:     search,
		RowScope:   rs,
		Keyset:     querybuilder.Keyset{Column: snap.KeysetColumn, Dir: dir, Cursor: cursor},
		Limit:      pageSize + 1, // +1 для определения следующей страницы
	}
	return plan, planMeta{keysetColumn: snap.KeysetColumn, pageSize: pageSize, visibleOrder: visibleOrder}, nil
}

// buildResult нарезает страницу, вычисляет next_cursor и оставляет только видимые колонки.
func buildResult(snap domain.PublishedSnapshot, rows []map[string]any, pm planMeta) domain.QueryResult {
	next := ""
	if len(rows) > pm.pageSize {
		last := rows[pm.pageSize-1]
		if v, ok := last[pm.keysetColumn]; ok {
			next = fmt.Sprint(v)
		}
		rows = rows[:pm.pageSize]
	}

	outRows := make([]map[string]any, len(rows))
	for i, r := range rows {
		m := make(map[string]any, len(pm.visibleOrder))
		for _, name := range pm.visibleOrder {
			m[name] = r[name] // только видимые колонки (скрытые/keyset не попадают)
		}
		outRows[i] = m
	}

	cols := make([]domain.ResultColumn, len(snap.Columns))
	for i, c := range snap.Columns {
		cols[i] = domain.ResultColumn{SourceName: c.SourceName, Label: c.Label, DisplayType: c.DisplayType}
	}
	return domain.QueryResult{Columns: cols, Rows: outRows, NextCursor: next, PageSize: pm.pageSize}
}

// --- helpers ---

func intersects(a, b []string) bool {
	set := make(map[string]struct{}, len(a))
	for _, x := range a {
		set[x] = struct{}{}
	}
	for _, y := range b {
		if _, ok := set[y]; ok {
			return true
		}
	}
	return false
}

func clampPageSize(n, min, def, max int) int {
	if min <= 0 {
		min = domain.MinPageSize
	}
	if max <= 0 {
		max = domain.MaxPageSize
	}
	if def <= 0 {
		def = domain.DefaultPageSize
	}
	if n <= 0 {
		n = def
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

func decodeCursor(cursor, keysetType string) (any, error) {
	if cursor == "" {
		return nil, nil
	}
	base := keysetType
	if i := strings.IndexByte(base, '('); i >= 0 {
		base = base[:i]
	}
	base = strings.TrimPrefix(base, "Nullable(")
	switch {
	case strings.HasPrefix(base, "UInt"):
		return strconv.ParseUint(cursor, 10, 64)
	case strings.HasPrefix(base, "Int"):
		return strconv.ParseInt(cursor, 10, 64)
	default:
		return cursor, nil
	}
}
