package application

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"ehd-api/internal/modules/auth/contract"
	"ehd-api/internal/modules/evga/domain"
	"ehd-api/internal/modules/evga/repository"
)

// ErrReadOnlyMode — запись выключена конфигом (спека 008 FR-1, 403 EVGA_READ_ONLY).
var ErrReadOnlyMode = errors.New("модуль ОБМ ЕВГА работает в режиме чтения")

// ErrCuratorReadOnly — куратор не меняет статусы (403 ACCESS_DENIED).
var ErrCuratorReadOnly = errors.New("роль куратора не допускает изменения записей")

// MaxBulkIDs — предел записей за один bulk-вызов (спека 008 FR-8).
const MaxBulkIDs = 1000

// StatusRepoPort — порт репозитория статусов.
type StatusRepoPort interface {
	ApplyStatusChange(ctx context.Context, ids []int64, targetStatus int64, attrs domain.TransitionAttrs, deptID *int64, changedBy *int64, source string) ([]repository.ChangeOutcome, int, error)
	History(ctx context.Context, recordID int64) ([]repository.HistoryEntry, error)
	UserIDByIIN(ctx context.Context, iin string) (*int64, error)
}

// ScopeProvider — доступ к scope/ИИН из основного сервиса модуля.
type ScopeProvider interface {
	Scope(ctx context.Context, id contract.Identity) (domain.DepartmentScope, error)
	UserIINOf(ctx context.Context, id contract.Identity) (iin string, verified bool, err error)
}

// StatusService — простановка статусов (спека 008 FR-7/8) и история (FR-12).
type StatusService struct {
	repo         StatusRepoPort
	scopes       ScopeProvider
	writeEnabled bool
	log          *zap.Logger
}

func NewStatusService(repo StatusRepoPort, scopes ScopeProvider, writeEnabled bool, log *zap.Logger) *StatusService {
	return &StatusService{repo: repo, scopes: scopes, writeEnabled: writeEnabled, log: log}
}

// BulkReport — отчёт массовой простановки (EVGA-FR-023).
type BulkReport struct {
	Processed  int
	Rejected   int
	Cascaded   int // каскадно обновлённые записи того же платежа (FR-14)
	Rejections []Rejection
}

type Rejection struct {
	ID     int64
	Reason string
}

// writeScope — общая проверка права записи: режим, роль, департамент.
// Возвращает deptID-ограничение (nil для админа) и changed_by для журнала.
func (s *StatusService) writeScope(ctx context.Context, id contract.Identity) (*int64, *int64, error) {
	if !s.writeEnabled {
		return nil, nil, ErrReadOnlyMode
	}
	isAuditor := hasRole(id, domain.RoleAuditor)
	if !id.IsAdmin && !isAuditor {
		// куратор (и любой без роли аудитора) статусы не меняет (спека 008 FR-7)
		return nil, nil, ErrCuratorReadOnly
	}

	var deptID *int64
	if !id.IsAdmin {
		scope, err := s.scopes.Scope(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		if scope.Unmapped || scope.DepartmentID == nil {
			return nil, nil, domain.ErrNotFound // без департамента записи недоступны
		}
		deptID = scope.DepartmentID
	}

	// changed_by журнала — пользователь obm_evga по ИИН; NULL, если не найден (FR-11)
	var changedBy *int64
	if iin, verified, err := s.scopes.UserIINOf(ctx, id); err == nil && verified && iin != "" {
		if uid, err := s.repo.UserIDByIIN(ctx, iin); err == nil {
			changedBy = uid
		}
	}
	return deptID, changedBy, nil
}

// ChangeStatus — одиночный переход (FR-7). Ошибка перехода возвращается как есть.
func (s *StatusService) ChangeStatus(ctx context.Context, id contract.Identity, recordID int64, target int64, attrs domain.TransitionAttrs) error {
	deptID, changedBy, err := s.writeScope(ctx, id)
	if err != nil {
		return err
	}
	outcomes, cascaded, err := s.repo.ApplyStatusChange(ctx, []int64{recordID}, target, attrs, deptID, changedBy, domain.ChangeSourceManual)
	if err != nil {
		return err
	}
	if len(outcomes) == 1 && outcomes[0].Err != nil {
		return outcomes[0].Err
	}
	s.log.Info("evga: status changed",
		zap.Int64("record_id", recordID), zap.Int64("to", target),
		zap.Int("cascaded", cascaded), zap.String("user_id", id.UserID))
	return nil
}

// BulkChangeStatus — массовый переход одной транзакцией с отчётом (FR-8).
func (s *StatusService) BulkChangeStatus(ctx context.Context, id contract.Identity, ids []int64, target int64, attrs domain.TransitionAttrs) (BulkReport, error) {
	if len(ids) == 0 || len(ids) > MaxBulkIDs {
		return BulkReport{}, domain.ErrBulkLimit
	}
	deptID, changedBy, err := s.writeScope(ctx, id)
	if err != nil {
		return BulkReport{}, err
	}
	outcomes, cascaded, err := s.repo.ApplyStatusChange(ctx, ids, target, attrs, deptID, changedBy, domain.ChangeSourceBulk)
	if err != nil {
		return BulkReport{}, err
	}
	rep := BulkReport{Cascaded: cascaded}
	for _, o := range outcomes {
		if o.Err == nil {
			rep.Processed++
		} else {
			rep.Rejected++
			rep.Rejections = append(rep.Rejections, Rejection{ID: o.ID, Reason: o.Reason})
		}
	}
	s.log.Info("evga: bulk status change",
		zap.Int("processed", rep.Processed), zap.Int("rejected", rep.Rejected),
		zap.Int64("to", target), zap.String("user_id", id.UserID))
	return rep, nil
}

// History — журнал записи (FR-12). Видимость записи проверяет хендлер (Card тем же scope).
func (s *StatusService) History(ctx context.Context, recordID int64) ([]repository.HistoryEntry, error) {
	return s.repo.History(ctx, recordID)
}
