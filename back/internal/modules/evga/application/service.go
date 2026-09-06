// Package application — use cases модуля ОБМ ЕВГА (спека 007): доверенная видимость
// по департаменту (резолв ИИН → obm_evga.users), реестр, карточка, справочники, экспорт.
package application

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"ehd-api/internal/modules/auth/contract"
	"ehd-api/internal/modules/evga/domain"
)

// Repo — порт репозитория реестра (реализуется repository.RegistryRepo).
type Repo interface {
	List(ctx context.Context, f domain.Filter, deptID *int64, page domain.Page) (domain.RecordPage, error)
	ListAll(ctx context.Context, f domain.Filter, deptID *int64, sort, order string, limit int) ([]domain.RiskRecord, error)
	Get(ctx context.Context, id int64, deptID *int64) (domain.RiskRecord, error)
	Statuses(ctx context.Context) ([]domain.Reference, error)
	Profiles(ctx context.Context) ([]domain.Reference, error)
	Departments(ctx context.Context) ([]domain.Reference, error)
	DepartmentByUserIIN(ctx context.Context, iin string) (*int64, string, error)
	Ping(ctx context.Context) error
}

// IINProvider — доступ к ИИН пользователя ЕХД (auth/contract, без сети).
type IINProvider interface {
	UserIIN(ctx context.Context, userID string) (iin string, verified bool, err error)
}

const scopeCacheTTL = 15 * time.Minute // spec FR-3

type scopeEntry struct {
	scope domain.DepartmentScope
	exp   time.Time
}

// Service — приложение модуля ЕВГА.
type Service struct {
	repo Repo
	iin  IINProvider
	log  *zap.Logger
	now  func() time.Time

	mu    sync.RWMutex
	cache map[string]scopeEntry // userID → scope
}

func NewService(repo Repo, iin IINProvider, log *zap.Logger) *Service {
	return &Service{repo: repo, iin: iin, log: log, now: time.Now, cache: map[string]scopeEntry{}}
}

// hasRole — проверка кода роли в доверенной личности.
func hasRole(id contract.Identity, code string) bool {
	for _, c := range id.RoleCodes {
		if c == code {
			return true
		}
	}
	return false
}

// HasModuleAccess — доступ к модулю: роли evga_* либо администратор (spec, контракты API).
func HasModuleAccess(id contract.Identity) bool {
	return id.IsAdmin || hasRole(id, domain.RoleAuditor) || hasRole(id, domain.RoleCurator)
}

// Scope — доверенная видимость (spec FR-3/4): куратор/админ — все департаменты;
// аудитор — департамент по ИИН из obm_evga.users; иначе Unmapped.
func (s *Service) Scope(ctx context.Context, id contract.Identity) (domain.DepartmentScope, error) {
	if id.IsAdmin || hasRole(id, domain.RoleCurator) {
		return domain.DepartmentScope{AllDepartments: true}, nil
	}

	s.mu.RLock()
	if e, ok := s.cache[id.UserID]; ok && s.now().Before(e.exp) {
		s.mu.RUnlock()
		return e.scope, nil
	}
	s.mu.RUnlock()

	scope := domain.DepartmentScope{Unmapped: true}
	iin, verified, err := s.iin.UserIIN(ctx, id.UserID)
	if err != nil {
		return domain.DepartmentScope{}, err
	}
	if verified && iin != "" {
		deptID, title, err := s.repo.DepartmentByUserIIN(ctx, iin)
		if err != nil {
			return domain.DepartmentScope{}, s.wrapSourceErr(ctx, err)
		}
		if deptID != nil {
			scope = domain.DepartmentScope{DepartmentID: deptID, DepartmentTitle: title}
		}
	}
	if scope.Unmapped {
		// ИИН не подтверждён/не найден/без департамента: пустой реестр, не ошибка (FR-3).
		s.log.Info("evga: user not mapped to department", zap.String("user_id", id.UserID))
	}

	s.mu.Lock()
	s.cache[id.UserID] = scopeEntry{scope: scope, exp: s.now().Add(scopeCacheTTL)}
	s.mu.Unlock()
	return scope, nil
}

// wrapSourceErr — ошибки связности внешней БД транслируются в ErrSourceUnavailable (503).
// Быстрый ping отличает недоступность источника от прочих ошибок запроса.
func (s *Service) wrapSourceErr(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	pingCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	if pingErr := s.repo.Ping(pingCtx); pingErr != nil {
		s.log.Warn("evga: source unavailable", zap.Error(pingErr))
		return domain.ErrSourceUnavailable
	}
	return err
}

// deptFilter — параметр видимости для репозитория.
func deptFilter(scope domain.DepartmentScope) *int64 {
	if scope.AllDepartments {
		return nil
	}
	return scope.DepartmentID
}

// List — реестр рисков (spec FR-4..8).
func (s *Service) List(ctx context.Context, id contract.Identity, f domain.Filter, page domain.Page) (domain.RecordPage, domain.DepartmentScope, error) {
	scope, err := s.Scope(ctx, id)
	if err != nil {
		return domain.RecordPage{}, scope, err
	}
	if scope.Unmapped {
		return domain.RecordPage{Items: []domain.RiskRecord{}}, scope, nil
	}
	if !scope.AllDepartments {
		f.DepartmentID = nil // фильтр по департаменту доступен только куратору/админу (FR-6)
	}
	res, err := s.repo.List(ctx, f, deptFilter(scope), page)
	if err != nil {
		return domain.RecordPage{}, scope, s.wrapSourceErr(ctx, err)
	}
	return res, scope, nil
}

// Card — карточка записи в пределах видимости (spec FR-9).
func (s *Service) Card(ctx context.Context, id contract.Identity, recordID int64) (domain.RiskRecord, error) {
	scope, err := s.Scope(ctx, id)
	if err != nil {
		return domain.RiskRecord{}, err
	}
	if scope.Unmapped {
		return domain.RiskRecord{}, domain.ErrNotFound
	}
	rec, err := s.repo.Get(ctx, recordID, deptFilter(scope))
	if err != nil && err != domain.ErrNotFound {
		return domain.RiskRecord{}, s.wrapSourceErr(ctx, err)
	}
	return rec, err
}

// References — справочники модуля (spec FR-10); департаменты — только при полной видимости.
type References struct {
	Statuses    []domain.Reference
	Profiles    []domain.Reference
	Departments []domain.Reference
}

func (s *Service) References(ctx context.Context, id contract.Identity) (References, error) {
	scope, err := s.Scope(ctx, id)
	if err != nil {
		return References{}, err
	}
	statuses, err := s.repo.Statuses(ctx)
	if err != nil {
		return References{}, s.wrapSourceErr(ctx, err)
	}
	profiles, err := s.repo.Profiles(ctx)
	if err != nil {
		return References{}, s.wrapSourceErr(ctx, err)
	}
	out := References{Statuses: statuses, Profiles: profiles}
	if scope.AllDepartments {
		deps, err := s.repo.Departments(ctx)
		if err != nil {
			return References{}, s.wrapSourceErr(ctx, err)
		}
		out.Departments = deps
	}
	return out, nil
}

// Ping — readyz внешней БД (spec FR-12).
func (s *Service) Ping(ctx context.Context) error { return s.repo.Ping(ctx) }

// UserIINOf — ИИН пользователя для внутренних нужд модуля (журнал статусов, спека 008 FR-11).
func (s *Service) UserIINOf(ctx context.Context, id contract.Identity) (string, bool, error) {
	return s.iin.UserIIN(ctx, id.UserID)
}
