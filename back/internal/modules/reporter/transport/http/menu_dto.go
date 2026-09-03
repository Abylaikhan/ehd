package http

import (
	"ehd-api/internal/modules/reporter/application"
	"ehd-api/internal/modules/reporter/domain"
)

type menuItemReq struct {
	ParentID     string   `json:"parent_id"`
	DataViewID   string   `json:"data_view_id"`
	NameRu       string   `json:"name_ru"`
	NameKk       string   `json:"name_kk"`
	IconKey      string   `json:"icon_key"`
	Position     int      `json:"position"`
	IsDisabled   bool     `json:"is_disabled"`
	PublicAccess bool     `json:"public_access"`
	RoleCodes    []string `json:"role_codes"`
}

func (r menuItemReq) toInput() application.MenuInput {
	return application.MenuInput{
		ParentID: r.ParentID, DataViewID: r.DataViewID,
		NameRu: r.NameRu, NameKk: r.NameKk, IconKey: r.IconKey,
		Position: r.Position, IsDisabled: r.IsDisabled, PublicAccess: r.PublicAccess,
		RoleCodes: r.RoleCodes,
	}
}

type menuItemResp struct {
	ID           string   `json:"id"`
	ParentID     string   `json:"parent_id"`
	DataViewID   string   `json:"data_view_id"`
	NameRu       string   `json:"name_ru"`
	NameKk       string   `json:"name_kk"`
	IconKey      string   `json:"icon_key"`
	Position     int      `json:"position"`
	IsDisabled   bool     `json:"is_disabled"`
	PublicAccess bool     `json:"public_access"`
	RoleCodes    []string `json:"role_codes"`
}

func toMenuItemResp(m domain.MenuItem) menuItemResp {
	roles := m.RoleCodes
	if roles == nil {
		roles = []string{}
	}
	return menuItemResp{
		ID: m.ID, ParentID: m.ParentID, DataViewID: m.DataViewID,
		NameRu: m.NameRu, NameKk: m.NameKk, IconKey: m.IconKey,
		Position: m.Position, IsDisabled: m.IsDisabled, PublicAccess: m.PublicAccess,
		RoleCodes: roles,
	}
}
