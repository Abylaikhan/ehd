package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"ehd-api/internal/modules/reporter/domain"
)

// ViewRepo — хранилище представлений, колонок и прав.
type ViewRepo interface {
	CreateView(ctx context.Context, v domain.DataView, cols []domain.ViewColumn) (domain.DataView, error)
	GetView(ctx context.Context, id string) (domain.DataView, error)
	ListViews(ctx context.Context) ([]domain.DataView, error)
	UpdateViewMeta(ctx context.Context, v domain.DataView) (domain.DataView, error)
	SetStatus(ctx context.Context, id, status string) error
	DeleteView(ctx context.Context, id string) error
	SlugExists(ctx context.Context, slug, excludeID string) (bool, error)
	GetColumns(ctx context.Context, viewID string) ([]domain.ViewColumn, error)
	SaveColumns(ctx context.Context, cols []domain.ViewColumn) error
	ReplaceColumns(ctx context.Context, viewID string, cols []domain.ViewColumn) error
	GetPermissions(ctx context.Context, viewID string) ([]string, error)
	SetPermissions(ctx context.Context, viewID string, roleCodes []string) error
	Publish(ctx context.Context, viewID, snapshotJSON, schemaHash string, at time.Time) error
	GetSnapshot(ctx context.Context, viewID string) (string, error)
}

// SourceInspector — доступ к источнику для проверки статуса и интроспекции колонок (slice 003).
type SourceInspector interface {
	GetSource(ctx context.Context, id string) (domain.DataSource, error)
	Columns(ctx context.Context, id, db, table string) ([]domain.Column, error)
}

// ViewService — сценарии управления представлениями.
type ViewService struct {
	repo    ViewRepo
	sources SourceInspector
	log     *zap.Logger
}

func NewViewService(repo ViewRepo, sources SourceInspector, log *zap.Logger) *ViewService {
	return &ViewService{repo: repo, sources: sources, log: log}
}

// ViewDetail — представление вместе с колонками и правами.
type ViewDetail struct {
	View      domain.DataView
	Columns   []domain.ViewColumn
	RoleCodes []string
}

// CreateViewInput — данные создания черновика.
type CreateViewInput struct {
	Name         string
	Slug         string
	Description  string
	DataSourceID string
	Database     string
	Table        string
}

// CreateView создаёт черновик и автоматически загружает колонки из интроспекции (REP-FR: «Создание Data View»).
func (s *ViewService) CreateView(ctx context.Context, in CreateViewInput) (domain.DataView, error) {
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	if strings.TrimSpace(in.Name) == "" || !domain.ValidSlug(in.Slug) ||
		strings.TrimSpace(in.Database) == "" || strings.TrimSpace(in.Table) == "" ||
		strings.TrimSpace(in.DataSourceID) == "" {
		return domain.DataView{}, domain.ErrViewValidation
	}

	src, err := s.sources.GetSource(ctx, in.DataSourceID)
	if err != nil {
		return domain.DataView{}, err
	}
	if src.Status != domain.SourceStatusActive {
		return domain.DataView{}, domain.ErrSourceInactive
	}

	taken, err := s.repo.SlugExists(ctx, in.Slug, "")
	if err != nil {
		return domain.DataView{}, err
	}
	if taken {
		return domain.DataView{}, domain.ErrSlugTaken
	}

	cols, err := s.sources.Columns(ctx, in.DataSourceID, in.Database, in.Table)
	if err != nil {
		return domain.DataView{}, err
	}
	if len(cols) == 0 {
		return domain.DataView{}, domain.ErrTableNotFound
	}

	viewCols := make([]domain.ViewColumn, len(cols))
	for i, c := range cols {
		viewCols[i] = domain.ViewColumn{
			SourceName:  c.Name,
			SourceType:  c.Type,
			Label:       c.Name,
			DisplayType: domain.DisplayTypeFor(c.Type),
			Position:    int(c.Position),
			Visible:     false,
			MaskRule:    domain.MaskNone,
			Format:      "{}",
		}
	}

	view := domain.DataView{
		Name:            in.Name,
		Slug:            in.Slug,
		Description:     in.Description,
		DataSourceID:    in.DataSourceID,
		DatabaseName:    in.Database,
		TableName:       in.Table,
		PageSizeDefault: domain.DefaultPageSize,
		PageSizeMin:     domain.MinPageSize,
		PageSizeMax:     domain.MaxPageSize,
		ExportRowLimit:  domain.MaxExportRows,
		RowScopeMode:    domain.RowScopeByProfile,
	}
	return s.repo.CreateView(ctx, view, viewCols)
}

// GetView возвращает представление с колонками и правами.
func (s *ViewService) GetView(ctx context.Context, id string) (ViewDetail, error) {
	v, err := s.repo.GetView(ctx, id)
	if err != nil {
		return ViewDetail{}, err
	}
	cols, err := s.repo.GetColumns(ctx, id)
	if err != nil {
		return ViewDetail{}, err
	}
	roles, err := s.repo.GetPermissions(ctx, id)
	if err != nil {
		return ViewDetail{}, err
	}
	return ViewDetail{View: v, Columns: cols, RoleCodes: roles}, nil
}

// ListViews возвращает список представлений.
func (s *ViewService) ListViews(ctx context.Context) ([]domain.DataView, error) {
	return s.repo.ListViews(ctx)
}

// UpdateViewInput — обновление мета-данных и параметров.
type UpdateViewInput struct {
	Name              string
	Slug              string
	Description       string
	PageSizeDefault   int
	PageSizeMin       int
	PageSizeMax       int
	DefaultSortColumn string
	DefaultSortDir    string
	ExportRowLimit    int
	RowScopeMode      string
}

// UpdateView обновляет мета и параметры; опубликованное представление переводится в draft (REP-FR-043).
func (s *ViewService) UpdateView(ctx context.Context, id string, in UpdateViewInput) (domain.DataView, error) {
	v, err := s.repo.GetView(ctx, id)
	if err != nil {
		return domain.DataView{}, err
	}
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	if strings.TrimSpace(in.Name) == "" || !domain.ValidSlug(in.Slug) || !domain.ValidRowScope(in.RowScopeMode) {
		return domain.DataView{}, domain.ErrViewValidation
	}
	if !validPagination(in.PageSizeMin, in.PageSizeDefault, in.PageSizeMax) {
		return domain.DataView{}, domain.ErrViewValidation
	}
	if in.ExportRowLimit <= 0 || in.ExportRowLimit > domain.MaxExportRows {
		return domain.DataView{}, domain.ErrViewValidation
	}
	if in.DefaultSortDir != "" && in.DefaultSortDir != domain.SortAsc && in.DefaultSortDir != domain.SortDesc {
		return domain.DataView{}, domain.ErrViewValidation
	}

	taken, err := s.repo.SlugExists(ctx, in.Slug, id)
	if err != nil {
		return domain.DataView{}, err
	}
	if taken {
		return domain.DataView{}, domain.ErrSlugTaken
	}

	v.Name = in.Name
	v.Slug = in.Slug
	v.Description = in.Description
	v.PageSizeDefault = in.PageSizeDefault
	v.PageSizeMin = in.PageSizeMin
	v.PageSizeMax = in.PageSizeMax
	v.DefaultSortColumn = in.DefaultSortColumn
	v.DefaultSortDir = in.DefaultSortDir
	v.ExportRowLimit = in.ExportRowLimit
	v.RowScopeMode = in.RowScopeMode
	v.Status = draftAfterEdit(v.Status)
	return s.repo.UpdateViewMeta(ctx, v)
}

// ColumnConfigInput — настройка одной колонки (сопоставляется по SourceName).
type ColumnConfigInput struct {
	SourceName  string
	Label       string
	DisplayType string
	Position    int
	Visible     bool
	Searchable  bool
	Filterable  bool
	Sortable    bool
	Exportable  bool
	Format      string
	MaskRule    string
	Width       int
	NullLabel   string
}

// UpdateColumns применяет конфигурацию колонок (по SourceName); source_name/source_type неизменны.
func (s *ViewService) UpdateColumns(ctx context.Context, id string, cfgs []ColumnConfigInput) error {
	v, err := s.repo.GetView(ctx, id)
	if err != nil {
		return err
	}
	existing, err := s.repo.GetColumns(ctx, id)
	if err != nil {
		return err
	}
	bySource := make(map[string]domain.ViewColumn, len(existing))
	for _, c := range existing {
		bySource[c.SourceName] = c
	}

	updated := make([]domain.ViewColumn, 0, len(cfgs))
	for _, cfg := range cfgs {
		col, ok := bySource[cfg.SourceName]
		if !ok {
			continue // неизвестная колонка игнорируется (source_name вне таблицы)
		}
		col.Label = cfg.Label
		if col.Label == "" {
			col.Label = col.SourceName
		}
		col.DisplayType = cfg.DisplayType
		if col.DisplayType == "" {
			col.DisplayType = domain.DisplayTypeFor(col.SourceType)
		}
		col.Position = cfg.Position
		col.Visible = cfg.Visible
		col.Searchable = cfg.Searchable
		col.Filterable = cfg.Filterable
		col.Sortable = cfg.Sortable
		col.Exportable = cfg.Exportable
		if cfg.Format != "" {
			col.Format = cfg.Format
		}
		if cfg.MaskRule != "" {
			col.MaskRule = cfg.MaskRule
		}
		col.Width = cfg.Width
		col.NullLabel = cfg.NullLabel
		updated = append(updated, col)
	}
	if err := s.repo.SaveColumns(ctx, updated); err != nil {
		return err
	}
	return s.demoteIfPublished(ctx, v)
}

// SetPermissions задаёт роли представления; опубликованное переводится в draft.
func (s *ViewService) SetPermissions(ctx context.Context, id string, roleCodes []string) error {
	v, err := s.repo.GetView(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.SetPermissions(ctx, id, dedupe(roleCodes)); err != nil {
		return err
	}
	return s.demoteIfPublished(ctx, v)
}

// RefreshColumns пересинхронизирует колонки из интроспекции, сохраняя настройки совпавших.
func (s *ViewService) RefreshColumns(ctx context.Context, id string) error {
	v, err := s.repo.GetView(ctx, id)
	if err != nil {
		return err
	}
	existing, err := s.repo.GetColumns(ctx, id)
	if err != nil {
		return err
	}
	prev := make(map[string]domain.ViewColumn, len(existing))
	for _, c := range existing {
		prev[c.SourceName] = c
	}

	cols, err := s.sources.Columns(ctx, v.DataSourceID, v.DatabaseName, v.TableName)
	if err != nil {
		return err
	}
	if len(cols) == 0 {
		return domain.ErrTableNotFound
	}
	merged := make([]domain.ViewColumn, len(cols))
	for i, c := range cols {
		if old, ok := prev[c.Name]; ok {
			old.SourceType = c.Type // тип мог измениться
			old.Position = int(c.Position)
			merged[i] = old
		} else {
			merged[i] = domain.ViewColumn{
				SourceName:  c.Name,
				SourceType:  c.Type,
				Label:       c.Name,
				DisplayType: domain.DisplayTypeFor(c.Type),
				Position:    int(c.Position),
				MaskRule:    domain.MaskNone,
				Format:      "{}",
			}
		}
	}
	if err := s.repo.ReplaceColumns(ctx, id, merged); err != nil {
		return err
	}
	return s.demoteIfPublished(ctx, v)
}

// Publish проверяет готовность (REP-FR-041) и фиксирует snapshot (REP-FR-042).
func (s *ViewService) Publish(ctx context.Context, id string) (domain.DataView, error) {
	v, err := s.repo.GetView(ctx, id)
	if err != nil {
		return domain.DataView{}, err
	}

	// источник должен быть активен
	src, err := s.sources.GetSource(ctx, v.DataSourceID)
	if err != nil {
		return domain.DataView{}, err
	}
	if src.Status != domain.SourceStatusActive {
		return domain.DataView{}, domain.ErrPublishValidation
	}
	// валидный уникальный slug
	if !domain.ValidSlug(v.Slug) {
		return domain.DataView{}, domain.ErrPublishValidation
	}
	if taken, err := s.repo.SlugExists(ctx, v.Slug, id); err != nil {
		return domain.DataView{}, err
	} else if taken {
		return domain.DataView{}, domain.ErrSlugTaken
	}
	// хотя бы одна видимая колонка
	cols, err := s.repo.GetColumns(ctx, id)
	if err != nil {
		return domain.DataView{}, err
	}
	visible := visibleColumns(cols)
	if len(visible) == 0 {
		return domain.DataView{}, domain.ErrPublishValidation
	}
	// хотя бы одна роль
	roles, err := s.repo.GetPermissions(ctx, id)
	if err != nil {
		return domain.DataView{}, err
	}
	if len(roles) == 0 {
		return domain.DataView{}, domain.ErrPublishValidation
	}

	snap := buildSnapshot(v, visible, roles)
	snap.SchemaHash = schemaHash(snap.Columns)
	payload, err := json.Marshal(snap)
	if err != nil {
		return domain.DataView{}, err
	}
	if err := s.repo.Publish(ctx, id, string(payload), snap.SchemaHash, time.Now().UTC()); err != nil {
		return domain.DataView{}, err
	}
	return s.repo.GetView(ctx, id)
}

// Disable отключает представление без удаления (REP-FR-044).
func (s *ViewService) Disable(ctx context.Context, id string) error {
	if _, err := s.repo.GetView(ctx, id); err != nil {
		return err
	}
	return s.repo.SetStatus(ctx, id, domain.ViewStatusDisabled)
}

// DeleteView удаляет представление.
func (s *ViewService) DeleteView(ctx context.Context, id string) error {
	return s.repo.DeleteView(ctx, id)
}

// Snapshot возвращает опубликованный snapshot (для отладки/следующего слайса).
func (s *ViewService) Snapshot(ctx context.Context, id string) (*domain.PublishedSnapshot, error) {
	raw, err := s.repo.GetSnapshot(ctx, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var snap domain.PublishedSnapshot
	if err := json.Unmarshal([]byte(raw), &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

// --- helpers ---

func (s *ViewService) demoteIfPublished(ctx context.Context, v domain.DataView) error {
	if v.Status == domain.ViewStatusPublished {
		return s.repo.SetStatus(ctx, v.ID, domain.ViewStatusDraft)
	}
	return nil
}

func draftAfterEdit(status string) string {
	if status == domain.ViewStatusPublished {
		return domain.ViewStatusDraft
	}
	return status
}

func validPagination(min, def, max int) bool {
	return min >= domain.MinPageSize && max <= domain.MaxPageSize && min <= def && def <= max
}

func visibleColumns(cols []domain.ViewColumn) []domain.ViewColumn {
	out := make([]domain.ViewColumn, 0, len(cols))
	for _, c := range cols {
		if c.Visible {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Position < out[j].Position })
	return out
}

func buildSnapshot(v domain.DataView, visible []domain.ViewColumn, roles []string) domain.PublishedSnapshot {
	cols := make([]domain.SnapshotColumn, len(visible))
	for i, c := range visible {
		cols[i] = domain.SnapshotColumn{
			SourceName:  c.SourceName,
			SourceType:  c.SourceType,
			Label:       c.Label,
			DisplayType: c.DisplayType,
			Position:    c.Position,
			Searchable:  c.Searchable,
			Filterable:  c.Filterable,
			Sortable:    c.Sortable,
			Exportable:  c.Exportable,
			Format:      c.Format,
			MaskRule:    c.MaskRule,
			NullLabel:   c.NullLabel,
		}
	}
	return domain.PublishedSnapshot{
		DatabaseName:      v.DatabaseName,
		TableName:         v.TableName,
		PageSizeDefault:   v.PageSizeDefault,
		PageSizeMin:       v.PageSizeMin,
		PageSizeMax:       v.PageSizeMax,
		DefaultSortColumn: v.DefaultSortColumn,
		DefaultSortDir:    v.DefaultSortDir,
		ExportRowLimit:    v.ExportRowLimit,
		RowScopeMode:      v.RowScopeMode,
		RoleCodes:         roles,
		Columns:           cols,
	}
}

func schemaHash(cols []domain.SnapshotColumn) string {
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = c.SourceName + ":" + c.SourceType
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func dedupe(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
