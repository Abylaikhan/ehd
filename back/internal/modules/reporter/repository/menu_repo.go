package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ehd-api/internal/modules/reporter/domain"
)

// MenuRepo — доступ к пунктам навигации Reporter в PostgreSQL.
type MenuRepo struct{ db *gorm.DB }

func NewMenuRepo(db *gorm.DB) *MenuRepo { return &MenuRepo{db: db} }

func uuidPtrString(p *uuid.UUID) string {
	if p == nil {
		return ""
	}
	return p.String()
}

func parseOptionalUUID(s string) (*uuid.UUID, error) {
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func toDomainMenu(m MenuItemModel, roles []string) domain.MenuItem {
	return domain.MenuItem{
		ID:           m.ID.String(),
		ParentID:     uuidPtrString(m.ParentID),
		DataViewID:   uuidPtrString(m.DataViewID),
		NameRu:       m.NameRu,
		NameKk:       m.NameKk,
		IconKey:      m.IconKey,
		Position:     m.Position,
		IsDisabled:   m.IsDisabled,
		PublicAccess: m.PublicAccess,
		RoleCodes:    roles,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// List возвращает все пункты меню (по порядку) с их ролями.
func (r *MenuRepo) List(ctx context.Context) ([]domain.MenuItem, error) {
	var ms []MenuItemModel
	if err := r.db.WithContext(ctx).Order("position").Order("created_at").Find(&ms).Error; err != nil {
		return nil, err
	}
	var rolesRows []MenuItemRoleModel
	if err := r.db.WithContext(ctx).Find(&rolesRows).Error; err != nil {
		return nil, err
	}
	rolesByItem := make(map[uuid.UUID][]string)
	for _, rr := range rolesRows {
		rolesByItem[rr.MenuItemID] = append(rolesByItem[rr.MenuItemID], rr.RoleCode)
	}
	out := make([]domain.MenuItem, len(ms))
	for i, m := range ms {
		out[i] = toDomainMenu(m, rolesByItem[m.ID])
	}
	return out, nil
}

// Get возвращает пункт меню по id.
func (r *MenuRepo) Get(ctx context.Context, id string) (domain.MenuItem, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return domain.MenuItem{}, domain.ErrMenuNotFound
	}
	var m MenuItemModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.MenuItem{}, domain.ErrMenuNotFound
		}
		return domain.MenuItem{}, err
	}
	var roles []MenuItemRoleModel
	if err := r.db.WithContext(ctx).Where("menu_item_id = ?", uid).Find(&roles).Error; err != nil {
		return domain.MenuItem{}, err
	}
	codes := make([]string, len(roles))
	for i, rr := range roles {
		codes[i] = rr.RoleCode
	}
	return toDomainMenu(m, codes), nil
}

// Create сохраняет пункт с ролями в одной транзакции.
func (r *MenuRepo) Create(ctx context.Context, it domain.MenuItem) (domain.MenuItem, error) {
	parent, err := parseOptionalUUID(it.ParentID)
	if err != nil {
		return domain.MenuItem{}, domain.ErrMenuValidation
	}
	viewID, err := parseOptionalUUID(it.DataViewID)
	if err != nil {
		return domain.MenuItem{}, domain.ErrMenuValidation
	}
	m := MenuItemModel{
		ID: uuid.New(), ParentID: parent, DataViewID: viewID,
		NameRu: it.NameRu, NameKk: it.NameKk, IconKey: it.IconKey,
		Position: it.Position, IsDisabled: it.IsDisabled, PublicAccess: it.PublicAccess,
	}
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
		return insertMenuRoles(tx, m.ID, it.RoleCodes)
	})
	if err != nil {
		return domain.MenuItem{}, err
	}
	return r.Get(ctx, m.ID.String())
}

// Update обновляет поля пункта и его роли.
func (r *MenuRepo) Update(ctx context.Context, it domain.MenuItem) (domain.MenuItem, error) {
	uid, err := uuid.Parse(it.ID)
	if err != nil {
		return domain.MenuItem{}, domain.ErrMenuNotFound
	}
	parent, err := parseOptionalUUID(it.ParentID)
	if err != nil {
		return domain.MenuItem{}, domain.ErrMenuValidation
	}
	viewID, err := parseOptionalUUID(it.DataViewID)
	if err != nil {
		return domain.MenuItem{}, domain.ErrMenuValidation
	}
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&MenuItemModel{}).Where("id = ?", uid).Updates(map[string]any{
			"parent_id":     parent,
			"data_view_id":  viewID,
			"name_ru":       it.NameRu,
			"name_kk":       it.NameKk,
			"icon_key":      it.IconKey,
			"position":      it.Position,
			"is_disabled":   it.IsDisabled,
			"public_access": it.PublicAccess,
		})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return domain.ErrMenuNotFound
		}
		if err := tx.Where("menu_item_id = ?", uid).Delete(&MenuItemRoleModel{}).Error; err != nil {
			return err
		}
		return insertMenuRoles(tx, uid, it.RoleCodes)
	})
	if err != nil {
		return domain.MenuItem{}, err
	}
	return r.Get(ctx, it.ID)
}

// Delete удаляет пункт (если у него нет вложенных) вместе с ролями.
func (r *MenuRepo) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return domain.ErrMenuNotFound
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var childCount int64
		if err := tx.Model(&MenuItemModel{}).Where("parent_id = ?", uid).Count(&childCount).Error; err != nil {
			return err
		}
		if childCount > 0 {
			return domain.ErrMenuHasChildren
		}
		if err := tx.Where("menu_item_id = ?", uid).Delete(&MenuItemRoleModel{}).Error; err != nil {
			return err
		}
		res := tx.Where("id = ?", uid).Delete(&MenuItemModel{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return domain.ErrMenuNotFound
		}
		return nil
	})
}

// ViewSlugByID возвращает карту data_view_id → slug (для маршрутов навигации).
func (r *MenuRepo) ViewSlugByID(ctx context.Context) (map[string]string, error) {
	var rows []struct {
		ID   uuid.UUID
		Slug string
	}
	if err := r.db.WithContext(ctx).Model(&DataViewModel{}).Select("id", "slug").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		out[row.ID.String()] = row.Slug
	}
	return out, nil
}

func insertMenuRoles(tx *gorm.DB, itemID uuid.UUID, codes []string) error {
	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		if err := tx.Create(&MenuItemRoleModel{ID: uuid.New(), MenuItemID: itemID, RoleCode: code}).Error; err != nil {
			return err
		}
	}
	return nil
}
