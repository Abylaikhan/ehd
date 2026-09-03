package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

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
	views     ViewRepo
	sources   *Service
	log       *zap.Logger
	exportSem chan struct{} // cap 1: одновременно один экспорт в системе (REP-FR export)
}

func NewQueryService(views ViewRepo, sources *Service, log *zap.Logger) *QueryService {
	return &QueryService{views: views, sources: sources, log: log, exportSem: make(chan struct{}, 1)}
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
	snap := buildSnapshot(view, visible, nil, typesByName(cols))

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
	keysetColumns []string
	pageSize      int
	visibleOrder  []string
}

// keysetOf возвращает keyset-колонки snapshot (fallback на одиночную колонку).
func keysetOf(snap domain.PublishedSnapshot) []string {
	if len(snap.KeysetColumns) > 0 {
		return snap.KeysetColumns
	}
	if snap.KeysetColumn != "" {
		return []string{snap.KeysetColumn}
	}
	return nil
}

func keysetTypesOf(snap domain.PublishedSnapshot) []string {
	if len(snap.KeysetTypes) > 0 {
		return snap.KeysetTypes
	}
	if snap.KeysetType != "" {
		return []string{snap.KeysetType}
	}
	return nil
}

// buildPlan валидирует QuerySpec по snapshot и строит план запроса (Принцип 3).
func buildPlan(snap domain.PublishedSnapshot, spec domain.QuerySpec, req Requester, preview bool) (querybuilder.Plan, planMeta, error) {
	keysetCols := keysetOf(snap)
	if len(keysetCols) == 0 {
		return querybuilder.Plan{}, planMeta{}, domain.ErrViewNotConfigured
	}

	// индекс видимых колонок snapshot
	byName := make(map[string]domain.SnapshotColumn, len(snap.Columns))
	visibleOrder := make([]string, 0, len(snap.Columns))
	selectCols := make([]string, 0, len(snap.Columns)+len(keysetCols))
	inSelect := make(map[string]struct{}, len(snap.Columns)+len(keysetCols))
	for _, c := range snap.Columns {
		byName[c.SourceName] = c
		visibleOrder = append(visibleOrder, c.SourceName)
		selectCols = append(selectCols, c.SourceName)
		inSelect[c.SourceName] = struct{}{}
	}
	filters, err := validateFilters(byName, spec)
	if err != nil {
		return querybuilder.Plan{}, planMeta{}, err
	}
	search := buildSearch(snap, spec)
	rs := buildRowScope(snap, req, preview)

	// Порядок = keyset-ключ; при пользовательской сортировке колонка-sortable ставится
	// впереди ключа (keyset остаётся тай-брейкером → корректная пагинация).
	orderCols := keysetCols
	orderTypes := keysetTypesOf(snap)
	if spec.SortColumn != "" {
		col, ok := byName[spec.SortColumn]
		if !ok || !col.Sortable {
			return querybuilder.Plan{}, planMeta{}, domain.ErrQueryValidation
		}
		orderCols = []string{spec.SortColumn}
		orderTypes = []string{col.SourceType}
		for i, k := range keysetCols {
			if k == spec.SortColumn {
				continue
			}
			orderCols = append(orderCols, k)
			if i < len(keysetTypesOf(snap)) {
				orderTypes = append(orderTypes, keysetTypesOf(snap)[i])
			} else {
				orderTypes = append(orderTypes, "")
			}
		}
	}
	// колонки порядка нужны в SELECT для чтения курсора (даже если скрыты)
	for _, k := range orderCols {
		if _, ok := inSelect[k]; !ok {
			selectCols = append(selectCols, k)
			inSelect[k] = struct{}{}
		}
	}

	dir := snap.KeysetDir
	if spec.SortDir == domain.SortAsc || spec.SortDir == domain.SortDesc {
		dir = spec.SortDir
	}
	if dir == "" {
		dir = domain.SortAsc
	}
	cursor, err := decodeCursor(spec.Cursor, orderTypes)
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
		Keyset:     querybuilder.Keyset{Columns: orderCols, Dir: dir, Cursor: cursor},
		Limit:      pageSize + 1, // +1 для определения следующей страницы
	}
	return plan, planMeta{keysetColumns: orderCols, pageSize: pageSize, visibleOrder: visibleOrder}, nil
}

// buildResult нарезает страницу, вычисляет next_cursor и оставляет только видимые колонки.
func buildResult(snap domain.PublishedSnapshot, rows []map[string]any, pm planMeta) domain.QueryResult {
	next := ""
	if len(rows) > pm.pageSize {
		last := rows[pm.pageSize-1]
		vals := make([]string, len(pm.keysetColumns))
		for i, k := range pm.keysetColumns {
			vals[i] = fmt.Sprint(last[k])
		}
		next = encodeCursor(vals)
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

// ExportResult — подготовленные данные для генерации XLSX.
type ExportResult struct {
	Filename string
	Headers  []string
	Rows     [][]any
}

// Export готовит весь отфильтрованный набор (только exportable-колонки) для XLSX (REP-BR-008/009).
func (s *QueryService) Export(ctx context.Context, req Requester, slug string, spec domain.QuerySpec) (ExportResult, error) {
	// одновременно допустим только один экспорт в системе
	select {
	case s.exportSem <- struct{}{}:
		defer func() { <-s.exportSem }()
	default:
		return ExportResult{}, domain.ErrExportBusy
	}

	view, snap, err := s.resolve(ctx, req, slug)
	if err != nil {
		return ExportResult{}, err
	}
	if snap.KeysetColumn == "" {
		return ExportResult{}, domain.ErrViewNotConfigured
	}

	byName := make(map[string]domain.SnapshotColumn, len(snap.Columns))
	var expCols []domain.SnapshotColumn
	for _, c := range snap.Columns {
		byName[c.SourceName] = c
		if c.Exportable {
			expCols = append(expCols, c)
		}
	}
	if len(expCols) == 0 {
		return ExportResult{}, domain.ErrQueryValidation
	}

	filters, err := validateFilters(byName, spec)
	if err != nil {
		return ExportResult{}, err
	}
	search := buildSearch(snap, spec)
	rs := buildRowScope(snap, req, false)

	limit := snap.ExportRowLimit
	if limit <= 0 || limit > domain.MaxExportRows {
		limit = domain.MaxExportRows
	}
	dir := snap.KeysetDir
	if dir == "" {
		dir = domain.SortAsc
	}

	selectCols := make([]string, len(expCols))
	for i, c := range expCols {
		selectCols[i] = c.SourceName
	}
	plan := querybuilder.Plan{
		Database:   snap.DatabaseName,
		Table:      snap.TableName,
		SelectCols: selectCols,
		Filters:    filters,
		Search:     search,
		RowScope:   rs,
		Keyset:     querybuilder.Keyset{Columns: keysetOf(snap), Dir: dir}, // без cursor
		Limit:      limit + 1,                                              // +1 — детект превышения без полного COUNT
	}
	sql, args, err := plan.Build()
	if err != nil {
		return ExportResult{}, domain.ErrQueryValidation
	}
	rows, err := s.sources.RunQuery(ctx, view.DataSourceID, snap.DatabaseName, sql, args)
	if err != nil {
		s.log.Warn("export query failed", zap.String("slug", slug), zap.Error(err))
		return ExportResult{}, domain.ErrSourceUnavailable
	}
	if len(rows) > limit {
		return ExportResult{}, domain.ErrExportTooLarge
	}

	headers := make([]string, len(expCols))
	for i, c := range expCols {
		headers[i] = c.Label
		if headers[i] == "" {
			headers[i] = c.SourceName
		}
	}
	matrix := make([][]any, len(rows))
	for i, r := range rows {
		cells := make([]any, len(expCols))
		for j, c := range expCols {
			cells[j] = cellValue(r[c.SourceName])
		}
		matrix[i] = cells
	}
	return ExportResult{Filename: slug, Headers: headers, Rows: matrix}, nil
}

// --- построение плана (переиспользуется query/count/export) ---

func validateFilters(byName map[string]domain.SnapshotColumn, spec domain.QuerySpec) ([]querybuilder.Filter, error) {
	filters := make([]querybuilder.Filter, 0, len(spec.Filters))
	for _, f := range spec.Filters {
		col, ok := byName[f.Column]
		if !ok || !col.Filterable {
			return nil, domain.ErrQueryValidation
		}
		if !domain.OperatorAllowed(col.DisplayType, f.Operator) {
			return nil, domain.ErrQueryValidation
		}
		if domain.OperatorNeedsValue(f.Operator) && f.Value == nil {
			return nil, domain.ErrQueryValidation
		}
		if f.Operator == domain.OpIn && len(f.Values) == 0 {
			return nil, domain.ErrQueryValidation
		}
		if f.Operator == domain.OpBetween && len(f.Values) != 2 {
			return nil, domain.ErrQueryValidation
		}
		filters = append(filters, querybuilder.Filter{
			Column: col.SourceName, DisplayType: col.DisplayType, Operator: f.Operator,
			Value: f.Value, Values: f.Values,
		})
	}
	return filters, nil
}

func buildSearch(snap domain.PublishedSnapshot, spec domain.QuerySpec) *querybuilder.Search {
	if strings.TrimSpace(spec.Search) == "" {
		return nil
	}
	var scols []string
	for _, c := range snap.Columns {
		if c.Searchable && (c.DisplayType == domain.DisplayText || c.DisplayType == domain.DisplayEnum) {
			scols = append(scols, c.SourceName)
		}
	}
	if len(scols) == 0 {
		return nil
	}
	return &querybuilder.Search{Columns: scols, Term: spec.Search}
}

func buildRowScope(snap domain.PublishedSnapshot, req Requester, preview bool) querybuilder.RowScope {
	if preview || req.IsAdmin || snap.RowScopeMode != domain.RowScopeByProfile {
		return querybuilder.RowScope{}
	}
	return querybuilder.RowScope{
		RegionColumn:     snap.RowScopeRegionColumn,
		Regions:          req.RegionCodes,
		DepartmentColumn: snap.RowScopeDepartmentColumn,
		Departments:      req.DepartmentCodes,
	}
}

// cellValue приводит значение ячейки к простому типу для XLSX (разыменование указателей, NULL→пусто).
func cellValue(v any) any {
	if v == nil {
		return ""
	}
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return ""
		}
		v = rv.Elem().Interface()
	}
	switch x := v.(type) {
	case time.Time:
		return x
	case string:
		return x
	case fmt.Stringer: // decimal.Decimal и т.п.
		return x.String()
	default:
		return v
	}
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

// encodeCursor кодирует значения keyset-ключа последней строки в непрозрачный курсор
// (base64 от JSON-массива строк).
func encodeCursor(vals []string) string {
	b, _ := json.Marshal(vals)
	return base64.RawURLEncoding.EncodeToString(b)
}

// decodeCursor раскодирует курсор в типизированный кортеж значений keyset-ключа.
// Пустой курсор (первая страница) → nil.
func decodeCursor(cursor string, keysetTypes []string) ([]any, error) {
	if cursor == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, err
	}
	var strs []string
	if err := json.Unmarshal(raw, &strs); err != nil {
		return nil, err
	}
	if len(strs) != len(keysetTypes) {
		return nil, errBadCursor
	}
	out := make([]any, len(strs))
	for i, s := range strs {
		out[i] = coerceCursorValue(s, keysetTypes[i])
	}
	return out, nil
}

func coerceCursorValue(s, keysetType string) any {
	base := keysetType
	if i := strings.IndexByte(base, '('); i >= 0 {
		base = base[:i]
	}
	base = strings.TrimPrefix(base, "Nullable(")
	switch {
	case strings.HasPrefix(base, "UInt"):
		if v, err := strconv.ParseUint(s, 10, 64); err == nil {
			return v
		}
	case strings.HasPrefix(base, "Int"):
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			return v
		}
	}
	return s
}

var errBadCursor = errors.New("некорректный курсор")
