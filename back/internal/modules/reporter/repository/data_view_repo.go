package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ehd-api/internal/modules/reporter/domain"
)

// DataViewRepo — доступ к представлениям, их колонкам и правам в PostgreSQL.
type DataViewRepo struct{ db *gorm.DB }

func NewDataViewRepo(db *gorm.DB) *DataViewRepo { return &DataViewRepo{db: db} }

// --- конвертеры ---

func toDomainView(m DataViewModel) domain.DataView {
	return domain.DataView{
		ID:                       m.ID.String(),
		Name:                     m.Name,
		Slug:                     m.Slug,
		Description:              m.Description,
		DataSourceID:             m.DataSourceID.String(),
		DatabaseName:             m.DatabaseName,
		TableName:                m.SourceTable,
		SourceMode:               m.SourceMode,
		Status:                   m.Status,
		Revision:                 m.Revision,
		SchemaHash:               m.SchemaHash,
		PageSizeDefault:          m.PageSizeDefault,
		PageSizeMin:              m.PageSizeMin,
		PageSizeMax:              m.PageSizeMax,
		DefaultSortColumn:        m.DefaultSortColumn,
		DefaultSortDir:           m.DefaultSortDir,
		ExportRowLimit:           m.ExportRowLimit,
		RowScopeMode:             m.RowScopeMode,
		KeysetColumn:             m.KeysetColumn,
		KeysetDir:                m.KeysetDir,
		RowScopeRegionColumn:     m.RowScopeRegionColumn,
		RowScopeDepartmentColumn: m.RowScopeDepartmentColumn,
		PublishedAt:              m.PublishedAt,
		CreatedAt:                m.CreatedAt,
		UpdatedAt:                m.UpdatedAt,
	}
}

func toDomainColumn(m ViewColumnModel) domain.ViewColumn {
	return domain.ViewColumn{
		ID:          m.ID.String(),
		ViewID:      m.ViewID.String(),
		SourceName:  m.SourceName,
		SourceType:  m.SourceType,
		Label:       m.Label,
		DisplayType: m.DisplayType,
		Position:    m.Position,
		Visible:     m.Visible,
		Searchable:  m.Searchable,
		Filterable:  m.Filterable,
		Sortable:    m.Sortable,
		Exportable:  m.Exportable,
		Format:      m.Format,
		MaskRule:    m.MaskRule,
		Width:       m.Width,
		NullLabel:   m.NullLabel,
	}
}

func columnModel(viewID uuid.UUID, c domain.ViewColumn) ViewColumnModel {
	format := c.Format
	if format == "" {
		format = "{}"
	}
	return ViewColumnModel{
		ID:          uuid.New(),
		ViewID:      viewID,
		SourceName:  c.SourceName,
		SourceType:  c.SourceType,
		Label:       c.Label,
		DisplayType: c.DisplayType,
		Position:    c.Position,
		Visible:     c.Visible,
		Searchable:  c.Searchable,
		Filterable:  c.Filterable,
		Sortable:    c.Sortable,
		Exportable:  c.Exportable,
		Format:      format,
		MaskRule:    c.MaskRule,
		Width:       c.Width,
		NullLabel:   c.NullLabel,
	}
}

// --- представление ---

// CreateView создаёт представление вместе с колонками в одной транзакции.
func (r *DataViewRepo) CreateView(ctx context.Context, v domain.DataView, cols []domain.ViewColumn) (domain.DataView, error) {
	srcID, err := uuid.Parse(v.DataSourceID)
	if err != nil {
		return domain.DataView{}, domain.ErrSourceNotFound
	}
	m := DataViewModel{
		ID:                       uuid.New(),
		Name:                     v.Name,
		Slug:                     v.Slug,
		Description:              v.Description,
		DataSourceID:             srcID,
		DatabaseName:             v.DatabaseName,
		SourceTable:              v.TableName,
		SourceMode:               domain.SourceModePhysicalObject,
		Status:                   domain.ViewStatusDraft,
		PageSizeDefault:          v.PageSizeDefault,
		PageSizeMin:              v.PageSizeMin,
		PageSizeMax:              v.PageSizeMax,
		ExportRowLimit:           v.ExportRowLimit,
		RowScopeMode:             v.RowScopeMode,
		KeysetColumn:             v.KeysetColumn,
		KeysetDir:                v.KeysetDir,
		RowScopeRegionColumn:     v.RowScopeRegionColumn,
		RowScopeDepartmentColumn: v.RowScopeDepartmentColumn,
	}
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
		for _, c := range cols {
			cm := columnModel(m.ID, c)
			if err := tx.Create(&cm).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return domain.DataView{}, err
	}
	return toDomainView(m), nil
}

func (r *DataViewRepo) GetView(ctx context.Context, id string) (domain.DataView, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return domain.DataView{}, domain.ErrViewNotFound
	}
	var m DataViewModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.DataView{}, domain.ErrViewNotFound
		}
		return domain.DataView{}, err
	}
	return toDomainView(m), nil
}

// GetViewBySlug возвращает представление по slug (для пользовательского запроса данных).
func (r *DataViewRepo) GetViewBySlug(ctx context.Context, slug string) (domain.DataView, error) {
	var m DataViewModel
	if err := r.db.WithContext(ctx).First(&m, "slug = ?", slug).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.DataView{}, domain.ErrViewNotFound
		}
		return domain.DataView{}, err
	}
	return toDomainView(m), nil
}

func (r *DataViewRepo) ListViews(ctx context.Context) ([]domain.DataView, error) {
	var ms []DataViewModel
	if err := r.db.WithContext(ctx).Order("created_at").Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]domain.DataView, len(ms))
	for i, m := range ms {
		out[i] = toDomainView(m)
	}
	return out, nil
}

// UpdateViewMeta обновляет мета-данные, параметры и статус представления.
func (r *DataViewRepo) UpdateViewMeta(ctx context.Context, v domain.DataView) (domain.DataView, error) {
	uid, err := uuid.Parse(v.ID)
	if err != nil {
		return domain.DataView{}, domain.ErrViewNotFound
	}
	updates := map[string]any{
		"name":                        v.Name,
		"slug":                        v.Slug,
		"description":                 v.Description,
		"page_size_default":           v.PageSizeDefault,
		"page_size_min":               v.PageSizeMin,
		"page_size_max":               v.PageSizeMax,
		"default_sort_column":         v.DefaultSortColumn,
		"default_sort_dir":            v.DefaultSortDir,
		"export_row_limit":            v.ExportRowLimit,
		"row_scope_mode":              v.RowScopeMode,
		"keyset_column":               v.KeysetColumn,
		"keyset_dir":                  v.KeysetDir,
		"row_scope_region_column":     v.RowScopeRegionColumn,
		"row_scope_department_column": v.RowScopeDepartmentColumn,
		"status":                      v.Status,
	}
	res := r.db.WithContext(ctx).Model(&DataViewModel{}).Where("id = ?", uid).Updates(updates)
	if res.Error != nil {
		return domain.DataView{}, res.Error
	}
	if res.RowsAffected == 0 {
		return domain.DataView{}, domain.ErrViewNotFound
	}
	return r.GetView(ctx, v.ID)
}

// SetStatus переводит представление в указанный статус.
func (r *DataViewRepo) SetStatus(ctx context.Context, id, status string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return domain.ErrViewNotFound
	}
	res := r.db.WithContext(ctx).Model(&DataViewModel{}).Where("id = ?", uid).Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrViewNotFound
	}
	return nil
}

// DeleteView удаляет представление вместе с колонками и правами.
func (r *DataViewRepo) DeleteView(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return domain.ErrViewNotFound
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("view_id = ?", uid).Delete(&ViewColumnModel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("view_id = ?", uid).Delete(&ViewPermissionModel{}).Error; err != nil {
			return err
		}
		res := tx.Where("id = ?", uid).Delete(&DataViewModel{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return domain.ErrViewNotFound
		}
		return nil
	})
}

// SlugExists сообщает, занят ли slug другим представлением (excludeID — исключаемый id).
func (r *DataViewRepo) SlugExists(ctx context.Context, slug, excludeID string) (bool, error) {
	q := r.db.WithContext(ctx).Model(&DataViewModel{}).Where("slug = ?", slug)
	if excludeID != "" {
		if uid, err := uuid.Parse(excludeID); err == nil {
			q = q.Where("id <> ?", uid)
		}
	}
	var n int64
	if err := q.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// --- колонки ---

func (r *DataViewRepo) GetColumns(ctx context.Context, viewID string) ([]domain.ViewColumn, error) {
	uid, err := uuid.Parse(viewID)
	if err != nil {
		return nil, domain.ErrViewNotFound
	}
	var ms []ViewColumnModel
	if err := r.db.WithContext(ctx).Where("view_id = ?", uid).Order("position").Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ViewColumn, len(ms))
	for i, m := range ms {
		out[i] = toDomainColumn(m)
	}
	return out, nil
}

// SaveColumns обновляет конфигурацию существующих колонок по их id.
func (r *DataViewRepo) SaveColumns(ctx context.Context, cols []domain.ViewColumn) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, c := range cols {
			cid, err := uuid.Parse(c.ID)
			if err != nil {
				return domain.ErrViewValidation
			}
			format := c.Format
			if format == "" {
				format = "{}"
			}
			updates := map[string]any{
				"label":        c.Label,
				"display_type": c.DisplayType,
				"position":     c.Position,
				"visible":      c.Visible,
				"searchable":   c.Searchable,
				"filterable":   c.Filterable,
				"sortable":     c.Sortable,
				"exportable":   c.Exportable,
				"format":       format,
				"mask_rule":    c.MaskRule,
				"width":        c.Width,
				"null_label":   c.NullLabel,
			}
			if err := tx.Model(&ViewColumnModel{}).Where("id = ?", cid).Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ReplaceColumns заменяет весь набор колонок представления (для refresh из интроспекции).
func (r *DataViewRepo) ReplaceColumns(ctx context.Context, viewID string, cols []domain.ViewColumn) error {
	uid, err := uuid.Parse(viewID)
	if err != nil {
		return domain.ErrViewNotFound
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("view_id = ?", uid).Delete(&ViewColumnModel{}).Error; err != nil {
			return err
		}
		for _, c := range cols {
			cm := columnModel(uid, c)
			if err := tx.Create(&cm).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// --- права ---

func (r *DataViewRepo) GetPermissions(ctx context.Context, viewID string) ([]string, error) {
	uid, err := uuid.Parse(viewID)
	if err != nil {
		return nil, domain.ErrViewNotFound
	}
	var ms []ViewPermissionModel
	if err := r.db.WithContext(ctx).Where("view_id = ?", uid).Order("role_code").Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.RoleCode
	}
	return out, nil
}

// SetPermissions заменяет набор ролей представления.
func (r *DataViewRepo) SetPermissions(ctx context.Context, viewID string, roleCodes []string) error {
	uid, err := uuid.Parse(viewID)
	if err != nil {
		return domain.ErrViewNotFound
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("view_id = ?", uid).Delete(&ViewPermissionModel{}).Error; err != nil {
			return err
		}
		for _, code := range roleCodes {
			pm := ViewPermissionModel{ID: uuid.New(), ViewID: uid, RoleCode: code}
			if err := tx.Create(&pm).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// --- публикация ---

// Publish атомарно фиксирует snapshot, schema_hash, published_at, статус и revision+1.
func (r *DataViewRepo) Publish(ctx context.Context, viewID, snapshotJSON, schemaHash string, at time.Time) error {
	uid, err := uuid.Parse(viewID)
	if err != nil {
		return domain.ErrViewNotFound
	}
	res := r.db.WithContext(ctx).Model(&DataViewModel{}).Where("id = ?", uid).Updates(map[string]any{
		"status":             domain.ViewStatusPublished,
		"published_snapshot": snapshotJSON,
		"schema_hash":        schemaHash,
		"published_at":       at,
		"revision":           gorm.Expr("revision + 1"),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrViewNotFound
	}
	return nil
}

// GetSnapshot возвращает опубликованный snapshot (JSON) представления.
func (r *DataViewRepo) GetSnapshot(ctx context.Context, viewID string) (string, error) {
	uid, err := uuid.Parse(viewID)
	if err != nil {
		return "", domain.ErrViewNotFound
	}
	var m DataViewModel
	if err := r.db.WithContext(ctx).Select("published_snapshot").First(&m, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", domain.ErrViewNotFound
		}
		return "", err
	}
	if m.PublishedSnapshot == nil {
		return "", nil
	}
	return *m.PublishedSnapshot, nil
}
