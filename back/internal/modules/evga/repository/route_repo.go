package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"ehd-api/internal/modules/evga/domain"
)

// RouteStepModel — этап маршрута уведомления. Хранится в БД ЕХД (не в obm_evga):
// платформенный процесс движется BPMS-скриптами, повторять их извне нельзя (спека 010 ADR).
type RouteStepModel struct {
	ID            int64  `gorm:"primaryKey"`
	NoticeID      int64  `gorm:"index;not null"` // its_risk_notice.id (obm_evga)
	Round         int    `gorm:"not null;default:0"`
	StepNN        int    `gorm:"not null"`
	Kind          string `gorm:"size:16;not null"`
	AssigneeObmID int64  `gorm:"not null"` // obm_evga.users.id
	AssigneeName  string `gorm:"size:255"`
	Status        string `gorm:"size:16;not null;default:draft"`
	Result        string `gorm:"size:16"`
	Comment       string
	OpenedAt      *time.Time
	ClosedAt      *time.Time
	ClosedByObmID *int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (RouteStepModel) TableName() string { return "evga_notice_route_steps" }

// MigrateRoute — AutoMigrate таблицы маршрута в БД ЕХД (Принцип 5: sandbox).
func MigrateRoute(db *gorm.DB) error { return db.AutoMigrate(&RouteStepModel{}) }

// RouteRepo — маршрут согласования (БД ЕХД).
type RouteRepo struct{ db *gorm.DB }

func NewRouteRepo(db *gorm.DB) *RouteRepo { return &RouteRepo{db: db} }

func toDomainStep(m RouteStepModel) domain.RouteStep {
	return domain.RouteStep{
		ID: m.ID, NoticeID: m.NoticeID, Round: m.Round, StepNN: m.StepNN, Kind: m.Kind,
		AssigneeObmID: m.AssigneeObmID, AssigneeName: m.AssigneeName,
		Status: m.Status, Result: m.Result, Comment: m.Comment,
		OpenedAt: m.OpenedAt, ClosedAt: m.ClosedAt,
	}
}

// Steps — все шаги уведомления: шаблон (round=0) и раунды исполнения, новые раунды выше.
func (rr *RouteRepo) Steps(ctx context.Context, noticeID int64) ([]domain.RouteStep, error) {
	var rows []RouteStepModel
	err := rr.db.WithContext(ctx).
		Where("notice_id = ?", noticeID).
		Order("round desc, step_nn asc, id asc").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.RouteStep, len(rows))
	for i, m := range rows {
		out[i] = toDomainStep(m)
	}
	return out, nil
}

// SaveTemplate — перезаписывает шаблон маршрута (round=0) по RouteInput (FR-2).
func (rr *RouteRepo) SaveTemplate(ctx context.Context, noticeID int64, in domain.RouteInput, names map[int64]string) error {
	return rr.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("notice_id = ? and round = 0", noticeID).Delete(&RouteStepModel{}).Error; err != nil {
			return err
		}
		rows := make([]RouteStepModel, 0, len(in.ApproverIDs)+1)
		for i, id := range in.ApproverIDs {
			rows = append(rows, RouteStepModel{
				NoticeID: noticeID, Round: 0, StepNN: i + 1,
				Kind: domain.RouteKindApprove, AssigneeObmID: id, AssigneeName: names[id],
				Status: domain.RouteStepDraft,
			})
		}
		rows = append(rows, RouteStepModel{
			NoticeID: noticeID, Round: 0, StepNN: len(in.ApproverIDs) + 1,
			Kind: domain.RouteKindOutgoing, AssigneeObmID: in.OutgoingUserID, AssigneeName: names[in.OutgoingUserID],
			Status: domain.RouteStepDraft,
		})
		return tx.Create(&rows).Error
	})
}

// Template — шаблон маршрута (round=0) в порядке шагов.
func (rr *RouteRepo) Template(ctx context.Context, noticeID int64) ([]domain.RouteStep, error) {
	var rows []RouteStepModel
	err := rr.db.WithContext(ctx).
		Where("notice_id = ? and round = 0", noticeID).
		Order("step_nn asc").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.RouteStep, len(rows))
	for i, m := range rows {
		out[i] = toDomainStep(m)
	}
	return out, nil
}

// StartRound — создаёт новый раунд исполнения из шаблона и открывает первый шаг (FR-4).
// Возвращает номер раунда.
func (rr *RouteRepo) StartRound(ctx context.Context, noticeID int64, template []domain.RouteStep) (int, error) {
	round := 0
	err := rr.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var maxRound *int
		if err := tx.Model(&RouteStepModel{}).
			Where("notice_id = ?", noticeID).
			Select("max(round)").Scan(&maxRound).Error; err != nil {
			return err
		}
		if maxRound != nil {
			round = *maxRound + 1
		} else {
			round = 1
		}
		if round < 1 {
			round = 1
		}
		now := time.Now()
		rows := make([]RouteStepModel, 0, len(template))
		for i, s := range template {
			m := RouteStepModel{
				NoticeID: noticeID, Round: round, StepNN: s.StepNN, Kind: s.Kind,
				AssigneeObmID: s.AssigneeObmID, AssigneeName: s.AssigneeName,
				Status: domain.RouteStepDraft,
			}
			if i == 0 {
				m.Status = domain.RouteStepOpen
				m.OpenedAt = &now
			}
			rows = append(rows, m)
		}
		return tx.Create(&rows).Error
	})
	return round, err
}

// OpenStep — текущий открытый шаг уведомления (последний раунд).
func (rr *RouteRepo) OpenStep(ctx context.Context, noticeID int64) (*domain.RouteStep, error) {
	var rows []RouteStepModel
	err := rr.db.WithContext(ctx).
		Where("notice_id = ? and status = ?", noticeID, domain.RouteStepOpen).
		Order("round desc, step_nn asc").
		Limit(1).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	s := toDomainStep(rows[0])
	return &s, nil
}

// CloseStep закрывает шаг с результатом; если openNext — открывает следующий шаг раунда.
// Возвращает следующий открытый шаг (nil, если раунд исчерпан или openNext=false).
func (rr *RouteRepo) CloseStep(ctx context.Context, stepID int64, result, comment string, closedBy int64, openNext bool) (*domain.RouteStep, error) {
	var next *domain.RouteStep
	err := rr.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var cur RouteStepModel
		if err := tx.First(&cur, stepID).Error; err != nil {
			return err
		}
		now := time.Now()
		if err := tx.Model(&RouteStepModel{}).Where("id = ?", stepID).Updates(map[string]any{
			"status": domain.RouteStepDone, "result": result, "comment": comment,
			"closed_at": now, "closed_by_obm_id": closedBy,
		}).Error; err != nil {
			return err
		}
		if !openNext {
			return nil
		}
		var nxt []RouteStepModel
		if err := tx.Where("notice_id = ? and round = ? and step_nn > ? and status = ?",
			cur.NoticeID, cur.Round, cur.StepNN, domain.RouteStepDraft).
			Order("step_nn asc").Limit(1).Find(&nxt).Error; err != nil {
			return err
		}
		if len(nxt) == 0 {
			return nil
		}
		if err := tx.Model(&RouteStepModel{}).Where("id = ?", nxt[0].ID).Updates(map[string]any{
			"status": domain.RouteStepOpen, "opened_at": now,
		}).Error; err != nil {
			return err
		}
		nxt[0].Status = domain.RouteStepOpen
		nxt[0].OpenedAt = &now
		s := toDomainStep(nxt[0])
		next = &s
		return nil
	})
	return next, err
}

// CancelOpenSteps — закрывает все открытые/черновые шаги текущего раунда (возврат на доработку).
func (rr *RouteRepo) CancelOpenSteps(ctx context.Context, noticeID int64) error {
	return rr.db.WithContext(ctx).Model(&RouteStepModel{}).
		Where("notice_id = ? and round > 0 and status in ?", noticeID,
			[]string{domain.RouteStepOpen, domain.RouteStepDraft}).
		Updates(map[string]any{"status": domain.RouteStepDone, "closed_at": time.Now()}).Error
}
