package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ehd-api/internal/modules/evga/domain"
)

// NoticeRepo — формирование и просмотр уведомлений (спека 009).
// Запись — только INSERT новых объектов уведомления и in$trash при удалении проекта;
// записи витрины НЕ изменяются (статус 4 — фаза 7).
type NoticeRepo struct{ db *gorm.DB }

func NewNoticeRepo(db *gorm.DB) *NoticeRepo { return &NoticeRepo{db: db} }

// busyLateral — действующее уведомление, занимающее запись витрины (spec 009 FR-2):
// снимок жив, шапка жива, заявка не отозвана (stat_id ≠ 15, ответ В4 07.09.2026).
const busyLateral = `left join lateral (
	select n.id as notice_id, rq.docnum as busy_docnum
	  from its_risk_notice_5_15a sn
	  join its_risk_notice n on n.id = sn.its_risk_notice_id and n."in$trash" is null
	  left join its_req rq on rq.id = n.req_id
	 where sn.tb_5_15a_id = t.id and sn."in$trash" is null
	   and (rq.id is null or (coalesce(rq."in$trash", 0) = 0 and coalesce(rq.stat_id, 0) <> 15))
	 order by sn.id desc
	 limit 1
) busy on true`

// previewRow — запись витрины с данными для классификации и снимка.
type previewRow struct {
	ID          int64
	GU          string
	GUBIN       string `gorm:"column:gu_bin"`
	SenderName  string
	PpoPp       string
	Paymentdate *time.Time
	IIN         string
	FM          string
	NM          string
	FT          string
	LA1         string
	AmountPart  string
	StatusID    *int64
	ProfileID   int64
	BusyNotice  *int64
	BusyDocnum  *string
}

// fetchForNotice загружает записи с блокировкой (lock=true — внутри транзакции создания).
func (nr *NoticeRepo) fetchForNotice(tx *gorm.DB, ids []int64, deptID *int64, lock bool) ([]previewRow, error) {
	q := tx.Table("its_tb_5_15a t").
		Select(`t.id, coalesce(t.gu,'') as gu, coalesce(t.gu_bin,'') as gu_bin,
			coalesce(t.sendername,'') as sender_name, t.ppo_pp, t.paymentdate,
			coalesce(t.iin,'') as iin, coalesce(t.fm,'') as fm, coalesce(t.nm,'') as nm,
			coalesce(t.ft,'') as ft, coalesce(t.la1,'') as la1,
			t.amount_part::text as amount_part, t.its_risk_status_id as status_id,
			t.its_risk_profile_id as profile_id,
			busy.notice_id as busy_notice, busy.busy_docnum`).
		Joins(busyLateral).
		Where(`t.id in ? and t."in$trash" is null`, ids)
	if deptID != nil {
		q = q.Where("t.its_departments_id = ?", *deptID)
	}
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "t"}})
	}
	var rows []previewRow
	err := q.Scan(&rows).Error
	return rows, err
}

// classify — валидные записи и отклонения по правилам FR-2.
func classify(ids []int64, rows []previewRow) (valid []previewRow, rejected []domain.RejectedRecord) {
	byID := make(map[int64]previewRow, len(rows))
	for _, r := range rows {
		byID[r.ID] = r
	}
	for _, id := range ids {
		r, ok := byID[id]
		switch {
		case !ok:
			rejected = append(rejected, domain.RejectedRecord{ID: id, Code: domain.RejectNotFound,
				Reason: "Запись не найдена или относится к другому департаменту"})
		case r.BusyNotice != nil:
			num := ""
			if r.BusyDocnum != nil {
				num = *r.BusyDocnum
			}
			reason := "Запись уже включена в действующее уведомление"
			if num != "" {
				reason += " №" + num
			}
			rejected = append(rejected, domain.RejectedRecord{ID: id, Code: domain.RejectInNotice, Reason: reason, NoticeNum: num})
		case r.StatusID == nil || *r.StatusID != domain.StatusInWork:
			rejected = append(rejected, domain.RejectedRecord{ID: id, Code: domain.RejectBadStatus,
				Reason: "Допустимы только записи в статусе «В работе у ДВГА»"})
		case strings.TrimSpace(r.GU) == "":
			rejected = append(rejected, domain.RejectedRecord{ID: id, Code: domain.RejectEmptyGU,
				Reason: "Не заполнен код ГУ отправителя — запись не может войти в уведомление"})
		default:
			valid = append(valid, r)
		}
	}
	return valid, rejected
}

// groupByGU — группировка валидных записей по коду ГУ (EVGA-FR-031).
func groupByGU(rows []previewRow) []domain.NoticeGroup {
	order := []string{}
	byGU := map[string][]previewRow{}
	for _, r := range rows {
		if _, ok := byGU[r.GU]; !ok {
			order = append(order, r.GU)
		}
		byGU[r.GU] = append(byGU[r.GU], r)
	}
	groups := make([]domain.NoticeGroup, 0, len(order))
	for _, gu := range order {
		g := domain.NoticeGroup{GU: gu}
		var total float64
		for _, r := range byGU[gu] {
			if g.SenderName == "" {
				g.SenderName = r.SenderName
			}
			if g.GUBIN == "" {
				g.GUBIN = r.GUBIN
			}
			if v, err := strconv.ParseFloat(r.AmountPart, 64); err == nil {
				total += v
			}
			fio := strings.TrimSpace(strings.Join([]string{r.FM, r.NM, r.FT}, " "))
			g.Records = append(g.Records, domain.GroupRecord{
				ID: r.ID, PpoPp: r.PpoPp, PaymentDate: r.Paymentdate,
				IIN: r.IIN, FIO: fio, AmountPart: r.AmountPart,
			})
		}
		g.Count = len(g.Records)
		g.TotalSum = fmt.Sprintf("%.2f", total)
		groups = append(groups, g)
	}
	return groups
}

// Preview — группировка отмеченных записей + отклонения (spec 009 FR-1/2).
func (nr *NoticeRepo) Preview(ctx context.Context, ids []int64, deptID *int64) ([]domain.NoticeGroup, []domain.RejectedRecord, error) {
	rows, err := nr.fetchForNotice(nr.db.WithContext(ctx), ids, deptID, false)
	if err != nil {
		return nil, nil, err
	}
	valid, rejected := classify(ids, rows)
	return groupByGU(valid), rejected, nil
}

// SearchCli — поиск адресата в справочнике организаций ЕСЭДО (spec 009 FR-3).
func (nr *NoticeRepo) SearchCli(ctx context.Context, q string) ([]domain.CliOrg, error) {
	var rows []struct {
		ID     int64
		Code   string
		BinIIN string `gorm:"column:bin_iin"`
		Title  string
	}
	err := nr.db.WithContext(ctx).Table("its_cli").
		Select(`id, coalesce(code,'') as code, coalesce(bin_iin,'') as bin_iin, coalesce(title,'') as title`).
		Where(`"in$trash" is null`).
		Where(`(title ilike ? or title_kk ilike ? or code = ? or bin_iin = ?)`,
			"%"+q+"%", "%"+q+"%", q, q).
		Order("title").
		Limit(20).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.CliOrg, len(rows))
	for i, r := range rows {
		out[i] = domain.CliOrg{ID: r.ID, Code: r.Code, BinIIN: r.BinIIN, Title: r.Title}
	}
	return out, nil
}

// CreateNotices — создание проектов уведомлений одной транзакцией (spec 009 FR-4..6).
// createdBy — id пользователя obm_evga (может быть nil), noticeTxt — текст шаблона.
func (nr *NoticeRepo) CreateNotices(
	ctx context.Context,
	groups []domain.CreateGroupInput,
	deptID *int64,
	ownerDeptID int64,
	createdBy *int64,
	noticeTxt string,
) ([]domain.CreatedNotice, []domain.RejectedRecord, error) {
	var created []domain.CreatedNotice
	var rejected []domain.RejectedRecord

	err := nr.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, g := range groups {
			rows, err := nr.fetchForNotice(tx, g.RecordIDs, deptID, true)
			if err != nil {
				return err
			}
			valid, rej := classify(g.RecordIDs, rows)
			rejected = append(rejected, rej...)
			if len(valid) == 0 {
				continue
			}
			// сервер перегруппирует сам: на каждую фактическую группу ГУ — своё уведомление (FR-4)
			for _, grp := range groupByGU(valid) {
				noticeID, err := nr.insertNotice(tx, grp, g.CliID, ownerDeptID, createdBy, noticeTxt)
				if err != nil {
					return err
				}
				created = append(created, domain.CreatedNotice{NoticeID: noticeID, GU: grp.GU, Records: grp.Count})
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return created, rejected, nil
}

// insertNotice — заявка + шапка + снимок + адресат для одной группы (FR-5).
func (nr *NoticeRepo) insertNotice(tx *gorm.DB, grp domain.NoticeGroup, cliID int64, deptID int64, createdBy *int64, noticeTxt string) (int64, error) {
	now := time.Now()

	// заявка платформы: статус «Проект создан», номер не присваивается (фаза 6)
	var req struct{ ID int64 }
	if err := tx.Raw(
		`insert into its_req (title, stat_id, created_at, created_by) values (?, ?, ?, ?) returning id`,
		"Уведомление "+grp.GU, domain.ReqStatDraft, now, createdBy,
	).Scan(&req).Error; err != nil {
		return 0, err
	}

	// шапка уведомления
	var notice struct {
		ID   int64
		UUID string `gorm:"column:sys$uuid"`
	}
	if err := tx.Raw(
		`insert into its_risk_notice (created_by, created_at, its_departments_id, notice_txt, req_id)
		 values (?, ?, ?, ?, ?) returning id, "sys$uuid"`,
		createdBy, now, deptID, noticeTxt, req.ID,
	).Scan(&notice).Error; err != nil {
		return 0, err
	}

	// best-effort совместимость с платформой: полиморфная привязка заявки (spec Clarifications)
	if err := tx.Exec(
		`update its_req set entity_pk = ?, entity_uuid = ? where id = ?`,
		notice.ID, notice.UUID, req.ID,
	).Error; err != nil {
		return 0, err
	}

	// снимок строк (EVGA-FR-038 + gu/gu_bin из миграции 001; поле tb_5_15a не заполняется, §8.4)
	ids := make([]int64, len(grp.Records))
	for i, r := range grp.Records {
		ids[i] = r.ID
	}
	if err := tx.Exec(
		`insert into its_risk_notice_5_15a
		   (its_risk_notice_id, tb_5_15a_id, its_risk_profile_id, gu, gu_bin, ppo_pp,
		    paymentdate, iin, fm, nm, ft, la1, amount_part, sendername)
		 select ?, t.id, t.its_risk_profile_id, t.gu, t.gu_bin, t.ppo_pp,
		        t.paymentdate, t.iin, t.fm, t.nm, t.ft, t.la1, t.amount_part, t.sendername
		   from its_tb_5_15a t where t.id in ? order by t.id`,
		notice.ID, ids,
	).Error; err != nil {
		return 0, err
	}

	// адресат (EVGA-FR-039)
	if err := tx.Exec(
		`insert into its_risk_notice_recepient (its_risk_notice_id, its_cli_id, nn) values (?, ?, 1)`,
		notice.ID, cliID,
	).Error; err != nil {
		return 0, err
	}
	return notice.ID, nil
}

// noticeListSelect — общие связки списка/карточки уведомлений.
const noticeListSelect = `
	n.id, coalesce(rq.docnum,'') as docnum, rq.stat_id as status_id,
	coalesce(st.title,'') as status_title,
	coalesce(agg.gu,'') as gu, coalesce(agg.sendername,'') as sender_name,
	coalesce(rc.title,'') as recipient,
	coalesce(agg.rows_count,0) as rows_count, coalesce(agg.total_sum,'0') as total_sum,
	n.created_at, coalesce(d.title,'') as department`

const noticeListJoins = `
	left join its_req rq on rq.id = n.req_id
	left join its_req_stat st on st.id = rq.stat_id
	left join its_departments d on d.id = n.its_departments_id
	left join lateral (
		select min(sn.gu) as gu, min(sn.sendername) as sendername,
		       count(*) as rows_count, coalesce(sum(sn.amount_part),0)::text as total_sum
		  from its_risk_notice_5_15a sn
		 where sn.its_risk_notice_id = n.id and sn."in$trash" is null
	) agg on true
	left join lateral (
		select c.title
		  from its_risk_notice_recepient r
		  join its_cli c on c.id = r.its_cli_id
		 where r.its_risk_notice_id = n.id and r."in$trash" is null
		 order by r.nn nulls last, r.id
		 limit 1
	) rc on true`

type noticeListRow struct {
	ID          int64
	Docnum      string
	StatusID    *int64
	StatusTitle string
	GU          string
	SenderName  string
	Recipient   string
	RowsCount   int64
	TotalSum    string
	CreatedAt   *time.Time
	Department  string
}

func (r noticeListRow) toDomain() domain.NoticeListItem {
	return domain.NoticeListItem{
		ID: r.ID, Docnum: r.Docnum, StatusID: r.StatusID, StatusTitle: r.StatusTitle,
		GU: r.GU, SenderName: r.SenderName, Recipient: r.Recipient,
		RowsCount: r.RowsCount, TotalSum: r.TotalSum, CreatedAt: r.CreatedAt, Department: r.Department,
	}
}

// ListNotices — уведомления департамента/всех (spec 009 FR-7).
func (nr *NoticeRepo) ListNotices(ctx context.Context, deptID, filterDept, statusID *int64, docnum string, page domain.Page) ([]domain.NoticeListItem, int64, error) {
	base := func() *gorm.DB {
		q := nr.db.WithContext(ctx).Table("its_risk_notice n").
			Joins(noticeListJoins).
			Where(`n."in$trash" is null`)
		if deptID != nil {
			q = q.Where("n.its_departments_id = ?", *deptID)
		} else if filterDept != nil {
			q = q.Where("n.its_departments_id = ?", *filterDept)
		}
		if statusID != nil {
			q = q.Where("rq.stat_id = ?", *statusID)
		}
		if docnum != "" {
			q = q.Where("rq.docnum ilike ?", "%"+docnum+"%")
		}
		return q
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page = page.Normalize()
	var rows []noticeListRow
	err := base().Select(noticeListSelect).
		Order("n.id desc").
		Limit(page.Size).Offset((page.Page - 1) * page.Size).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	items := make([]domain.NoticeListItem, len(rows))
	for i, r := range rows {
		items[i] = r.toDomain()
	}
	return items, total, nil
}

// GetNotice — карточка уведомления со строками снимка (spec 009 FR-8).
func (nr *NoticeRepo) GetNotice(ctx context.Context, id int64, deptID *int64) (domain.NoticeCard, error) {
	q := nr.db.WithContext(ctx).Table("its_risk_notice n").
		Joins(noticeListJoins).
		Where(`n.id = ? and n."in$trash" is null`, id)
	if deptID != nil {
		q = q.Where("n.its_departments_id = ?", *deptID)
	}
	var heads []noticeListRow
	if err := q.Select(noticeListSelect).Limit(1).Scan(&heads).Error; err != nil {
		return domain.NoticeCard{}, err
	}
	if len(heads) == 0 {
		return domain.NoticeCard{}, domain.ErrNotFound
	}
	card := domain.NoticeCard{NoticeListItem: heads[0].toDomain()}
	if err := nr.db.WithContext(ctx).Table("its_risk_notice").
		Select(`coalesce(notice_txt,'')`).
		Where("id = ?", id).
		Scan(&card.NoticeTxt).Error; err != nil {
		return domain.NoticeCard{}, err
	}

	var rows []struct {
		ID          int64
		TB515AID    *int64 `gorm:"column:tb_5_15a_id"`
		PpoPp       string
		Paymentdate *time.Time
		IIN         string
		FM          string
		NM          string
		FT          string
		LA1         string
		AmountPart  string
		GU          string
		GUBIN       string `gorm:"column:gu_bin"`
		SenderName  string
	}
	err := nr.db.WithContext(ctx).Table("its_risk_notice_5_15a sn").
		Select(`sn.id, sn.tb_5_15a_id, coalesce(sn.ppo_pp,'') as ppo_pp, sn.paymentdate,
			coalesce(sn.iin,'') as iin, coalesce(sn.fm,'') as fm, coalesce(sn.nm,'') as nm,
			coalesce(sn.ft,'') as ft, coalesce(sn.la1,'') as la1,
			sn.amount_part::text as amount_part, coalesce(sn.gu,'') as gu,
			coalesce(sn.gu_bin,'') as gu_bin, coalesce(sn.sendername,'') as sender_name`).
		Where(`sn.its_risk_notice_id = ? and sn."in$trash" is null`, id).
		Order("sn.id").
		Scan(&rows).Error
	if err != nil {
		return domain.NoticeCard{}, err
	}
	for _, r := range rows {
		card.Rows = append(card.Rows, domain.NoticeRow{
			ID: r.ID, TB515AID: r.TB515AID, PpoPp: r.PpoPp, PaymentDate: r.Paymentdate,
			IIN: r.IIN, FM: r.FM, NM: r.NM, FT: r.FT, LA1: r.LA1,
			AmountPart: r.AmountPart, GU: r.GU, GUBIN: r.GUBIN, SenderName: r.SenderName,
		})
	}
	return card, nil
}

// DeleteNotice — удаление проекта (spec 009 FR-9): только stat «Проект создан», in$trash каскадно.
func (nr *NoticeRepo) DeleteNotice(ctx context.Context, id int64, deptID *int64) error {
	return nr.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Table("its_risk_notice n").
			Select("n.id, n.req_id, rq.stat_id").
			Joins("left join its_req rq on rq.id = n.req_id").
			Where(`n.id = ? and n."in$trash" is null`, id).
			Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "n"}})
		if deptID != nil {
			q = q.Where("n.its_departments_id = ?", *deptID)
		}
		var rows []struct {
			ID     int64
			ReqID  *int64
			StatID *int64
		}
		if err := q.Scan(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return domain.ErrNotFound
		}
		if rows[0].StatID == nil || *rows[0].StatID != domain.ReqStatDraft {
			return domain.ErrNoticeNotEditable
		}
		if err := tx.Exec(`update its_risk_notice set "in$trash" = 1 where id = ?`, id).Error; err != nil {
			return err
		}
		if err := tx.Exec(`update its_risk_notice_5_15a set "in$trash" = 1 where its_risk_notice_id = ?`, id).Error; err != nil {
			return err
		}
		if err := tx.Exec(`update its_risk_notice_recepient set "in$trash" = 1 where its_risk_notice_id = ?`, id).Error; err != nil {
			return err
		}
		if rows[0].ReqID != nil {
			if err := tx.Exec(`update its_req set "in$trash" = 1 where id = ?`, *rows[0].ReqID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
