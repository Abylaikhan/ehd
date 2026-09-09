package application

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"

	"ehd-api/internal/modules/auth/contract"
	"ehd-api/internal/modules/evga/domain"
	"ehd-api/internal/modules/evga/repository"
)

// OutgoingRepoPort — порт репозитория исходящих.
type OutgoingRepoPort interface {
	CreateOutgoing(ctx context.Context, noticeID int64, docnum string, createdBy *int64) (int64, error)
	Out(ctx context.Context, noticeID int64) (*repository.OutInfo, error)
	SetRegistrationData(ctx context.Context, noticeID int64, docNum string, docDate time.Time) error
	Register(ctx context.Context, noticeID int64) (repository.RegisterResult, error)
	PendingRegistration(ctx context.Context, limit int) ([]int64, error)
}

// OutgoingService — создание исходящего и регистрация (спека 011).
type OutgoingService struct {
	repo         OutgoingRepoPort
	approvalRepo ApprovalRepoPort
	route        RouteRepoPort
	users        UserResolver
	scopes       ScopeProvider
	writeEnabled bool
	log          *zap.Logger
}

func NewOutgoingService(repo OutgoingRepoPort, approvalRepo ApprovalRepoPort, route RouteRepoPort, users UserResolver, scopes ScopeProvider, writeEnabled bool, log *zap.Logger) *OutgoingService {
	return &OutgoingService{repo: repo, approvalRepo: approvalRepo, route: route, users: users, scopes: scopes, writeEnabled: writeEnabled, log: log}
}

// CreateOutgoing — этап «Создать исходящее» (FR-1): исполнитель открытого шага outgoing
// при статусе заявки «На подписании»; создаёт its_out и закрывает шаг.
func (s *OutgoingService) CreateOutgoing(ctx context.Context, id contract.Identity, noticeID int64) (int64, error) {
	if !s.writeEnabled {
		return 0, ErrReadOnlyMode
	}
	scope, err := s.scopes.Scope(ctx, id)
	if err != nil {
		return 0, err
	}
	if scope.Unmapped && !id.IsAdmin {
		return 0, domain.ErrNotFound
	}
	meta, err := s.approvalRepo.Meta(ctx, noticeID, deptFilter(scope))
	if err != nil {
		return 0, err
	}
	if meta.StatID == nil || *meta.StatID != domain.ReqStatSigning {
		return 0, domain.ErrRouteStateInvalid
	}

	step, err := s.route.OpenStep(ctx, noticeID)
	if err != nil {
		return 0, err
	}
	if step == nil || step.Kind != domain.RouteKindOutgoing {
		return 0, domain.ErrRouteStateInvalid
	}
	var actor *int64
	if iin, verified, err := s.scopes.UserIINOf(ctx, id); err == nil && verified && iin != "" {
		if uid, err := s.users.UserIDByIIN(ctx, iin); err == nil {
			actor = uid
		}
	}
	if !id.IsAdmin && (actor == nil || *actor != step.AssigneeObmID) {
		return 0, domain.ErrNotAssignee
	}

	outID, err := s.repo.CreateOutgoing(ctx, noticeID, meta.Docnum, actor)
	if err != nil {
		return 0, err
	}
	closedBy := int64(0)
	if actor != nil {
		closedBy = *actor
	}
	if _, err := s.route.CloseStep(ctx, step.ID, domain.RouteResultApproved, "Исходящее создано", closedBy, false); err != nil {
		return 0, err
	}
	s.log.Info("evga: outgoing created",
		zap.Int64("notice_id", noticeID), zap.Int64("out_id", outID), zap.String("user_id", id.UserID))
	return outID, nil
}

// Out — исходящее уведомления для карточки (в пределах видимости).
func (s *OutgoingService) Out(ctx context.Context, id contract.Identity, noticeID int64) (*repository.OutInfo, error) {
	scope, err := s.scopes.Scope(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err := s.approvalRepo.Meta(ctx, noticeID, deptFilter(scope)); err != nil {
		return nil, err
	}
	return s.repo.Out(ctx, noticeID)
}

// SimulateRegistration — админ-симуляция канцелярии (FR-6): номер+дата → транзакция регистрации.
func (s *OutgoingService) SimulateRegistration(ctx context.Context, id contract.Identity, noticeID int64, docNum string, docDate time.Time) (repository.RegisterResult, error) {
	if !s.writeEnabled {
		return repository.RegisterResult{}, ErrReadOnlyMode
	}
	if !id.IsAdmin {
		return repository.RegisterResult{}, ErrCuratorReadOnly
	}
	if strings.TrimSpace(docNum) == "" {
		return repository.RegisterResult{}, domain.ErrCommentRequired
	}
	if err := s.repo.SetRegistrationData(ctx, noticeID, strings.TrimSpace(docNum), docDate); err != nil {
		return repository.RegisterResult{}, err
	}
	return s.register(ctx, noticeID)
}

// register — транзакция §11.3 c логом результата.
func (s *OutgoingService) register(ctx context.Context, noticeID int64) (repository.RegisterResult, error) {
	res, err := s.repo.Register(ctx, noticeID)
	if err != nil {
		return res, err
	}
	if res.Applied {
		s.log.Info("evga: outgoing registered",
			zap.Int64("notice_id", noticeID), zap.String("out_num", res.OutDocNum),
			zap.Int("records_to_status4", res.Updated), zap.Int("cascaded", res.Cascaded),
			zap.String("exec_due", res.ExecDue))
	}
	return res, nil
}

// WatchOnce — один проход вотчера (FR-5): регистрирует уведомления, которым платформа
// уже проставила номер. Возвращает число обработанных.
func (s *OutgoingService) WatchOnce(ctx context.Context) int {
	if !s.writeEnabled {
		return 0
	}
	ids, err := s.repo.PendingRegistration(ctx, 50)
	if err != nil {
		s.log.Warn("evga: outgoing watcher query failed", zap.Error(err))
		return 0
	}
	done := 0
	for _, id := range ids {
		if res, err := s.register(ctx, id); err != nil {
			s.log.Warn("evga: outgoing watcher register failed", zap.Int64("notice_id", id), zap.Error(err))
		} else if res.Applied {
			done++
		}
	}
	return done
}

// Watch — фоновый цикл вотчера до отмены контекста (FR-5).
func (s *OutgoingService) Watch(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		return
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	s.log.Info("evga: outgoing watcher started", zap.Duration("interval", interval))
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.WatchOnce(ctx)
		}
	}
}
