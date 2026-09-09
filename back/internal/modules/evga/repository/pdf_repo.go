package repository

import (
	"context"
	"strings"
	"time"

	"ehd-api/internal/modules/evga/domain"
	"ehd-api/internal/modules/evga/pdf"
)

// PDFData собирает данные PDF уведомления (спека 012 FR-3) из obm_evga.
// Реализовано на RegistryRepo — использует то же подключение.
func (rp *RegistryRepo) PDFData(ctx context.Context, noticeID int64, deptID *int64) (pdf.Data, error) {
	var head []struct {
		Docnum      string
		DocDate     *time.Time
		SignerDate  *time.Time
		RecipRU     string
		RecipKZ     string
		RecipCode   string
		DeptTitle   string
		SignerName  string
		SignerPos   string
		SignerPosKZ string
		ExecName    string
		ExecPhone   string
		ExecEmail   string
	}
	q := rp.db.WithContext(ctx).Table("its_risk_notice n").
		Select(`coalesce(rq.docnum,'') as docnum, o.doc_date, n.signer_date,
			coalesce(cli.title,'') as recip_ru, coalesce(cli.title_kk, cli.title, '') as recip_kz,
			coalesce(cli.code,'') as recip_code,
			coalesce(d.title,'') as dept_title,
			coalesce(su.short_name, su.login, '') as signer_name,
			coalesce(sp.title,'') as signer_pos, coalesce(sp.title_kk, sp.title, '') as signer_pos_kz,
			coalesce(cu.short_name, cu.login, '') as exec_name,
			coalesce(cu.office_tel_number,'') as exec_phone, coalesce(cu.email,'') as exec_email`).
		Joins("left join its_req rq on rq.id = n.req_id").
		Joins(`left join its_out o on o.id = n.its_out_id and o."in$trash" is null`).
		Joins(`left join its_risk_notice_recepient rc on rc.its_risk_notice_id = n.id and rc."in$trash" is null`).
		Joins("left join its_cli cli on cli.id = rc.its_cli_id").
		Joins("left join its_departments d on d.id = n.its_departments_id").
		Joins("left join users su on su.id = n.signer_id").
		Joins("left join hr_position sp on sp.id = su.hr_position_id").
		Joins("left join users cu on cu.id = n.created_by").
		Where(`n.id = ? and n."in$trash" is null`, noticeID)
	if deptID != nil {
		q = q.Where("n.its_departments_id = ?", *deptID)
	}
	if err := q.Limit(1).Scan(&head).Error; err != nil {
		return pdf.Data{}, err
	}
	if len(head) == 0 {
		return pdf.Data{}, domain.ErrNotFound
	}
	h := head[0]

	// дата уведомления: регистрация → подписание → сегодня (проект); спека 012 ADR
	docDate := time.Now()
	if h.DocDate != nil {
		docDate = *h.DocDate
	} else if h.SignerDate != nil {
		docDate = *h.SignerDate
	}

	var rows []struct {
		GU          string
		PpoPp       string
		Paymentdate *time.Time
		FM          string
		NM          string
		FT          string
		IIN         string
		LA1         string
		AmountPart  string
	}
	err := rp.db.WithContext(ctx).Table("its_risk_notice_5_15a sn").
		Select(`coalesce(sn.gu,'') as gu, coalesce(sn.ppo_pp,'') as ppo_pp, sn.paymentdate,
			coalesce(sn.fm,'') as fm, coalesce(sn.nm,'') as nm, coalesce(sn.ft,'') as ft,
			coalesce(sn.iin,'') as iin, coalesce(sn.la1,'') as la1,
			coalesce(sn.amount_part::text,'') as amount_part`).
		Where(`sn.its_risk_notice_id = ? and sn."in$trash" is null`, noticeID).
		Order("sn.id").
		Scan(&rows).Error
	if err != nil {
		return pdf.Data{}, err
	}
	out := pdf.Data{
		DocNum: h.Docnum, DocDate: docDate,
		RecipientRU: strings.TrimSpace(h.RecipRU), RecipientKZ: strings.TrimSpace(h.RecipKZ),
		RecipientCode:  strings.TrimSpace(h.RecipCode),
		DeptTitle:      h.DeptTitle,
		SignerPosition: h.SignerPos, SignerPosKZ: h.SignerPosKZ, SignerName: h.SignerName,
		ExecName: h.ExecName, ExecPhone: h.ExecPhone, ExecEmail: h.ExecEmail,
	}
	for _, r := range rows {
		out.Rows = append(out.Rows, pdf.Row{
			GU: r.GU, PpoPp: r.PpoPp, PaymentDate: r.Paymentdate,
			FM: r.FM, NM: r.NM, FT: r.FT, IIN: r.IIN, LA1: r.LA1, AmountPart: r.AmountPart,
		})
	}
	return out, nil
}
