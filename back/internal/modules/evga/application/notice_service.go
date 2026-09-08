package application

import (
	"context"
	"strings"

	"go.uber.org/zap"

	"ehd-api/internal/modules/auth/contract"
	"ehd-api/internal/modules/evga/domain"
)

// NoticeRepoPort — порт репозитория уведомлений (спека 009).
type NoticeRepoPort interface {
	Preview(ctx context.Context, ids []int64, deptID *int64) ([]domain.NoticeGroup, []domain.RejectedRecord, error)
	SearchCli(ctx context.Context, q string) ([]domain.CliOrg, error)
	CreateNotices(ctx context.Context, groups []domain.CreateGroupInput, deptID *int64, ownerDeptID int64, createdBy *int64, noticeTxt string) ([]domain.CreatedNotice, []domain.RejectedRecord, error)
	ListNotices(ctx context.Context, deptID, filterDept, statusID *int64, docnum string, page domain.Page) ([]domain.NoticeListItem, int64, error)
	GetNotice(ctx context.Context, id int64, deptID *int64) (domain.NoticeCard, error)
	DeleteNotice(ctx context.Context, id int64, deptID *int64) error
}

// UserResolver — id пользователя obm_evga по ИИН (реализует StatusRepo).
type UserResolver interface {
	UserIDByIIN(ctx context.Context, iin string) (*int64, error)
}

// NoticeService — формирование и просмотр уведомлений (спека 009 FR-10).
type NoticeService struct {
	repo         NoticeRepoPort
	users        UserResolver
	scopes       ScopeProvider
	writeEnabled bool
	log          *zap.Logger
}

func NewNoticeService(repo NoticeRepoPort, users UserResolver, scopes ScopeProvider, writeEnabled bool, log *zap.Logger) *NoticeService {
	return &NoticeService{repo: repo, users: users, scopes: scopes, writeEnabled: writeEnabled, log: log}
}

// auditorScope — область аудитора/админа для формирования (куратор — отказ).
// Возвращает deptID-фильтр (nil у админа), департамент-владелец создаваемых уведомлений
// и created_by (пользователь obm_evga по ИИН, может быть nil).
func (s *NoticeService) auditorScope(ctx context.Context, id contract.Identity, needWrite bool) (deptFilterID *int64, ownerDept int64, createdBy *int64, err error) {
	if needWrite && !s.writeEnabled {
		return nil, 0, nil, ErrReadOnlyMode
	}
	isAuditor := hasRole(id, domain.RoleAuditor)
	if !id.IsAdmin && !isAuditor {
		return nil, 0, nil, ErrCuratorReadOnly
	}
	scope, err := s.scopes.Scope(ctx, id)
	if err != nil {
		return nil, 0, nil, err
	}
	if !scope.AllDepartments {
		if scope.Unmapped || scope.DepartmentID == nil {
			return nil, 0, nil, domain.ErrNotFound
		}
		deptFilterID = scope.DepartmentID
		ownerDept = *scope.DepartmentID
	}
	if iin, verified, err := s.scopes.UserIINOf(ctx, id); err == nil && verified && iin != "" {
		if uid, err := s.users.UserIDByIIN(ctx, iin); err == nil {
			createdBy = uid
		}
	}
	return deptFilterID, ownerDept, createdBy, nil
}

// Preview — группировка отмеченных записей (spec FR-1/2). Чтение — write-mode не требуется.
func (s *NoticeService) Preview(ctx context.Context, id contract.Identity, ids []int64) ([]domain.NoticeGroup, []domain.RejectedRecord, error) {
	if len(ids) == 0 || len(ids) > MaxBulkIDs {
		return nil, nil, domain.ErrBulkLimit
	}
	deptID, _, _, err := s.auditorScope(ctx, id, false)
	if err != nil {
		return nil, nil, err
	}
	groups, rejected, err := s.repo.Preview(ctx, ids, deptID)
	if err != nil {
		return nil, nil, err
	}
	return groups, rejected, nil
}

// SearchCli — поиск адресата (spec FR-3); доступен всем ролям модуля.
func (s *NoticeService) SearchCli(ctx context.Context, q string) ([]domain.CliOrg, error) {
	q = strings.TrimSpace(q)
	if len([]rune(q)) < 2 {
		return []domain.CliOrg{}, nil
	}
	return s.repo.SearchCli(ctx, q)
}

// CreateResult — итог создания (spec FR-6).
type CreateResult struct {
	Created  []domain.CreatedNotice
	Rejected []domain.RejectedRecord
}

// Create — создание проектов уведомлений (spec FR-4..6).
func (s *NoticeService) Create(ctx context.Context, id contract.Identity, groups []domain.CreateGroupInput) (CreateResult, error) {
	if len(groups) == 0 {
		return CreateResult{}, domain.ErrBulkLimit
	}
	total := 0
	for _, g := range groups {
		if g.CliID <= 0 {
			// кнопка «Создать» неактивна без адресата (EVGA-FR-034) — сервер дублирует проверку
			return CreateResult{}, domain.ErrAddresseeRequired
		}
		total += len(g.RecordIDs)
	}
	if total == 0 || total > MaxBulkIDs {
		return CreateResult{}, domain.ErrBulkLimit
	}

	deptID, ownerDept, createdBy, err := s.auditorScope(ctx, id, true)
	if err != nil {
		return CreateResult{}, err
	}
	if deptID == nil {
		// уведомление обязано принадлежать департаменту (EVGA-FR-037); у админа без
		// привязки по ИИН департамента-владельца нет — создание только аудитором
		return CreateResult{}, domain.ErrNoOwnerDepartment
	}

	created, rejected, err := s.repo.CreateNotices(ctx, groups, deptID, ownerDept, createdBy, domain.DefaultNoticeText)
	if err != nil {
		return CreateResult{}, err
	}
	s.log.Info("evga: notices created",
		zap.Int("created", len(created)), zap.Int("rejected", len(rejected)),
		zap.String("user_id", id.UserID))
	return CreateResult{Created: created, Rejected: rejected}, nil
}

// List — список уведомлений (spec FR-7): аудитор — свои; куратор/админ — все + фильтр.
func (s *NoticeService) List(ctx context.Context, id contract.Identity, filterDept, statusID *int64, docnum string, page domain.Page) ([]domain.NoticeListItem, int64, domain.DepartmentScope, error) {
	scope, err := s.scopes.Scope(ctx, id)
	if err != nil {
		return nil, 0, scope, err
	}
	if scope.Unmapped {
		return []domain.NoticeListItem{}, 0, scope, nil
	}
	deptID := deptFilter(scope)
	if deptID != nil {
		filterDept = nil
	}
	items, total, err := s.repo.ListNotices(ctx, deptID, filterDept, statusID, docnum, page)
	if err != nil {
		return nil, 0, scope, err
	}
	return items, total, scope, nil
}

// Get — карточка уведомления (spec FR-8).
func (s *NoticeService) Get(ctx context.Context, id contract.Identity, noticeID int64) (domain.NoticeCard, error) {
	scope, err := s.scopes.Scope(ctx, id)
	if err != nil {
		return domain.NoticeCard{}, err
	}
	if scope.Unmapped {
		return domain.NoticeCard{}, domain.ErrNotFound
	}
	return s.repo.GetNotice(ctx, noticeID, deptFilter(scope))
}

// Delete — удаление проекта (spec FR-9).
func (s *NoticeService) Delete(ctx context.Context, id contract.Identity, noticeID int64) error {
	deptID, _, _, err := s.auditorScope(ctx, id, true)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteNotice(ctx, noticeID, deptID); err != nil {
		return err
	}
	s.log.Info("evga: notice draft deleted", zap.Int64("notice_id", noticeID), zap.String("user_id", id.UserID))
	return nil
}
