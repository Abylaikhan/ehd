package http

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"ehd-api/internal/modules/evga/application"
	"ehd-api/internal/modules/evga/domain"
	"ehd-api/pkg/httpserver"
)

// --- разбор query-параметров (spec FR-6/7) ---

func qInt64(c *fiber.Ctx, name string) (*int64, error) {
	s := c.Query(name)
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, badFilter(name, "not_an_integer")
	}
	return &v, nil
}

func qFloat(c *fiber.Ctx, name string) (*float64, error) {
	s := c.Query(name)
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, badFilter(name, "not_a_number")
	}
	return &v, nil
}

func qDate(c *fiber.Ctx, name string) (*time.Time, error) {
	s := c.Query(name)
	if s == "" {
		return nil, nil
	}
	v, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, badFilter(name, "expected_YYYY-MM-DD")
	}
	return &v, nil
}

func qBool(c *fiber.Ctx, name string) (*bool, error) {
	switch c.Query(name) {
	case "":
		return nil, nil
	case "true", "1":
		v := true
		return &v, nil
	case "false", "0":
		v := false
		return &v, nil
	default:
		return nil, badFilter(name, "expected_true_or_false")
	}
}

func badFilter(field, reason string) error {
	return httpserver.NewError(fiber.StatusBadRequest, "INVALID_FILTER", "Некорректный параметр фильтра",
		httpserver.ErrorDetail{Field: field, Reason: reason})
}

// parseFilter собирает domain.Filter из query.
func parseFilter(c *fiber.Ctx) (domain.Filter, error) {
	var f domain.Filter
	var err error
	if f.ProfileID, err = qInt64(c, "profile_id"); err != nil {
		return f, err
	}
	if f.StatusID, err = qInt64(c, "status_id"); err != nil {
		return f, err
	}
	if f.God, err = qInt64(c, "god"); err != nil {
		return f, err
	}
	if f.Mes, err = qInt64(c, "mes"); err != nil {
		return f, err
	}
	if f.PaymentFrom, err = qDate(c, "paymentdate_from"); err != nil {
		return f, err
	}
	if f.PaymentTo, err = qDate(c, "paymentdate_to"); err != nil {
		return f, err
	}
	if f.AmountFrom, err = qFloat(c, "amount_from"); err != nil {
		return f, err
	}
	if f.AmountTo, err = qFloat(c, "amount_to"); err != nil {
		return f, err
	}
	if f.InNotice, err = qBool(c, "in_notice"); err != nil {
		return f, err
	}
	if f.ConfirmedNoDecision, err = qBool(c, "confirmed_no_decision"); err != nil {
		return f, err
	}
	if f.DepartmentID, err = qInt64(c, "department_id"); err != nil {
		return f, err
	}
	f.GU = c.Query("gu")
	f.SenderName = c.Query("sendername")
	f.IIN = c.Query("iin")
	f.NoticeNum = c.Query("notice_num")
	return f, nil
}

func parsePage(c *fiber.Ctx) (domain.Page, error) {
	page := domain.Page{Sort: c.Query("sort"), Order: c.Query("order")}
	if v, err := qInt64(c, "page"); err != nil {
		return page, err
	} else if v != nil {
		page.Page = int(*v)
	}
	if v, err := qInt64(c, "page_size"); err != nil {
		return page, err
	} else if v != nil {
		page.Size = int(*v)
	}
	return page, nil
}

// --- ответы ---

type recordResp struct {
	ID          int64   `json:"id"`
	PpoPp       string  `json:"ppo_pp"`
	PaymentDate *string `json:"paymentdate"`
	IIN         string  `json:"iin"`
	FM          string  `json:"fm"`
	NM          string  `json:"nm"`
	FT          string  `json:"ft"`
	LA1         string  `json:"la1"`
	AmountPart  string  `json:"amount_part"`
	GU          string  `json:"gu"`
	GUBIN       string  `json:"gu_bin"`
	SenderName  string  `json:"sendername"`
	God         *int64  `json:"god"`
	Mes         *int64  `json:"mes"`

	ProfileID    int64  `json:"profile_id"`
	ProfileTitle string `json:"profile_title"`
	StatusID     *int64 `json:"status_id"`
	StatusTitle  string `json:"status_title"`
	StatusNote   string `json:"status_note"`

	DepartmentID    *int64 `json:"department_id"`
	DepartmentTitle string `json:"department_title"`

	NoticeNum string `json:"notice_num"`
	OutNum    string `json:"out_num"`

	// Контрольный срок (EVGA-FR-080). ExecDue — YYYY-MM-DD или пусто;
	// DeadlineState — ""|"expiring"|"expired".
	ExecDue       string `json:"exec_due,omitempty"`
	DeadlineState string `json:"deadline_state,omitempty"`
}

type cardResp struct {
	recordResp
	Refund           string  `json:"refund"`
	AmountForVozvrat string  `json:"amount_for_vozvrat"`
	ActivityID       *int64  `json:"activity_id"`
	IsGBDFL          *int16  `json:"is_gbdfl"`
	FlFM             string  `json:"fl_fm"`
	FlNM             string  `json:"fl_nm"`
	FlFT             string  `json:"fl_ft"`
	CreatedAt        *string `json:"created_at"`
	UpdatedAt        *string `json:"updated_at"`
}

func fmtDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

func fmtTS(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func toRecordResp(r domain.RiskRecord) recordResp {
	return recordResp{
		ID: r.ID, PpoPp: r.PpoPp, PaymentDate: fmtDate(r.PaymentDate),
		IIN: r.IIN, FM: r.FM, NM: r.NM, FT: r.FT, LA1: r.LA1,
		AmountPart: r.AmountPart, GU: r.GU, GUBIN: r.GUBIN, SenderName: r.SenderName,
		God: r.God, Mes: r.Mes,
		ProfileID: r.ProfileID, ProfileTitle: r.ProfileTitle,
		StatusID: r.StatusID, StatusTitle: r.StatusTitle, StatusNote: r.StatusNote,
		DepartmentID: r.DepartmentID, DepartmentTitle: r.DepartmentTitle,
		NoticeNum: r.NoticeNum, OutNum: r.OutNum,
		ExecDue: r.ExecDue, DeadlineState: r.DeadlineState,
	}
}

func toCardResp(r domain.RiskRecord) cardResp {
	return cardResp{
		recordResp:       toRecordResp(r),
		Refund:           r.Refund,
		AmountForVozvrat: r.AmountForVozvrat,
		ActivityID:       r.ActivityID,
		IsGBDFL:          r.IsGBDFL,
		FlFM:             r.FlFM, FlNM: r.FlNM, FlFT: r.FlFT,
		CreatedAt: fmtTS(r.CreatedAt), UpdatedAt: fmtTS(r.UpdatedAt),
	}
}

type scopeResp struct {
	DepartmentID    *int64 `json:"department_id"`
	DepartmentTitle string `json:"department_title,omitempty"`
	AllDepartments  bool   `json:"all_departments"`
	Unmapped        bool   `json:"unmapped"`
}

func toScopeResp(s domain.DepartmentScope) scopeResp {
	return scopeResp{
		DepartmentID:    s.DepartmentID,
		DepartmentTitle: s.DepartmentTitle,
		AllDepartments:  s.AllDepartments,
		Unmapped:        s.Unmapped,
	}
}

type refResp struct {
	ID    int64  `json:"id"`
	Code  string `json:"code,omitempty"`
	Title string `json:"title"`
}

func toRefsResp(in []domain.Reference) []refResp {
	out := make([]refResp, len(in))
	for i, r := range in {
		out[i] = refResp{ID: r.ID, Code: r.Code, Title: r.Title}
	}
	return out
}

func toReferencesResp(r application.References) fiber.Map {
	m := fiber.Map{
		"statuses":   toRefsResp(r.Statuses),
		"profiles":   toRefsResp(r.Profiles),
		"activities": toRefsResp(r.Activities),
	}
	if r.Departments != nil {
		m["departments"] = toRefsResp(r.Departments)
	}
	return m
}
