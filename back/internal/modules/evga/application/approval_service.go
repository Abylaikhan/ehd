package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"ehd-api/internal/modules/auth/contract"
	"ehd-api/internal/modules/auth/eds"
	"ehd-api/internal/modules/evga/domain"
	"ehd-api/internal/modules/evga/repository"
)

// signChallengeTTL — срок жизни challenge на подписание (спека 015 FR-1).
const signChallengeTTL = 5 * time.Minute

// signChallenges — одноразовые challenge подписания, привязанные к пользователю и уведомлению.
type signChallenges struct {
	mu   sync.Mutex
	vals map[string]signChallenge
}

type signChallenge struct {
	userID   string
	noticeID int64
	expires  time.Time
}

func newSignChallenges() *signChallenges {
	return &signChallenges{vals: map[string]signChallenge{}}
}

func (c *signChallenges) issue(userID string, noticeID int64, now time.Time) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	ch := hex.EncodeToString(buf)
	c.mu.Lock()
	defer c.mu.Unlock()
	// попутно чистим протухшие
	for k, v := range c.vals {
		if now.After(v.expires) {
			delete(c.vals, k)
		}
	}
	c.vals[ch] = signChallenge{userID: userID, noticeID: noticeID, expires: now.Add(signChallengeTTL)}
	return ch, nil
}

// consume — одноразовое использование: challenge должен принадлежать этому пользователю и уведомлению.
func (c *signChallenges) consume(ch, userID string, noticeID int64, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.vals[ch]
	if !ok {
		return false
	}
	delete(c.vals, ch)
	return v.userID == userID && v.noticeID == noticeID && now.Before(v.expires)
}

// ApprovalRepoPort — порт obm_evga-части согласования.
type ApprovalRepoPort interface {
	Meta(ctx context.Context, noticeID int64, deptID *int64) (repository.NoticeMeta, error)
	IssueNumber(ctx context.Context, meta repository.NoticeMeta, now time.Time) (string, error)
	SetReqStatus(ctx context.Context, reqID int64, statID int64) error
	SetSigner(ctx context.Context, noticeID int64, signerObmID int64, at time.Time) error
	DeptUsers(ctx context.Context, deptID int64, q string) ([]domain.ObmUser, error)
	UserNames(ctx context.Context, ids []int64) (map[int64]string, error)
	DeptOfUsers(ctx context.Context, ids []int64, deptID int64) (bool, error)
}

// RouteRepoPort — порт маршрута (БД ЕХД).
type RouteRepoPort interface {
	Steps(ctx context.Context, noticeID int64) ([]domain.RouteStep, error)
	Template(ctx context.Context, noticeID int64) ([]domain.RouteStep, error)
	SaveTemplate(ctx context.Context, noticeID int64, in domain.RouteInput, names map[int64]string) error
	StartRound(ctx context.Context, noticeID int64, template []domain.RouteStep) (int, error)
	OpenStep(ctx context.Context, noticeID int64) (*domain.RouteStep, error)
	CloseStep(ctx context.Context, stepID int64, result, comment string, closedBy int64, openNext bool) (*domain.RouteStep, error)
	CancelOpenSteps(ctx context.Context, noticeID int64) error
	// SetStepSignature — CMS подписи согласующего на шаге (спека 015 FR-3).
	SetStepSignature(ctx context.Context, stepID int64, cms string, at time.Time) error
}

// ApprovalService — маршрут и согласование уведомлений (спека 010).
type ApprovalService struct {
	obm          ApprovalRepoPort
	route        RouteRepoPort
	users        UserResolver
	scopes       ScopeProvider
	writeEnabled bool
	log          *zap.Logger
	now          func() time.Time
	// подписание согласующим (спека 015): verifier из auth/eds, рубильник EVGA_APPROVE_EDS
	eds         eds.Verifier
	edsRequired bool
	challenges  *signChallenges
}

func NewApprovalService(obm ApprovalRepoPort, route RouteRepoPort, users UserResolver, scopes ScopeProvider, writeEnabled bool, log *zap.Logger) *ApprovalService {
	return &ApprovalService{
		obm: obm, route: route, users: users, scopes: scopes,
		writeEnabled: writeEnabled, log: log, now: time.Now,
		challenges: newSignChallenges(),
	}
}

// EnableApproveEDS — включить обязательную ЭЦП согласующего (спека 015 FR-5).
func (s *ApprovalService) EnableApproveEDS(v eds.Verifier) {
	s.eds = v
	s.edsRequired = v != nil
}

// ApproveEDSRequired — требуется ли подпись при согласовании (для фронта).
func (s *ApprovalService) ApproveEDSRequired() bool { return s.edsRequired }

// editableStates — статусы заявки, в которых маршрут редактируем и возможна отправка (FR-2/4, §11.4).
func editableState(statID *int64) bool {
	return statID == nil || *statID == domain.ReqStatDraft || *statID == domain.ReqStatRework
}

// actorMeta — доступ + метаданные уведомления + obm-id действующего пользователя.
func (s *ApprovalService) actorMeta(ctx context.Context, id contract.Identity, noticeID int64, needWrite bool) (repository.NoticeMeta, *int64, error) {
	if needWrite && !s.writeEnabled {
		return repository.NoticeMeta{}, nil, ErrReadOnlyMode
	}
	scope, err := s.scopes.Scope(ctx, id)
	if err != nil {
		return repository.NoticeMeta{}, nil, err
	}
	if scope.Unmapped {
		return repository.NoticeMeta{}, nil, domain.ErrNotFound
	}
	meta, err := s.obm.Meta(ctx, noticeID, deptFilter(scope))
	if err != nil {
		return repository.NoticeMeta{}, nil, err
	}
	var actor *int64
	if iin, verified, err := s.scopes.UserIINOf(ctx, id); err == nil && verified && iin != "" {
		if uid, err := s.users.UserIDByIIN(ctx, iin); err == nil {
			actor = uid
		}
	}
	return meta, actor, nil
}

// RouteView — маршрут: шаблон + ход исполнения (FR-1).
type RouteView struct {
	Editable bool
	Template []domain.RouteStep
	History  []domain.RouteStep
}

// Route — просмотр маршрута; доступен всем ролям модуля в пределах видимости.
func (s *ApprovalService) Route(ctx context.Context, id contract.Identity, noticeID int64) (RouteView, error) {
	meta, _, err := s.actorMeta(ctx, id, noticeID, false)
	if err != nil {
		return RouteView{}, err
	}
	tpl, err := s.route.Template(ctx, noticeID)
	if err != nil {
		return RouteView{}, err
	}
	// предзаполнение: этап «Создать исходящее» = инициатор (EVGA-FR-054, AT-09а)
	if len(tpl) == 0 && meta.CreatedBy != nil {
		names, _ := s.obm.UserNames(ctx, []int64{*meta.CreatedBy})
		tpl = []domain.RouteStep{{
			NoticeID: noticeID, Round: 0, StepNN: 1, Kind: domain.RouteKindOutgoing,
			AssigneeObmID: *meta.CreatedBy, AssigneeName: names[*meta.CreatedBy],
			Status: domain.RouteStepDraft,
		}}
	}
	all, err := s.route.Steps(ctx, noticeID)
	if err != nil {
		return RouteView{}, err
	}
	history := make([]domain.RouteStep, 0, len(all))
	for _, st := range all {
		if st.Round > 0 {
			history = append(history, st)
		}
	}
	return RouteView{Editable: editableState(meta.StatID), Template: tpl, History: history}, nil
}

// SaveRoute — редактирование шаблона маршрута (FR-2/3).
func (s *ApprovalService) SaveRoute(ctx context.Context, id contract.Identity, noticeID int64, in domain.RouteInput) error {
	if !id.IsAdmin && !hasRole(id, domain.RoleAuditor) {
		return ErrCuratorReadOnly
	}
	meta, _, err := s.actorMeta(ctx, id, noticeID, true)
	if err != nil {
		return err
	}
	if !editableState(meta.StatID) {
		return domain.ErrNoticeNotEditable
	}
	if err := in.Validate(); err != nil {
		return err
	}
	// участники — только сотрудники департамента уведомления (FR-3)
	if meta.DepartmentID != nil {
		ids := append(append([]int64{}, in.ApproverIDs...), in.OutgoingUserID)
		ok, err := s.obm.DeptOfUsers(ctx, ids, *meta.DepartmentID)
		if err != nil {
			return err
		}
		if !ok {
			return domain.ErrRouteIncomplete
		}
	}
	names, err := s.obm.UserNames(ctx, append(append([]int64{}, in.ApproverIDs...), in.OutgoingUserID))
	if err != nil {
		return err
	}
	return s.route.SaveTemplate(ctx, noticeID, in, names)
}

// Submit — отправка на согласование (FR-4): номер + статус 2 + первый шаг.
func (s *ApprovalService) Submit(ctx context.Context, id contract.Identity, noticeID int64) (string, error) {
	if !id.IsAdmin && !hasRole(id, domain.RoleAuditor) {
		return "", ErrCuratorReadOnly
	}
	meta, _, err := s.actorMeta(ctx, id, noticeID, true)
	if err != nil {
		return "", err
	}
	if !editableState(meta.StatID) || meta.ReqID == nil {
		return "", domain.ErrRouteStateInvalid
	}
	tpl, err := s.route.Template(ctx, noticeID)
	if err != nil {
		return "", err
	}
	in := templateToInput(tpl)
	if err := in.Validate(); err != nil {
		return "", err
	}

	num, err := s.obm.IssueNumber(ctx, meta, s.now())
	if err != nil {
		return "", err
	}
	if _, err := s.route.StartRound(ctx, noticeID, tpl); err != nil {
		return "", err
	}
	if err := s.obm.SetReqStatus(ctx, *meta.ReqID, domain.ReqStatApproving); err != nil {
		return "", err
	}
	s.log.Info("evga: notice submitted",
		zap.Int64("notice_id", noticeID), zap.String("docnum", num), zap.String("user_id", id.UserID))
	return num, nil
}

func templateToInput(tpl []domain.RouteStep) domain.RouteInput {
	in := domain.RouteInput{}
	for _, st := range tpl {
		switch st.Kind {
		case domain.RouteKindApprove:
			in.ApproverIDs = append(in.ApproverIDs, st.AssigneeObmID)
		case domain.RouteKindOutgoing:
			in.OutgoingUserID = st.AssigneeObmID
		}
	}
	return in
}

// Approve — согласование=подписание текущим согласующим (FR-5, В1).
// ApproveChallenge — одноразовый challenge на подписание шага (спека 015 FR-1).
// Выдаётся только исполнителю открытого шага согласования.
func (s *ApprovalService) ApproveChallenge(ctx context.Context, id contract.Identity, noticeID int64) (string, error) {
	meta, actor, err := s.actorMeta(ctx, id, noticeID, true)
	if err != nil {
		return "", err
	}
	if meta.StatID == nil || *meta.StatID != domain.ReqStatApproving {
		return "", domain.ErrRouteStateInvalid
	}
	step, err := s.route.OpenStep(ctx, noticeID)
	if err != nil {
		return "", err
	}
	if step == nil || step.Kind != domain.RouteKindApprove {
		return "", domain.ErrRouteStateInvalid
	}
	if actor == nil || *actor != step.AssigneeObmID {
		return "", domain.ErrNotAssignee
	}
	return s.challenges.issue(id.UserID, noticeID, s.now())
}

// verifySignature — проверка ЭЦП согласующего (спека 015 FR-2): challenge одноразов,
// CMS проходит криптопроверку, ИИН сертификата совпадает с ИИН пользователя.
func (s *ApprovalService) verifySignature(ctx context.Context, id contract.Identity, noticeID int64, challenge, signedData string) error {
	if strings.TrimSpace(challenge) == "" || strings.TrimSpace(signedData) == "" {
		return domain.ErrSignatureRequired
	}
	if !s.challenges.consume(challenge, id.UserID, noticeID, s.now()) {
		return domain.ErrChallengeInvalid
	}
	sd, err := s.eds.Verify(challenge, signedData)
	if err != nil || !sd.Valid {
		return domain.ErrSignatureInvalid
	}
	iin, verified, err := s.scopes.UserIINOf(ctx, id)
	if err != nil || !verified || iin == "" {
		return domain.ErrSignerMismatch
	}
	if sd.IIN != iin {
		// подписывать можно только своим сертификатом (ИИН не логируем — ПДн)
		return domain.ErrSignerMismatch
	}
	return nil
}

func (s *ApprovalService) Approve(ctx context.Context, id contract.Identity, noticeID int64, comment, challenge, signedData string) error {
	meta, actor, err := s.actorMeta(ctx, id, noticeID, true)
	if err != nil {
		return err
	}
	if meta.StatID == nil || *meta.StatID != domain.ReqStatApproving || meta.ReqID == nil {
		return domain.ErrRouteStateInvalid
	}
	step, err := s.route.OpenStep(ctx, noticeID)
	if err != nil {
		return err
	}
	if step == nil || step.Kind != domain.RouteKindApprove {
		return domain.ErrRouteStateInvalid
	}
	if actor == nil || *actor != step.AssigneeObmID {
		return domain.ErrNotAssignee
	}
	// ЭЦП согласующего (спека 015 FR-2/FR-5): при включённом рубильнике обязательна
	if s.edsRequired {
		if err := s.verifySignature(ctx, id, noticeID, challenge, signedData); err != nil {
			return err
		}
		if err := s.route.SetStepSignature(ctx, step.ID, signedData, s.now()); err != nil {
			return err
		}
	}

	next, err := s.route.CloseStep(ctx, step.ID, domain.RouteResultApproved, comment, *actor, true)
	if err != nil {
		return err
	}
	if next == nil || next.Kind == domain.RouteKindOutgoing {
		// последний согласующий = подписант (В1): статус «На подписании», фиксация подписанта
		if err := s.obm.SetSigner(ctx, noticeID, *actor, s.now()); err != nil {
			return err
		}
		if err := s.obm.SetReqStatus(ctx, *meta.ReqID, domain.ReqStatSigning); err != nil {
			return err
		}
	}
	s.log.Info("evga: notice approved",
		zap.Int64("notice_id", noticeID), zap.Int64("step", step.ID), zap.String("user_id", id.UserID))
	return nil
}

// Reject — возврат на доработку с обязательным комментарием (FR-6).
func (s *ApprovalService) Reject(ctx context.Context, id contract.Identity, noticeID int64, comment string) error {
	if strings.TrimSpace(comment) == "" {
		return domain.ErrCommentRequired
	}
	meta, actor, err := s.actorMeta(ctx, id, noticeID, true)
	if err != nil {
		return err
	}
	if meta.StatID == nil || *meta.StatID != domain.ReqStatApproving || meta.ReqID == nil {
		return domain.ErrRouteStateInvalid
	}
	step, err := s.route.OpenStep(ctx, noticeID)
	if err != nil {
		return err
	}
	if step == nil || step.Kind != domain.RouteKindApprove {
		return domain.ErrRouteStateInvalid
	}
	if actor == nil || *actor != step.AssigneeObmID {
		return domain.ErrNotAssignee
	}
	if _, err := s.route.CloseStep(ctx, step.ID, domain.RouteResultRejected, strings.TrimSpace(comment), *actor, false); err != nil {
		return err
	}
	if err := s.route.CancelOpenSteps(ctx, noticeID); err != nil {
		return err
	}
	if err := s.obm.SetReqStatus(ctx, *meta.ReqID, domain.ReqStatRework); err != nil {
		return err
	}
	s.log.Info("evga: notice rejected to rework",
		zap.Int64("notice_id", noticeID), zap.String("user_id", id.UserID))
	return nil
}

// DeptUsersSearch — сотрудники департамента уведомления для маршрута (FR-3).
func (s *ApprovalService) DeptUsersSearch(ctx context.Context, id contract.Identity, noticeID int64, q string) ([]domain.ObmUser, error) {
	meta, _, err := s.actorMeta(ctx, id, noticeID, false)
	if err != nil {
		return nil, err
	}
	if meta.DepartmentID == nil {
		return []domain.ObmUser{}, nil
	}
	return s.obm.DeptUsers(ctx, *meta.DepartmentID, strings.TrimSpace(q))
}
