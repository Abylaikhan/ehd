package http

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"ehd-api/internal/modules/evga/domain"
	"ehd-api/pkg/httpserver"
)

// mapApprovalErr — ошибки согласования → контракт (спека 010).
func mapApprovalErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrRouteIncomplete):
		return httpserver.NewError(fiber.StatusUnprocessableEntity, "ROUTE_INCOMPLETE", err.Error())
	case errors.Is(err, domain.ErrRouteStateInvalid):
		return httpserver.NewError(fiber.StatusUnprocessableEntity, "ROUTE_STATE_INVALID", err.Error())
	case errors.Is(err, domain.ErrNotAssignee):
		return httpserver.NewError(fiber.StatusForbidden, "ACCESS_DENIED", err.Error())
	case errors.Is(err, domain.ErrCommentRequired):
		return httpserver.NewError(fiber.StatusUnprocessableEntity, "COMMENT_REQUIRED", err.Error(),
			httpserver.ErrorDetail{Field: "comment", Reason: "required"})
	case errors.Is(err, domain.ErrNumberingUnavailable):
		return httpserver.NewError(fiber.StatusUnprocessableEntity, "NUMBERING_UNAVAILABLE", err.Error())
	default:
		return mapErr(err)
	}
}

func noticeIDParam(c *fiber.Ctx) (int64, error) {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return 0, httpserver.NewError(fiber.StatusBadRequest, "INVALID_FILTER", "Некорректный идентификатор",
			httpserver.ErrorDetail{Field: "id", Reason: "not_an_integer"})
	}
	return id, nil
}

type routeStepResp struct {
	ID           int64   `json:"id"`
	Round        int     `json:"round"`
	StepNN       int     `json:"step_nn"`
	Kind         string  `json:"kind"`
	AssigneeID   int64   `json:"assignee_id"`
	AssigneeName string  `json:"assignee_name"`
	Status       string  `json:"status"`
	Result       string  `json:"result"`
	Comment      string  `json:"comment"`
	OpenedAt     *string `json:"opened_at"`
	ClosedAt     *string `json:"closed_at"`
}

func toRouteStepResp(s domain.RouteStep) routeStepResp {
	fmtT := func(t *time.Time) *string {
		if t == nil {
			return nil
		}
		v := t.Format(time.RFC3339)
		return &v
	}
	return routeStepResp{
		ID: s.ID, Round: s.Round, StepNN: s.StepNN, Kind: s.Kind,
		AssigneeID: s.AssigneeObmID, AssigneeName: s.AssigneeName,
		Status: s.Status, Result: s.Result, Comment: s.Comment,
		OpenedAt: fmtT(s.OpenedAt), ClosedAt: fmtT(s.ClosedAt),
	}
}

// routeGet — GET /notices/:id/route (FR-1).
func (h *Handler) routeGet(c *fiber.Ctx) error {
	id, err := noticeIDParam(c)
	if err != nil {
		return err
	}
	view, err := h.approval.Route(c.UserContext(), identityFrom(c), id)
	if err != nil {
		return mapApprovalErr(err)
	}
	tpl := make([]routeStepResp, len(view.Template))
	for i, s := range view.Template {
		tpl[i] = toRouteStepResp(s)
	}
	hist := make([]routeStepResp, len(view.History))
	for i, s := range view.History {
		hist[i] = toRouteStepResp(s)
	}
	return c.JSON(fiber.Map{"editable": view.Editable, "template": tpl, "history": hist})
}

type routePutReq struct {
	ApproverIDs    []int64 `json:"approver_ids"`
	OutgoingUserID int64   `json:"outgoing_user_id"`
}

// routePut — PUT /notices/:id/route (FR-2).
func (h *Handler) routePut(c *fiber.Ctx) error {
	id, err := noticeIDParam(c)
	if err != nil {
		return err
	}
	var req routePutReq
	if err := c.BodyParser(&req); err != nil {
		return httpserver.NewError(fiber.StatusBadRequest, "INVALID_FILTER", "Некорректное тело запроса",
			httpserver.ErrorDetail{Field: "body", Reason: "invalid_json"})
	}
	in := domain.RouteInput{ApproverIDs: req.ApproverIDs, OutgoingUserID: req.OutgoingUserID}
	if err := h.approval.SaveRoute(c.UserContext(), identityFrom(c), id, in); err != nil {
		return mapApprovalErr(err)
	}
	return h.routeGet(c)
}

// noticeSubmit — POST /notices/:id/submit (FR-4).
func (h *Handler) noticeSubmit(c *fiber.Ctx) error {
	id, err := noticeIDParam(c)
	if err != nil {
		return err
	}
	num, err := h.approval.Submit(c.UserContext(), identityFrom(c), id)
	if err != nil {
		return mapApprovalErr(err)
	}
	return c.JSON(fiber.Map{"docnum": num, "status_id": domain.ReqStatApproving})
}

type commentReq struct {
	Comment string `json:"comment"`
}

// noticeApprove — POST /notices/:id/approve (FR-5).
func (h *Handler) noticeApprove(c *fiber.Ctx) error {
	id, err := noticeIDParam(c)
	if err != nil {
		return err
	}
	var req commentReq
	_ = c.BodyParser(&req)
	if err := h.approval.Approve(c.UserContext(), identityFrom(c), id, req.Comment); err != nil {
		return mapApprovalErr(err)
	}
	return c.JSON(fiber.Map{"approved": true})
}

// noticeReject — POST /notices/:id/reject (FR-6).
func (h *Handler) noticeReject(c *fiber.Ctx) error {
	id, err := noticeIDParam(c)
	if err != nil {
		return err
	}
	var req commentReq
	_ = c.BodyParser(&req)
	if err := h.approval.Reject(c.UserContext(), identityFrom(c), id, req.Comment); err != nil {
		return mapApprovalErr(err)
	}
	return c.JSON(fiber.Map{"rejected": true})
}

// deptUsers — GET /notices/:id/participants?q= (FR-3).
func (h *Handler) deptUsers(c *fiber.Ctx) error {
	id, err := noticeIDParam(c)
	if err != nil {
		return err
	}
	items, err := h.approval.DeptUsersSearch(c.UserContext(), identityFrom(c), id, c.Query("q"))
	if err != nil {
		return mapApprovalErr(err)
	}
	out := make([]fiber.Map, len(items))
	for i, u := range items {
		out[i] = fiber.Map{"id": u.ID, "name": u.Name, "login": u.Login}
	}
	return c.JSON(fiber.Map{"items": out})
}
