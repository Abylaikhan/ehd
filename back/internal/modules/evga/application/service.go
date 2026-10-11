// Package application — use cases модуля ОБМ ЭВГА (спека 007): доверенная видимость
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
	Activities(ctx context.Context) ([]domain.Reference, error)
	DepartmentByUserIIN(ctx context.Context, iin string) (*int64, string, error)
	Ping(ctx context.Context) error
}

// IINProvider — доступ к ИИН пользователя ЕХД (auth/contract, без сети).
type IINProvider interface {
	UserIIN(ctx context.Context, userID string) (iin string, verified bool, err error)
}

// UserDeptPort — привязка «пользователь ЕХД → департамент ДВГА» в БД ЕХД (спека 016 FR-1).
type UserDeptPort interface {
	Get(ctx context.Context, userID string) (*int64, error)
	Set(ctx context.Context, userID string, departmentID int64) error
	Clear(ctx context.Context, userID string) error
}

const scopeCacheTTL = 15 * time.Minute // spec FR-3

type scopeEntry struct {
	scope domain.DepartmentScope
	exp   time.Time
}

// Service — приложение модуля ЭВГА.
type Service struct {
	repo     Repo
	iin      IINProvider
	log      *zap.Logger
	now      func() time.Time
	warnDays int // окно «истекающий» для подсветки сроков (EVGA-FR-080)
	// переопределение департамента из админки (спека 016); nil — функция не подключена
	userDept UserDeptPort

	mu    sync.RWMutex
	cache map[string]scopeEntry // userID → scope
}

// SetUserDeptRepo подключает хранилище привязок «пользователь → департамент» (спека 016).
func (s *Service) SetUserDeptRepo(r UserDeptPort) { s.userDept = r }

func NewService(repo Repo, iin IINProvider, log *zap.Logger) *Service {
	return &Service{
		repo: repo, iin: iin, log: log, now: time.Now,
		warnDays: domain.DefaultDeadlineWarnDays,
		cache:    map[string]scopeEntry{},
	}
}

// SetDeadlineWarnDays переопределяет окно предупреждения о сроке (EVGA_DEADLINE_WARN_DAYS).
// Некорректное значение (<=0) игнорируется — остаётся дефолт.
func (s *Service) SetDeadlineWarnDays(days int) {
	if days > 0 {
		s.warnDays = days
	}
}

// enrichDeadline проставляет вычисленное состояние контрольного срока (EVGA-FR-080).
func (s *Service) enrichDeadline(items []domain.RiskRecord) {
	now := s.now()
	for i := range items {
		items[i].DeadlineState = domain.DeadlineState(items[i].ExecDue, items[i].StatusID, s.warnDays, now)
	}
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

	// 1) переопределение из админки имеет приоритет над резолвом по ИИН (спека 016 FR-2)
	if s.userDept != nil {
		if deptID, err := s.userDept.Get(ctx, id.UserID); err == nil && deptID != nil {
			title, err := s.departmentTitle(ctx, *deptID)
			if err != nil {
				return domain.DepartmentScope{}, s.wrapSourceErr(ctx, err)
			}
			scope = domain.DepartmentScope{DepartmentID: deptID, DepartmentTitle: title}
		}
	}

	// 2) иначе — прежний резолв по ИИН из obm_evga.users (spec 007 FR-3)
	if scope.Unmapped {
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

// departmentTitle — название департамента ДВГА по id (из справочника obm_evga).
func (s *Service) departmentTitle(ctx context.Context, deptID int64) (string, error) {
	deps, err := s.repo.Departments(ctx)
	if err != nil {
		return "", err
	}
	for _, d := range deps {
		if d.ID == deptID {
			return d.Title, nil
		}
	}
	return "", nil
}

// invalidateScope — сбросить кэш области пользователя (спека 016 FR-4).
func (s *Service) invalidateScope(userID string) {
	s.mu.Lock()
	delete(s.cache, userID)
	s.mu.Unlock()
}

// UserDepartment — текущая привязка пользователя (спека 016 FR-3, только админ).
func (s *Service) UserDepartment(ctx context.Context, actor contract.Identity, userID string) (*int64, error) {
	if !actor.IsAdmin {
		return nil, ErrCuratorReadOnly
	}
	if s.userDept == nil {
		return nil, nil
	}
	return s.userDept.Get(ctx, userID)
}

// UserDeptInfo — эффективный департамент пользователя для списка админки (спека 020).
// Source: "override" — задан администратором, "iin" — выведен по ИИН из obm_evga.
type UserDeptInfo struct {
	DepartmentID    int64
	DepartmentTitle string
	Source          string
}

// UsersDepartments — эффективные департаменты пачкой, для колонки в списке пользователей
// (спека 020 FR-2). Только админ. Пользователи без привязки в результат не попадают.
func (s *Service) UsersDepartments(ctx context.Context, actor contract.Identity, userIDs []string) (map[string]UserDeptInfo, error) {
	if !actor.IsAdmin {
		return nil, ErrCuratorReadOnly
	}
	out := make(map[string]UserDeptInfo, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	// справочник департаментов тянем один раз на всю пачку
	deps, err := s.repo.Departments(ctx)
	if err != nil {
		return nil, s.wrapSourceErr(ctx, err)
	}
	titles := make(map[int64]string, len(deps))
	for _, d := range deps {
		titles[d.ID] = d.Title
	}

	for _, uid := range userIDs {
		// 1) переопределение из админки (спека 016)
		if s.userDept != nil {
			if id, err := s.userDept.Get(ctx, uid); err == nil && id != nil {
				out[uid] = UserDeptInfo{DepartmentID: *id, DepartmentTitle: titles[*id], Source: "override"}
				continue
			}
		}
		// 2) иначе — резолв по ИИН из obm_evga.users (ИИН не логируем)
		iin, verified, err := s.iin.UserIIN(ctx, uid)
		if err != nil || !verified || iin == "" {
			continue
		}
		id, title, err := s.repo.DepartmentByUserIIN(ctx, iin)
		if err != nil || id == nil {
			continue
		}
		out[uid] = UserDeptInfo{DepartmentID: *id, DepartmentTitle: title, Source: "iin"}
	}
	return out, nil
}

// SetUserDepartment — назначить (deptID != nil) или снять (nil) привязку (спека 016 FR-3/FR-4).
func (s *Service) SetUserDepartment(ctx context.Context, actor contract.Identity, userID string, deptID *int64) error {
	if !actor.IsAdmin {
		return ErrCuratorReadOnly
	}
	if s.userDept == nil {
		return domain.ErrNotFound
	}
	if deptID == nil {
		if err := s.userDept.Clear(ctx, userID); err != nil {
			return err
		}
		s.invalidateScope(userID)
		s.log.Info("evga: привязка к департаменту снята", zap.String("user_id", userID))
		return nil
	}
	// департамент должен существовать в справочнике obm_evga
	title, err := s.departmentTitle(ctx, *deptID)
	if err != nil {
		return s.wrapSourceErr(ctx, err)
	}
	if title == "" {
		return domain.ErrInvalidSort // INVALID_FILTER: неизвестный департамент
	}
	if err := s.userDept.Set(ctx, userID, *deptID); err != nil {
		return err
	}
	s.invalidateScope(userID)
	s.log.Info("evga: пользователь привязан к департаменту",
		zap.String("user_id", userID), zap.Int64("department_id", *deptID))
	return nil
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
	s.enrichDeadline(res.Items)
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
	if err == nil {
		rec.DeadlineState = domain.DeadlineState(rec.ExecDue, rec.StatusID, s.warnDays, s.now())
	}
	return rec, err
}

// References — справочники модуля (spec FR-10); департаменты — только при полной видимости.
type References struct {
	Statuses    []domain.Reference
	Profiles    []domain.Reference
	Departments []domain.Reference
	Activities  []domain.Reference
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
	activities, err := s.repo.Activities(ctx)
	if err != nil {
		return References{}, s.wrapSourceErr(ctx, err)
	}
	out := References{Statuses: statuses, Profiles: profiles, Activities: activities}
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
