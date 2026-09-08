package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ehd-api/internal/modules/evga/domain"
)

// forUpdate — блокировка выбранных строк до конца транзакции.
func forUpdate() clause.Expression { return clause.Locking{Strength: "UPDATE"} }

// StatusRepo — смена статусов записей витрины и журнал (спека 008 FR-9..12).
// Единственные допустимые операции записи: UPDATE полей отработки its_tb_5_15a
// и INSERT в its_tb_5_15a_status_log (спека 008 FR-2).
type StatusRepo struct{ db *gorm.DB }

func NewStatusRepo(db *gorm.DB) *StatusRepo { return &StatusRepo{db: db} }

// ChangeOutcome — исход по одной записи (для отчёта bulk, EVGA-FR-023).
type ChangeOutcome struct {
	ID     int64
	Err    error // nil — обработана
	Reason string
}

// lockedRow — заблокированная строка витрины с данными для валидации перехода.
type lockedRow struct {
	ID               int64
	PpoPp            string
	StatusID         *int64
	AmountPart       string
	AmountForVozvrat *string
}

// applyOne — UPDATE полей отработки + строка журнала для одной записи (FR-10/11).
// Валидация перехода выполняется вызывающей стороной.
func applyOne(tx *gorm.DB, row lockedRow, targetStatus int64, attrs domain.TransitionAttrs, changedBy *int64, source string) error {
	updates := map[string]any{
		"its_risk_status_id": targetStatus,
		"updated_at":         time.Now(),
	}
	logRow := map[string]any{
		"tb_5_15a_id":    row.ID,
		"status_from_id": row.StatusID,
		"status_to_id":   targetStatus,
		"change_source":  source,
		"changed_at":     time.Now(),
	}
	if note := strings.TrimSpace(attrs.Note); note != "" {
		updates["risk_status_note"] = note
		logRow["note"] = note
	}
	if targetStatus == domain.StatusRefundDue {
		updates["amount_for_vozvrat"] = strings.TrimSpace(attrs.AmountForVozvrat)
		logRow["amount_for_vozvrat"] = strings.TrimSpace(attrs.AmountForVozvrat)
	}
	if targetStatus == domain.StatusRefunded {
		updates["refund"] = strings.TrimSpace(attrs.Refund)
		logRow["refund"] = strings.TrimSpace(attrs.Refund)
	}
	if targetStatus == domain.StatusAudit && attrs.ActivityID != nil {
		updates["its_activity_id"] = *attrs.ActivityID
	}
	if changedBy != nil {
		updates["updated_by"] = *changedBy
		logRow["changed_by"] = *changedBy
	}
	if err := tx.Table("its_tb_5_15a").Where("id = ?", row.ID).Updates(updates).Error; err != nil {
		return err
	}
	return tx.Table("its_tb_5_15a_status_log").Create(logRow).Error
}

// ApplyStatusChange выполняет переходы записей ids → targetStatus в ОДНОЙ транзакции
// (EVGA-FR-024): SELECT … FOR UPDATE, валидация матрицы по каждой записи, UPDATE валидных,
// журнал. Отклонённые записи не обрабатываются и не мешают коммиту остальных.
// deptID — видимость аудитора (nil для админа), changedBy — id пользователя obm_evga (может быть nil).
func (sr *StatusRepo) ApplyStatusChange(
	ctx context.Context,
	ids []int64,
	targetStatus int64,
	attrs domain.TransitionAttrs,
	deptID *int64,
	changedBy *int64,
	source string,
) ([]ChangeOutcome, int, error) {
	outcomes := make([]ChangeOutcome, 0, len(ids))
	cascaded := 0

	err := sr.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// блокировка строк: одновременные аудиторы не потеряют обновления (FR-9)
		q := tx.Table("its_tb_5_15a").
			Select(`id, coalesce(ppo_pp,'') as ppo_pp, its_risk_status_id as status_id, amount_part::text as amount_part, amount_for_vozvrat::text as amount_for_vozvrat`).
			Where(`id in ? and "in$trash" is null`, ids)
		if deptID != nil {
			q = q.Where("its_departments_id = ?", *deptID)
		}
		var rows []lockedRow
		if err := q.Clauses(forUpdate()).Scan(&rows).Error; err != nil {
			return err
		}

		byID := make(map[int64]lockedRow, len(rows))
		for _, r := range rows {
			byID[r.ID] = r
		}

		changedPpo := map[string]bool{}
		for _, id := range ids {
			row, ok := byID[id]
			if !ok {
				outcomes = append(outcomes, ChangeOutcome{ID: id, Err: domain.ErrNotFound,
					Reason: "Запись не найдена или относится к другому департаменту"})
				continue
			}
			if err := validateRow(row, targetStatus, attrs); err != nil {
				outcomes = append(outcomes, ChangeOutcome{ID: id, Err: err, Reason: err.Error()})
				continue
			}
			if err := applyOne(tx, row, targetStatus, attrs, changedBy, source); err != nil {
				return err
			}
			if row.PpoPp != "" {
				changedPpo[row.PpoPp] = true
			}
			outcomes = append(outcomes, ChangeOutcome{ID: id})
		}

		// каскад по платежу (спека 008 FR-14, ответ №3-1): та же смена применяется
		// к записям того же ppo_pp других профилей, если их переход допустим;
		// департамент не фильтруется — записи одного платежа принадлежат одному ДВГА.
		if len(changedPpo) > 0 && source != domain.ChangeSourceSystem {
			ppoList := make([]string, 0, len(changedPpo))
			for p := range changedPpo {
				ppoList = append(ppoList, p)
			}
			var siblings []lockedRow
			err := tx.Table("its_tb_5_15a").
				Select(`id, coalesce(ppo_pp,'') as ppo_pp, its_risk_status_id as status_id, amount_part::text as amount_part, amount_for_vozvrat::text as amount_for_vozvrat`).
				Where(`ppo_pp in ? and id not in ? and "in$trash" is null`, ppoList, ids).
				Clauses(forUpdate()).
				Scan(&siblings).Error
			if err != nil {
				return err
			}
			for _, sib := range siblings {
				if validateRow(sib, targetStatus, attrs) != nil {
					continue // недопустимый переход соседа — не трогаем («старые записи не трогает»)
				}
				if err := applyOne(tx, sib, targetStatus, attrs, changedBy, domain.ChangeSourceSystem); err != nil {
					return err
				}
				cascaded++
			}
		}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	return outcomes, cascaded, nil
}

// validateRow — валидация перехода для строки с раскрытием NULL-полей.
func validateRow(row lockedRow, targetStatus int64, attrs domain.TransitionAttrs) error {
	current := int64(0)
	if row.StatusID != nil {
		current = *row.StatusID
	}
	curVozvrat := ""
	if row.AmountForVozvrat != nil {
		curVozvrat = *row.AmountForVozvrat
	}
	return domain.ValidateTransition(current, targetStatus, attrs, row.AmountPart, curVozvrat)
}

// HistoryEntry — строка истории статусов записи (FR-12).
type HistoryEntry struct {
	ID               int64      `json:"id"`
	StatusFromID     *int64     `json:"status_from_id"`
	StatusFromTitle  string     `json:"status_from_title"`
	StatusToID       int64      `json:"status_to_id"`
	StatusToTitle    string     `json:"status_to_title"`
	Note             string     `json:"note"`
	AmountForVozvrat *string    `json:"amount_for_vozvrat"`
	Refund           *string    `json:"refund"`
	ChangedByName    string     `json:"changed_by_name"`
	ChangedAt        *time.Time `json:"changed_at"`
	ChangeSource     string     `json:"change_source"`
}

// History — журнал переходов записи, новые сверху.
func (sr *StatusRepo) History(ctx context.Context, recordID int64) ([]HistoryEntry, error) {
	var rows []HistoryEntry
	err := sr.db.WithContext(ctx).Table("its_tb_5_15a_status_log l").
		Select(`l.id, l.status_from_id, coalesce(sf.title,'') as status_from_title,
			l.status_to_id, coalesce(st.title,'') as status_to_title,
			coalesce(l.note,'') as note, l.amount_for_vozvrat::text as amount_for_vozvrat,
			l.refund::text as refund,
			coalesce(u.short_name, u.login, '') as changed_by_name,
			l.changed_at, coalesce(l.change_source,'') as change_source`).
		Joins("left join its_risk_status sf on sf.id = l.status_from_id").
		Joins("left join its_risk_status st on st.id = l.status_to_id").
		Joins("left join users u on u.id = l.changed_by").
		Where("l.tb_5_15a_id = ?", recordID).
		Order("l.id desc").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// UserIDByIIN — id пользователя obm_evga по ИИН (changed_by журнала; ИИН не логируется).
func (sr *StatusRepo) UserIDByIIN(ctx context.Context, iin string) (*int64, error) {
	var rows []struct{ ID int64 }
	err := sr.db.WithContext(ctx).Table("users").
		Select("id").
		Where(`iin = ? and "in$trash" is null`, iin).
		Order("id").
		Limit(1).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0].ID, nil
}

// StatusLogTableExists — миграция 003 применена? (при отсутствии — понятная ошибка).
func (sr *StatusRepo) StatusLogTableExists(ctx context.Context) (bool, error) {
	var n int64
	err := sr.db.WithContext(ctx).Raw(
		"select count(*) from information_schema.tables where table_schema = 'public' and table_name = 'its_tb_5_15a_status_log'",
	).Scan(&n).Error
	return n > 0, err
}

// IsNotFoundErr — вспомогательное для транспорта.
func IsNotFoundErr(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }
