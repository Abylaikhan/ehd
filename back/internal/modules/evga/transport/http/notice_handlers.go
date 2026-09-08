package http

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"ehd-api/internal/modules/evga/domain"
	"ehd-api/pkg/httpserver"
)

// --- уведомления (спека 009) ---

func fmtDatePtr(t *time.Time) *string { return fmtDate(t) }

type previewReq struct {
	IDs []int64 `json:"ids"`
}

type groupRecordResp struct {
	ID          int64   `json:"id"`
	PpoPp       string  `json:"ppo_pp"`
	PaymentDate *string `json:"paymentdate"`
	IIN         string  `json:"iin"`
	FIO         string  `json:"fio"`
	AmountPart  string  `json:"amount_part"`
}

type groupResp struct {
	GU         string            `json:"gu"`
	GUBIN      string            `json:"gu_bin"`
	SenderName string            `json:"sendername"`
	Count      int               `json:"count"`
	TotalSum   string            `json:"total_sum"`
	Records    []groupRecordResp `json:"records"`
}

func toGroupResp(g domain.NoticeGroup) groupResp {
	out := groupResp{GU: g.GU, GUBIN: g.GUBIN, SenderName: g.SenderName, Count: g.Count, TotalSum: g.TotalSum}
	for _, r := range g.Records {
		out.Records = append(out.Records, groupRecordResp{
			ID: r.ID, PpoPp: r.PpoPp, PaymentDate: fmtDatePtr(r.PaymentDate),
			IIN: r.IIN, FIO: r.FIO, AmountPart: r.AmountPart,
		})
	}
	return out
}

type rejectedResp struct {
	ID        int64  `json:"id"`
	Code      string `json:"code"`
	Reason    string `json:"reason"`
	NoticeNum string `json:"notice_num,omitempty"`
}

func toRejectedResp(in []domain.RejectedRecord) []rejectedResp {
	out := make([]rejectedResp, len(in))
	for i, r := range in {
		out[i] = rejectedResp{ID: r.ID, Code: r.Code, Reason: r.Reason, NoticeNum: r.NoticeNum}
	}
	return out
}

// noticePreview — POST /notices/preview (spec 009 FR-1/2).
func (h *Handler) noticePreview(c *fiber.Ctx) error {
	var req previewReq
	if err := c.BodyParser(&req); err != nil {
		return httpserver.NewError(fiber.StatusBadRequest, "INVALID_FILTER", "Некорректное тело запроса",
			httpserver.ErrorDetail{Field: "body", Reason: "invalid_json"})
	}
	groups, rejected, err := h.notices.Preview(c.UserContext(), identityFrom(c), req.IDs)
	if err != nil {
		return mapErr(err)
	}
	groupsResp := make([]groupResp, len(groups))
	for i, g := range groups {
		groupsResp[i] = toGroupResp(g)
	}
	return c.JSON(fiber.Map{"groups": groupsResp, "rejected": toRejectedResp(rejected)})
}

type createNoticesReq struct {
	Groups []struct {
		RecordIDs []int64 `json:"record_ids"`
		CliID     int64   `json:"its_cli_id"`
	} `json:"groups"`
}

// noticeCreate — POST /notices (spec 009 FR-4..6).
func (h *Handler) noticeCreate(c *fiber.Ctx) error {
	var req createNoticesReq
	if err := c.BodyParser(&req); err != nil {
		return httpserver.NewError(fiber.StatusBadRequest, "INVALID_FILTER", "Некорректное тело запроса",
			httpserver.ErrorDetail{Field: "body", Reason: "invalid_json"})
	}
	groups := make([]domain.CreateGroupInput, len(req.Groups))
	for i, g := range req.Groups {
		groups[i] = domain.CreateGroupInput{RecordIDs: g.RecordIDs, CliID: g.CliID}
	}
	res, err := h.notices.Create(c.UserContext(), identityFrom(c), groups)
	if err != nil {
		return mapErr(err)
	}
	created := make([]fiber.Map, len(res.Created))
	for i, n := range res.Created {
		created[i] = fiber.Map{"notice_id": n.NoticeID, "gu": n.GU, "records": n.Records}
	}
	return c.JSON(fiber.Map{"created": created, "rejected": toRejectedResp(res.Rejected)})
}

// cliSearch — GET /cli?q= (spec 009 FR-3).
func (h *Handler) cliSearch(c *fiber.Ctx) error {
	items, err := h.notices.SearchCli(c.UserContext(), c.Query("q"))
	if err != nil {
		return mapErr(err)
	}
	out := make([]fiber.Map, len(items))
	for i, o := range items {
		out[i] = fiber.Map{"id": o.ID, "code": o.Code, "bin_iin": o.BinIIN, "title": o.Title}
	}
	return c.JSON(fiber.Map{"items": out})
}

type noticeItemResp struct {
	ID          int64   `json:"id"`
	Docnum      string  `json:"docnum"`
	StatusID    *int64  `json:"status_id"`
	StatusTitle string  `json:"status_title"`
	GU          string  `json:"gu"`
	SenderName  string  `json:"sendername"`
	Recipient   string  `json:"recipient"`
	RowsCount   int64   `json:"rows_count"`
	TotalSum    string  `json:"total_sum"`
	CreatedAt   *string `json:"created_at"`
	Department  string  `json:"department"`
}

func toNoticeItemResp(n domain.NoticeListItem) noticeItemResp {
	return noticeItemResp{
		ID: n.ID, Docnum: n.Docnum, StatusID: n.StatusID, StatusTitle: n.StatusTitle,
		GU: n.GU, SenderName: n.SenderName, Recipient: n.Recipient,
		RowsCount: n.RowsCount, TotalSum: n.TotalSum, CreatedAt: fmtTS(n.CreatedAt), Department: n.Department,
	}
}

// noticeList — GET /notices (spec 009 FR-7).
func (h *Handler) noticeList(c *fiber.Ctx) error {
	filterDept, err := qInt64(c, "department_id")
	if err != nil {
		return err
	}
	statusID, err := qInt64(c, "status_id")
	if err != nil {
		return err
	}
	page, err := parsePage(c)
	if err != nil {
		return err
	}
	items, total, scope, err := h.notices.List(c.UserContext(), identityFrom(c), filterDept, statusID, c.Query("docnum"), page)
	if err != nil {
		return mapErr(err)
	}
	page = page.Normalize()
	out := make([]noticeItemResp, len(items))
	for i, n := range items {
		out[i] = toNoticeItemResp(n)
	}
	return c.JSON(fiber.Map{
		"items": out, "total": total, "page": page.Page, "page_size": page.Size,
		"scope": toScopeResp(scope),
	})
}

// noticeGet — GET /notices/:id (spec 009 FR-8).
func (h *Handler) noticeGet(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return httpserver.NewError(fiber.StatusBadRequest, "INVALID_FILTER", "Некорректный идентификатор",
			httpserver.ErrorDetail{Field: "id", Reason: "not_an_integer"})
	}
	card, err := h.notices.Get(c.UserContext(), identityFrom(c), id)
	if err != nil {
		return mapErr(err)
	}
	rows := make([]fiber.Map, len(card.Rows))
	for i, r := range card.Rows {
		rows[i] = fiber.Map{
			"id": r.ID, "tb_5_15a_id": r.TB515AID, "ppo_pp": r.PpoPp,
			"paymentdate": fmtDatePtr(r.PaymentDate), "iin": r.IIN,
			"fm": r.FM, "nm": r.NM, "ft": r.FT, "la1": r.LA1,
			"amount_part": r.AmountPart, "gu": r.GU, "gu_bin": r.GUBIN, "sendername": r.SenderName,
		}
	}
	resp := fiber.Map{
		"header": toNoticeItemResp(card.NoticeListItem), "notice_txt": card.NoticeTxt, "rows": rows,
	}
	return c.JSON(resp)
}

// noticeDelete — DELETE /notices/:id (spec 009 FR-9).
func (h *Handler) noticeDelete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return httpserver.NewError(fiber.StatusBadRequest, "INVALID_FILTER", "Некорректный идентификатор",
			httpserver.ErrorDetail{Field: "id", Reason: "not_an_integer"})
	}
	if err := h.notices.Delete(c.UserContext(), identityFrom(c), id); err != nil {
		return mapErr(err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}
