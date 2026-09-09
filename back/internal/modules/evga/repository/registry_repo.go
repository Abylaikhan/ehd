// Package repository — доступ к внешней БД obm_evga (PostgreSQL, ТОЛЬКО ЧТЕНИЕ).
// Никаких INSERT/UPDATE/DELETE/DDL (спека 007 FR-2); соединение открывается
// с default_transaction_read_only=on (wiring в internal/app).
package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"ehd-api/internal/modules/evga/domain"
)

// RegistryRepo — запросы реестра к obm_evga. Все выборки фильтруют
// платформенную корзину: "in$trash" is null (spec FR-8).
type RegistryRepo struct{ db *gorm.DB }

func NewRegistryRepo(db *gorm.DB) *RegistryRepo { return &RegistryRepo{db: db} }

// recordColumns — явный список колонок грида/карточки (Принцип 3: без SELECT *).
const recordColumns = `
	t.id, t.ppo_pp, t.paymentdate, coalesce(t.iin,'') as iin,
	coalesce(t.fm,'') as fm, coalesce(t.nm,'') as nm, coalesce(t.ft,'') as ft,
	coalesce(t.la1,'') as la1, t.amount_part::text as amount_part,
	coalesce(t.gu,'') as gu, coalesce(t.gu_bin,'') as gu_bin,
	coalesce(t.sendername,'') as sender_name, t.god, t.mes,
	t.its_risk_profile_id as profile_id, coalesce(p.title,'') as profile_title,
	t.its_risk_status_id as status_id, coalesce(s.title,'') as status_title,
	coalesce(t.risk_status_note,'') as status_note,
	t.its_departments_id as department_id, coalesce(d.title,'') as department_title,
	ln.notice_id, coalesce(ln.notice_num,'') as notice_num, coalesce(ln.out_num,'') as out_num,
	coalesce(ln.exec_due,'') as exec_due,
	t.refund::text as refund, t.amount_for_vozvrat::text as amount_for_vozvrat,
	t.its_activity_id as activity_id, t.is_gbdfl,
	coalesce(t.fl_fm,'') as fl_fm, coalesce(t.fl_nm,'') as fl_nm, coalesce(t.fl_ft,'') as fl_ft,
	t.created_at, t.updated_at`

// noticeLateral — последний действующий снимок записи → № уведомления и № исходящего.
// Связь по tb_5_15a_id (рабочий FK, ТЗ §8.4); its_out — через шапку уведомления
// (its_tb_5_15a.its_out_id в витрине — HTML-ссылка платформы, для связи непригодна).
const noticeLateral = `left join lateral (
	select n.id as notice_id, rq.docnum as notice_num, o.doc_num as out_num,
	       o.exec_due_time as exec_due
	  from its_risk_notice_5_15a sn
	  join its_risk_notice n on n.id = sn.its_risk_notice_id and n."in$trash" is null
	  left join its_req rq on rq.id = n.req_id
	  left join its_out o on o.id = n.its_out_id and o."in$trash" is null
	 where sn.tb_5_15a_id = t.id and sn."in$trash" is null
	 order by sn.id desc
	 limit 1
) ln on true`

// rowScan — плоская строка выборки; суммы сканируются строками (numeric без потери точности).
type rowScan struct {
	ID               int64
	PpoPp            string
	Paymentdate      *time.Time
	IIN              string
	FM               string
	NM               string
	FT               string
	LA1              string
	AmountPart       string
	GU               string
	GUBIN            string `gorm:"column:gu_bin"`
	SenderName       string
	God              *int64
	Mes              *int64
	ProfileID        int64
	ProfileTitle     string
	StatusID         *int64
	StatusTitle      string
	StatusNote       string
	DepartmentID     *int64
	DepartmentTitle  string
	NoticeID         *int64
	NoticeNum        string
	OutNum           string
	ExecDue          string
	Refund           *string
	AmountForVozvrat *string
	ActivityID       *int64
	IsGbdfl          *int16
	FlFM             string
	FlNM             string
	FlFT             string
	CreatedAt        *time.Time
	UpdatedAt        *time.Time
}

func (r rowScan) toDomain() domain.RiskRecord {
	rec := domain.RiskRecord{
		ID: r.ID, PpoPp: r.PpoPp, PaymentDate: r.Paymentdate,
		IIN: r.IIN, FM: r.FM, NM: r.NM, FT: r.FT, LA1: r.LA1,
		AmountPart: r.AmountPart, GU: r.GU, GUBIN: r.GUBIN, SenderName: r.SenderName,
		God: r.God, Mes: r.Mes,
		ProfileID: r.ProfileID, ProfileTitle: r.ProfileTitle,
		StatusID: r.StatusID, StatusTitle: r.StatusTitle, StatusNote: r.StatusNote,
		DepartmentID: r.DepartmentID, DepartmentTitle: r.DepartmentTitle,
		NoticeID: r.NoticeID, NoticeNum: r.NoticeNum, OutNum: r.OutNum,
		ExecDue:    r.ExecDue,
		ActivityID: r.ActivityID, IsGBDFL: r.IsGbdfl,
		FlFM: r.FlFM, FlNM: r.FlNM, FlFT: r.FlFT,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if r.Refund != nil {
		rec.Refund = *r.Refund
	}
	if r.AmountForVozvrat != nil {
		rec.AmountForVozvrat = *r.AmountForVozvrat
	}
	return rec
}

// base собирает FROM/JOIN/WHERE по фильтру и видимости. needNotice — включать LATERAL
// (для списка всегда; для COUNT только когда фильтр его требует).
func (rp *RegistryRepo) base(ctx context.Context, f domain.Filter, deptID *int64, needNotice bool) *gorm.DB {
	q := rp.db.WithContext(ctx).Table("its_tb_5_15a t").
		Joins("left join its_risk_profile p on p.id = t.its_risk_profile_id").
		Joins("left join its_risk_status s on s.id = t.its_risk_status_id").
		Joins("left join its_departments d on d.id = t.its_departments_id").
		Where(`t."in$trash" is null`)

	if needNotice || f.InNotice != nil || f.NoticeNum != "" {
		q = q.Joins(noticeLateral)
	}
	if deptID != nil {
		q = q.Where("t.its_departments_id = ?", *deptID)
	} else if f.DepartmentID != nil {
		q = q.Where("t.its_departments_id = ?", *f.DepartmentID)
	}
	if f.ProfileID != nil {
		q = q.Where("t.its_risk_profile_id = ?", *f.ProfileID)
	}
	if f.StatusID != nil {
		q = q.Where("t.its_risk_status_id = ?", *f.StatusID)
	}
	// EVGA-FR-082: «Подтверждено без решения» = текущий статус 11 (решение 7/12 уводит из 11).
	if f.ConfirmedNoDecision != nil && *f.ConfirmedNoDecision {
		q = q.Where("t.its_risk_status_id = ?", domain.StatusConfirmed)
	}
	if f.God != nil {
		q = q.Where("t.god = ?", *f.God)
	}
	if f.Mes != nil {
		q = q.Where("t.mes = ?", *f.Mes)
	}
	if f.PaymentFrom != nil {
		q = q.Where("t.paymentdate >= ?", *f.PaymentFrom)
	}
	if f.PaymentTo != nil {
		q = q.Where("t.paymentdate <= ?", *f.PaymentTo)
	}
	if f.GU != "" {
		q = q.Where("t.gu = ?", f.GU)
	}
	if f.SenderName != "" {
		q = q.Where("t.sendername ilike ?", "%"+f.SenderName+"%")
	}
	if f.IIN != "" {
		q = q.Where("t.iin = ?", f.IIN)
	}
	if f.AmountFrom != nil {
		q = q.Where("t.amount_part >= ?", *f.AmountFrom)
	}
	if f.AmountTo != nil {
		q = q.Where("t.amount_part <= ?", *f.AmountTo)
	}
	if f.InNotice != nil {
		if *f.InNotice {
			q = q.Where("ln.notice_id is not null")
		} else {
			q = q.Where("ln.notice_id is null")
		}
	}
	if f.NoticeNum != "" {
		q = q.Where("ln.notice_num ilike ?", "%"+f.NoticeNum+"%")
	}
	return q
}

// List — страница реестра + общее количество по той же выборке (spec FR-5..7).
func (rp *RegistryRepo) List(ctx context.Context, f domain.Filter, deptID *int64, page domain.Page) (domain.RecordPage, error) {
	sortExpr, err := domain.SortExpr(page.Sort, page.Order)
	if err != nil {
		return domain.RecordPage{}, err
	}
	page = page.Normalize()

	var total int64
	if err := rp.base(ctx, f, deptID, false).Count(&total).Error; err != nil {
		return domain.RecordPage{}, err
	}

	var rows []rowScan
	err = rp.base(ctx, f, deptID, true).
		Select(recordColumns).
		Order(sortExpr).
		Limit(page.Size).
		Offset((page.Page - 1) * page.Size).
		Scan(&rows).Error
	if err != nil {
		return domain.RecordPage{}, err
	}

	items := make([]domain.RiskRecord, len(rows))
	for i, r := range rows {
		items[i] = r.toDomain()
	}
	return domain.RecordPage{Items: items, Total: total}, nil
}

// ListAll — выборка для экспорта: те же фильтры, лимит limit+1 (для признака truncated).
func (rp *RegistryRepo) ListAll(ctx context.Context, f domain.Filter, deptID *int64, sort, order string, limit int) ([]domain.RiskRecord, error) {
	sortExpr, err := domain.SortExpr(sort, order)
	if err != nil {
		return nil, err
	}
	var rows []rowScan
	err = rp.base(ctx, f, deptID, true).
		Select(recordColumns).
		Order(sortExpr).
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	items := make([]domain.RiskRecord, len(rows))
	for i, r := range rows {
		items[i] = r.toDomain()
	}
	return items, nil
}

// Get — карточка записи в пределах видимости (spec FR-9).
func (rp *RegistryRepo) Get(ctx context.Context, id int64, deptID *int64) (domain.RiskRecord, error) {
	var rows []rowScan
	err := rp.base(ctx, domain.Filter{}, deptID, true).
		Select(recordColumns).
		Where("t.id = ?", id).
		Limit(1).
		Scan(&rows).Error
	if err != nil {
		return domain.RiskRecord{}, err
	}
	if len(rows) == 0 {
		return domain.RiskRecord{}, domain.ErrNotFound
	}
	return rows[0].toDomain(), nil
}

// refRow — строка справочника.
type refRow struct {
	ID    int64
	Code  string
	Title string
}

func toRefs(rows []refRow) []domain.Reference {
	out := make([]domain.Reference, len(rows))
	for i, r := range rows {
		out[i] = domain.Reference{ID: r.ID, Code: r.Code, Title: r.Title}
	}
	return out
}

// Statuses — действующие статусы витрины: "in$trash" is null, работа по id (EVGA-BR-010/011).
func (rp *RegistryRepo) Statuses(ctx context.Context) ([]domain.Reference, error) {
	var rows []refRow
	err := rp.db.WithContext(ctx).Table("its_risk_status").
		Select(`id, coalesce(code,'') as code, coalesce(title,'') as title`).
		Where(`"in$trash" is null and title <> '-'`).
		Order("id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return toRefs(rows), nil
}

// Profiles — профили риска.
func (rp *RegistryRepo) Profiles(ctx context.Context) ([]domain.Reference, error) {
	var rows []refRow
	err := rp.db.WithContext(ctx).Table("its_risk_profile").
		Select(`id, coalesce(code,'') as code, coalesce(title,'') as title`).
		Where(`"in$trash" is null`).
		Order("id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return toRefs(rows), nil
}

// Activities — аудиторские мероприятия (переход в «Аудит», спека 008/006-ui).
func (rp *RegistryRepo) Activities(ctx context.Context) ([]domain.Reference, error) {
	var rows []refRow
	err := rp.db.WithContext(ctx).Table("its_activity").
		Select(`id, coalesce(code,'') as code, coalesce(title,'') as title`).
		Where(`"in$trash" is null`).
		Order("id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return toRefs(rows), nil
}

// Departments — департаменты ДВГА (куратору/админу для фильтра).
func (rp *RegistryRepo) Departments(ctx context.Context) ([]domain.Reference, error) {
	var rows []refRow
	err := rp.db.WithContext(ctx).Table("its_departments").
		Select(`id, coalesce(code,'') as code, coalesce(title,'') as title`).
		Where(`"in$trash" is null`).
		Order("title").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return toRefs(rows), nil
}

// DepartmentByUserIIN — департамент пользователя obm_evga по ИИН (spec FR-3, OQ-11).
// ИИН в запрос попадает параметром и не логируется.
func (rp *RegistryRepo) DepartmentByUserIIN(ctx context.Context, iin string) (*int64, string, error) {
	var rows []struct {
		DepartmentID    *int64
		DepartmentTitle string
	}
	err := rp.db.WithContext(ctx).Table("users u").
		Select(`u.its_departments_id as department_id, coalesce(d.title,'') as department_title`).
		Joins(`left join its_departments d on d.id = u.its_departments_id and d."in$trash" is null`).
		Where(`u.iin = ? and u."in$trash" is null`, iin).
		Order("u.id").
		Limit(1).
		Scan(&rows).Error
	if err != nil {
		return nil, "", err
	}
	if len(rows) == 0 {
		return nil, "", nil
	}
	return rows[0].DepartmentID, rows[0].DepartmentTitle, nil
}

// Ping — проверка доступности внешней БД (readyz, spec FR-12).
func (rp *RegistryRepo) Ping(ctx context.Context) error {
	sqlDB, err := rp.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// TransactionReadOnly — фактический режим сессии (интеграционная проверка FR-2).
func (rp *RegistryRepo) TransactionReadOnly(ctx context.Context) (string, error) {
	var v string
	err := rp.db.WithContext(ctx).Raw("show default_transaction_read_only").Scan(&v).Error
	return v, err
}

// IsNotFound — вспомогательное для маппинга ошибок транспорта.
func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }
