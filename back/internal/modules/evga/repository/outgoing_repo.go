package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ehd-api/internal/modules/evga/domain"
)

// OutgoingRepo — исходящее письмо и регистрация (obm_evga; спека 011).
// Реальную отправку в ЕСЭДО и заполнение doc_num/doc_date выполняет платформа
// (ответ №3-7); мы создаём its_out и реагируем на появление номера.
type OutgoingRepo struct{ db *gorm.DB }

func NewOutgoingRepo(db *gorm.DB) *OutgoingRepo { return &OutgoingRepo{db: db} }

// OutInfo — данные исходящего для карточки/вотчера.
type OutInfo struct {
	ID          int64
	DocNum      string
	DocDate     *time.Time
	ExecDueTime string
}

// CreateOutgoing создаёт its_out и связывает с уведомлением (FR-1/2).
// Возвращает id исходящего. Повторное создание → ErrRouteStateInvalid.
func (or *OutgoingRepo) CreateOutgoing(ctx context.Context, noticeID int64, docnum string, createdBy *int64) (int64, error) {
	var outID int64
	err := or.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// блокировка шапки: гонка двух создателей исходящего
		var rows []struct {
			ItsOutID     *int64
			DepartmentID *int64
			ItsCliID     *int64
			ReqID        *int64
		}
		err := tx.Raw(
			`select n.its_out_id, n.its_departments_id as department_id, d.its_cli_id, n.req_id
			   from its_risk_notice n
			   left join its_departments d on d.id = n.its_departments_id
			  where n.id = ? and n."in$trash" is null
			  for update of n`, noticeID,
		).Scan(&rows).Error
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return domain.ErrNotFound
		}
		if rows[0].ItsOutID != nil {
			return domain.ErrRouteStateInvalid // исходящее уже создано (FR-2)
		}

		out := map[string]any{
			"subject":            fmt.Sprintf("Уведомление №%s", docnum),
			"its_risk_notice_id": noticeID,
			"created_at":         time.Now(),
		}
		if rows[0].ReqID != nil {
			out["req_id"] = *rows[0].ReqID
		}
		if rows[0].ItsCliID != nil {
			out["sender_cli_id"] = *rows[0].ItsCliID
		}
		if createdBy != nil {
			out["created_by"] = *createdBy
		}
		if err := tx.Table("its_out").Create(out).Error; err != nil {
			return err
		}
		if err := tx.Raw(`select id from its_out where its_risk_notice_id = ? order by id desc limit 1`, noticeID).
			Scan(&outID).Error; err != nil {
			return err
		}
		return tx.Exec(`update its_risk_notice set its_out_id = ? where id = ?`, outID, noticeID).Error
	})
	return outID, err
}

// Out — исходящее уведомления (nil, если не создано).
func (or *OutgoingRepo) Out(ctx context.Context, noticeID int64) (*OutInfo, error) {
	var rows []OutInfo
	err := or.db.WithContext(ctx).Table("its_out o").
		Select(`o.id, coalesce(o.doc_num,'') as doc_num, o.doc_date, coalesce(o.exec_due_time,'') as exec_due_time`).
		Joins("join its_risk_notice n on n.its_out_id = o.id").
		Where(`n.id = ? and o."in$trash" is null`, noticeID).
		Limit(1).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

// SetRegistrationData — симуляция канцелярии (админ, FR-6): номер и дата в its_out.
func (or *OutgoingRepo) SetRegistrationData(ctx context.Context, noticeID int64, docNum string, docDate time.Time) error {
	res := or.db.WithContext(ctx).Exec(
		`update its_out o set doc_num = ?, doc_date = ?
		  from its_risk_notice n
		 where n.its_out_id = o.id and n.id = ? and coalesce(o.doc_num,'') = ''`,
		docNum, docDate, noticeID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrRouteStateInvalid
	}
	return nil
}

// RegisterResult — итог транзакции регистрации (FR-3).
type RegisterResult struct {
	Applied   bool // false — уже зарегистрировано ранее (FR-4)
	Updated   int  // записей снимка переведено в статус 4
	Cascaded  int  // записей-соседей по ppo_pp переведено в 4 (EVGA-FR-067а)
	ExecDue   string
	OutDocNum string
}

// Register — транзакция §11.3: статус 4 записям уведомления + каскад по платежу,
// контрольный срок (+10 будних, №3-5), статус заявки → 3 «Отправлен объекту аудита».
func (or *OutgoingRepo) Register(ctx context.Context, noticeID int64) (RegisterResult, error) {
	res := RegisterResult{}
	err := or.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// шапка + заявка + исходящее под блокировкой
		var meta []struct {
			ReqID   *int64
			StatID  *int64
			Docnum  string
			OutID   *int64
			DocNum  string
			DocDate *time.Time
		}
		err := tx.Raw(
			`select n.req_id, rq.stat_id, coalesce(rq.docnum,'') as docnum,
			        n.its_out_id as out_id, coalesce(o.doc_num,'') as doc_num, o.doc_date
			   from its_risk_notice n
			   left join its_req rq on rq.id = n.req_id
			   left join its_out o on o.id = n.its_out_id
			  where n.id = ? and n."in$trash" is null
			  for update of n`, noticeID,
		).Scan(&meta).Error
		if err != nil {
			return err
		}
		if len(meta) == 0 {
			return domain.ErrNotFound
		}
		m := meta[0]
		if m.StatID != nil && *m.StatID == domain.ReqStatSent {
			return nil // идемпотентность (FR-4)
		}
		if m.ReqID == nil || m.OutID == nil || m.DocNum == "" || m.DocDate == nil {
			return domain.ErrRouteStateInvalid // регистрация ещё не произошла
		}
		res.OutDocNum = m.DocNum

		// записи снимка уведомления
		var recs []struct {
			ID    int64
			PpoPp string
		}
		if err := tx.Raw(
			`select t.id, coalesce(t.ppo_pp,'') as ppo_pp
			   from its_risk_notice_5_15a sn
			   join its_tb_5_15a t on t.id = sn.tb_5_15a_id and t."in$trash" is null
			  where sn.its_risk_notice_id = ? and sn."in$trash" is null
			  for update of t`, noticeID,
		).Scan(&recs).Error; err != nil {
			return err
		}
		ids := make([]int64, 0, len(recs))
		ppo := make([]string, 0, len(recs))
		for _, r := range recs {
			ids = append(ids, r.ID)
			if r.PpoPp != "" {
				ppo = append(ppo, r.PpoPp)
			}
		}

		note := fmt.Sprintf("Уведомление №%s направлено, исходящее №%s", m.Docnum, m.DocNum)
		apply := func(recIDs []int64) (int, error) {
			applied := 0
			for _, id := range recIDs {
				r := tx.Exec(
					`update its_tb_5_15a set its_risk_status_id = ?, risk_status_note = ?, updated_at = now()
					  where id = ? and its_risk_status_id = ?`,
					domain.StatusNoticeSent, note, id, domain.StatusInWork)
				if r.Error != nil {
					return applied, r.Error
				}
				if r.RowsAffected == 0 {
					continue // не в статусе 1 — не трогаем (каскад-правило допустимости)
				}
				if err := tx.Exec(
					`insert into its_tb_5_15a_status_log (tb_5_15a_id, status_from_id, status_to_id, note, change_source)
					 values (?, ?, ?, ?, ?)`,
					id, domain.StatusInWork, domain.StatusNoticeSent, note, domain.ChangeSourceSystem,
				).Error; err != nil {
					return applied, err
				}
				applied++
			}
			return applied, nil
		}

		if res.Updated, err = apply(ids); err != nil {
			return err
		}

		// каскад по платежу: записи тех же ppo_pp других профилей (EVGA-FR-067а)
		if len(ppo) > 0 {
			var sibs []struct{ ID int64 }
			if err := tx.Raw(
				`select id from its_tb_5_15a
				  where ppo_pp in ? and id not in ? and "in$trash" is null and its_risk_status_id = ?
				  for update`, ppo, ids, domain.StatusInWork,
			).Scan(&sibs).Error; err != nil {
				return err
			}
			sibIDs := make([]int64, len(sibs))
			for i, s := range sibs {
				sibIDs[i] = s.ID
			}
			if res.Cascaded, err = apply(sibIDs); err != nil {
				return err
			}
		}

		// контрольный срок: doc_date + 10 будних дней (№3-5) → its_out.exec_due_time
		due := domain.AddBusinessDays(*m.DocDate, 10).Format("2006-01-02")
		res.ExecDue = due
		if err := tx.Exec(`update its_out set exec_due_time = ? where id = ?`, due, *m.OutID).Error; err != nil {
			return err
		}

		// статус заявки → «Отправлен объекту аудита» (=«Исполнен»)
		if err := tx.Exec(`update its_req set stat_id = ? where id = ?`, domain.ReqStatSent, *m.ReqID).Error; err != nil {
			return err
		}
		res.Applied = true
		return nil
	})
	return res, err
}

// PendingRegistration — уведомления, у которых платформа уже проставила номер
// регистрации, а наша транзакция §11.3 ещё не выполнена (вотчер, FR-5).
func (or *OutgoingRepo) PendingRegistration(ctx context.Context, limit int) ([]int64, error) {
	var rows []struct{ ID int64 }
	err := or.db.WithContext(ctx).Raw(
		`select n.id
		   from its_risk_notice n
		   join its_req rq on rq.id = n.req_id and rq.stat_id = ?
		   join its_out o on o.id = n.its_out_id and o."in$trash" is null
		  where n."in$trash" is null
		    and coalesce(o.doc_num,'') <> '' and o.doc_date is not null
		  order by n.id
		  limit ?`, domain.ReqStatSigning, limit,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]int64, len(rows))
	for i, r := range rows {
		out[i] = r.ID
	}
	return out, nil
}
