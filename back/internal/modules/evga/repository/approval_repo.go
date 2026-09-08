package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"ehd-api/internal/modules/evga/domain"
)

// ApprovalRepo — операции согласования в obm_evga (спека 010):
// номер уведомления, статус заявки, подписант, сотрудники департамента.
type ApprovalRepo struct{ db *gorm.DB }

func NewApprovalRepo(db *gorm.DB) *ApprovalRepo { return &ApprovalRepo{db: db} }

// NoticeMeta — данные уведомления для операций согласования.
type NoticeMeta struct {
	ID           int64
	DepartmentID *int64
	ReqID        *int64
	StatID       *int64
	Docnum       string
	CreatedBy    *int64 // инициатор (obm_evga.users.id) — предзаполнение этапа «Создать исходящее»
}

// Meta — уведомление с заявкой; deptID — видимость аудитора.
func (ar *ApprovalRepo) Meta(ctx context.Context, noticeID int64, deptID *int64) (NoticeMeta, error) {
	q := ar.db.WithContext(ctx).Table("its_risk_notice n").
		Select(`n.id, n.its_departments_id as department_id, n.req_id, rq.stat_id,
			coalesce(rq.docnum,'') as docnum, n.created_by`).
		Joins("left join its_req rq on rq.id = n.req_id").
		Where(`n.id = ? and n."in$trash" is null`, noticeID)
	if deptID != nil {
		q = q.Where("n.its_departments_id = ?", *deptID)
	}
	var rows []NoticeMeta
	if err := q.Limit(1).Scan(&rows).Error; err != nil {
		return NoticeMeta{}, err
	}
	if len(rows) == 0 {
		return NoticeMeta{}, domain.ErrNotFound
	}
	return rows[0], nil
}

// IssueNumber — присваивает номер уведомлению, если его ещё нет (EVGA-FR-071…074):
// блокировка строки департамента FOR UPDATE, годовой сброс, формат по region_code.
func (ar *ApprovalRepo) IssueNumber(ctx context.Context, meta NoticeMeta, now time.Time) (string, error) {
	if meta.Docnum != "" {
		return meta.Docnum, nil // повторная отправка: номер сохраняется (AT-12)
	}
	if meta.DepartmentID == nil || meta.ReqID == nil {
		return "", domain.ErrNumberingUnavailable
	}
	var num string
	err := ar.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// блокировка счётчика департамента (EVGA-BR-021, AT-10)
		var dep []struct {
			NoticeNum     *int64
			NoticeNumYear *int32
			RegionCode    *string
		}
		err := tx.Raw(
			`select d.notice_num, d.notice_num_year, r.region_code
			   from its_departments d
			   left join its_regions r on r.id = d.its_regions_id
			  where d.id = ?
			  for update of d`, *meta.DepartmentID,
		).Scan(&dep).Error
		if err != nil {
			return err
		}
		if len(dep) == 0 {
			return domain.ErrNumberingUnavailable
		}
		region := ""
		if dep[0].RegionCode != nil {
			region = *dep[0].RegionCode
		}
		counter := int64(0)
		if dep[0].NoticeNum != nil {
			counter = *dep[0].NoticeNum
		}
		seq := domain.NextNoticeSeq(counter, dep[0].NoticeNumYear, now.Year())
		formatted, err := domain.FormatNoticeNumber(region, now.Year(), seq)
		if err != nil {
			return err
		}
		if err := tx.Exec(
			`update its_departments set notice_num = ?, notice_num_year = ? where id = ?`,
			seq, now.Year(), *meta.DepartmentID,
		).Error; err != nil {
			return err
		}
		if err := tx.Exec(`update its_req set docnum = ? where id = ?`, formatted, *meta.ReqID).Error; err != nil {
			return err
		}
		num = formatted
		return nil
	})
	return num, err
}

// SetReqStatus — статус заявки уведомления (its_req.stat_id).
func (ar *ApprovalRepo) SetReqStatus(ctx context.Context, reqID int64, statID int64) error {
	return ar.db.WithContext(ctx).Exec(
		`update its_req set stat_id = ? where id = ?`, statID, reqID).Error
}

// SetSigner — фактический подписант уведомления (AT-13; В1: подписал последний согласующий).
func (ar *ApprovalRepo) SetSigner(ctx context.Context, noticeID int64, signerObmID int64, at time.Time) error {
	return ar.db.WithContext(ctx).Exec(
		`update its_risk_notice set signer_id = ?, signer_date = ? where id = ?`,
		signerObmID, at, noticeID).Error
}

// DeptUsers — сотрудники департамента для маршрута (FR-3): только с заполненным ИИН
// (иначе не смогут действовать в ЕХД), живые записи.
func (ar *ApprovalRepo) DeptUsers(ctx context.Context, deptID int64, q string) ([]domain.ObmUser, error) {
	query := ar.db.WithContext(ctx).Table("users").
		Select(`id, coalesce(short_name, login, '') as name, coalesce(login,'') as login`).
		Where(`its_departments_id = ? and "in$trash" is null and coalesce(iin,'') <> ''`, deptID).
		Order("name").
		Limit(30)
	if q != "" {
		query = query.Where("(short_name ilike ? or login ilike ?)", "%"+q+"%", "%"+q+"%")
	}
	var rows []struct {
		ID    int64
		Name  string
		Login string
	}
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ObmUser, len(rows))
	for i, r := range rows {
		out[i] = domain.ObmUser{ID: r.ID, Name: r.Name, Login: r.Login}
	}
	return out, nil
}

// UserNames — имена сотрудников по id (для снапшота имён в шагах маршрута).
func (ar *ApprovalRepo) UserNames(ctx context.Context, ids []int64) (map[int64]string, error) {
	if len(ids) == 0 {
		return map[int64]string{}, nil
	}
	var rows []struct {
		ID   int64
		Name string
	}
	err := ar.db.WithContext(ctx).Table("users").
		Select(`id, coalesce(short_name, login, '') as name`).
		Where("id in ?", ids).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[int64]string, len(rows))
	for _, r := range rows {
		out[r.ID] = r.Name
	}
	return out, nil
}

// DeptOfUsers — проверка принадлежности сотрудников департаменту (FR-3).
func (ar *ApprovalRepo) DeptOfUsers(ctx context.Context, ids []int64, deptID int64) (bool, error) {
	if len(ids) == 0 {
		return true, nil
	}
	var n int64
	err := ar.db.WithContext(ctx).Table("users").
		Where(`id in ? and its_departments_id = ? and "in$trash" is null`, ids, deptID).
		Count(&n).Error
	return n == int64(len(ids)), err
}
