package application

import (
	"context"
	"strings"

	"ehd-api/internal/modules/reporter/domain"
)

// MenuRepo — хранилище пунктов навигации.
type MenuRepo interface {
	List(ctx context.Context) ([]domain.MenuItem, error)
	Get(ctx context.Context, id string) (domain.MenuItem, error)
	Create(ctx context.Context, it domain.MenuItem) (domain.MenuItem, error)
	Update(ctx context.Context, it domain.MenuItem) (domain.MenuItem, error)
	Delete(ctx context.Context, id string) error
	ViewSlugByID(ctx context.Context) (map[string]string, error)
}

// MenuService — управление меню и построение разрешённого дерева навигации.
type MenuService struct {
	repo MenuRepo
}

func NewMenuService(repo MenuRepo) *MenuService { return &MenuService{repo: repo} }

// MenuInput — данные создания/обновления пункта меню.
type MenuInput struct {
	ParentID     string
	DataViewID   string
	NameRu       string
	NameKk       string
	IconKey      string
	Position     int
	IsDisabled   bool
	PublicAccess bool
	RoleCodes    []string
}

func menuItemFrom(id string, in MenuInput) domain.MenuItem {
	return domain.MenuItem{
		ID: id, ParentID: in.ParentID, DataViewID: in.DataViewID,
		NameRu: in.NameRu, NameKk: in.NameKk, IconKey: in.IconKey,
		Position: in.Position, IsDisabled: in.IsDisabled, PublicAccess: in.PublicAccess,
		RoleCodes: in.RoleCodes,
	}
}

// AdminList возвращает все пункты меню (реестр).
func (s *MenuService) AdminList(ctx context.Context) ([]domain.MenuItem, error) {
	return s.repo.List(ctx)
}

// Create создаёт пункт меню (с проверкой вложенности/глубины).
func (s *MenuService) Create(ctx context.Context, in MenuInput) (domain.MenuItem, error) {
	if strings.TrimSpace(in.NameRu) == "" {
		return domain.MenuItem{}, domain.ErrMenuValidation
	}
	items, err := s.repo.List(ctx)
	if err != nil {
		return domain.MenuItem{}, err
	}
	if err := validateParent(items, "", in.ParentID); err != nil {
		return domain.MenuItem{}, err
	}
	return s.repo.Create(ctx, menuItemFrom("", in))
}

// Update обновляет пункт меню.
func (s *MenuService) Update(ctx context.Context, id string, in MenuInput) (domain.MenuItem, error) {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return domain.MenuItem{}, err
	}
	if strings.TrimSpace(in.NameRu) == "" {
		return domain.MenuItem{}, domain.ErrMenuValidation
	}
	items, err := s.repo.List(ctx)
	if err != nil {
		return domain.MenuItem{}, err
	}
	if err := validateParent(items, id, in.ParentID); err != nil {
		return domain.MenuItem{}, err
	}
	return s.repo.Update(ctx, menuItemFrom(id, in))
}

// Delete удаляет пункт меню.
func (s *MenuService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// Navigation строит разрешённое дерево навигации пользователя (роли/public/админ).
func (s *MenuService) Navigation(ctx context.Context, req Requester) ([]domain.MenuNode, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	slugMap, err := s.repo.ViewSlugByID(ctx)
	if err != nil {
		return nil, err
	}
	byParent := make(map[string][]domain.MenuItem)
	for _, it := range items {
		if it.IsDisabled {
			continue
		}
		byParent[it.ParentID] = append(byParent[it.ParentID], it)
	}
	return buildNav(byParent, "", slugMap, req), nil
}

func buildNav(byParent map[string][]domain.MenuItem, parentID string, slugMap map[string]string, req Requester) []domain.MenuNode {
	var out []domain.MenuNode
	for _, it := range byParent[parentID] {
		if menuDenied(it, req) {
			continue
		}
		children := buildNav(byParent, it.ID, slugMap, req)
		to := ""
		if it.DataViewID != "" {
			if slug, ok := slugMap[it.DataViewID]; ok && slug != "" {
				to = "/reporter/" + slug
			}
		}
		if to == "" && len(children) == 0 {
			continue // раздел без видимых детей и без маршрута
		}
		out = append(out, domain.MenuNode{ID: it.ID, Title: it.NameRu, Icon: it.IconKey, To: to, Children: children})
	}
	return out
}

// menuDenied — пункт скрыт, только если у него явно заданы роли и они не совпадают
// с ролями пользователя (админ и public_access видят всегда; пункт без ролей —
// нейтральный раздел, показывается при наличии видимых детей/маршрута).
func menuDenied(it domain.MenuItem, req Requester) bool {
	if req.IsAdmin || it.PublicAccess || len(it.RoleCodes) == 0 {
		return false
	}
	return !intersects(it.RoleCodes, req.RoleCodes)
}

// --- валидация дерева (цикл/глубина) ---

func validateParent(items []domain.MenuItem, selfID, parentID string) error {
	byID := make(map[string]domain.MenuItem, len(items))
	byParent := make(map[string][]string)
	for _, it := range items {
		byID[it.ID] = it
		byParent[it.ParentID] = append(byParent[it.ParentID], it.ID)
	}

	subtreeH := 1
	if selfID != "" {
		subtreeH = subtreeHeight(byParent, selfID)
	}

	if parentID == "" {
		if subtreeH > domain.MaxMenuDepth {
			return domain.ErrMenuDepth
		}
		return nil
	}
	if _, ok := byID[parentID]; !ok {
		return domain.ErrMenuValidation
	}
	if selfID != "" && (parentID == selfID || isDescendant(byParent, selfID, parentID)) {
		return domain.ErrMenuCycle
	}
	if depthOf(byID, parentID)+subtreeH > domain.MaxMenuDepth {
		return domain.ErrMenuDepth
	}
	return nil
}

// depthOf — 1-базовая глубина пункта (проход вверх по цепочке родителей).
func depthOf(byID map[string]domain.MenuItem, id string) int {
	d := 1
	cur := byID[id]
	for cur.ParentID != "" && d < 32 {
		next, exists := byID[cur.ParentID]
		if !exists {
			break
		}
		d++
		cur = next
	}
	return d
}

func subtreeHeight(byParent map[string][]string, id string) int {
	kids := byParent[id]
	if len(kids) == 0 {
		return 1
	}
	maxH := 0
	for _, k := range kids {
		if h := subtreeHeight(byParent, k); h > maxH {
			maxH = h
		}
	}
	return 1 + maxH
}

func isDescendant(byParent map[string][]string, ancestorID, nodeID string) bool {
	for _, k := range byParent[ancestorID] {
		if k == nodeID || isDescendant(byParent, k, nodeID) {
			return true
		}
	}
	return false
}
