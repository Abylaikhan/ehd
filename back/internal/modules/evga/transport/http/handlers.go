package http

import (
	"bytes"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"ehd-api/internal/modules/evga/application"
	"ehd-api/internal/modules/evga/domain"
	"ehd-api/pkg/httpserver"
)

// Handler — HTTP-обработчики модуля ЕВГА.
type Handler struct {
	svc      *application.Service
	status   *application.StatusService
	notices  *application.NoticeService
	approval *application.ApprovalService
}

func NewHandler(svc *application.Service, status *application.StatusService, notices *application.NoticeService, approval *application.ApprovalService) *Handler {
	return &Handler{svc: svc, status: status, notices: notices, approval: approval}
}

// mapErr — доменные ошибки → единый контракт ошибок (спеки 007/008).
func mapErr(err error) error {
	var te *domain.TransitionError
	switch {
	case errors.Is(err, domain.ErrInvalidSort):
		return httpserver.NewError(fiber.StatusBadRequest, "INVALID_FILTER", "Недопустимая сортировка",
			httpserver.ErrorDetail{Field: "sort", Reason: "not_in_whitelist"})
	case errors.Is(err, domain.ErrBulkLimit):
		return httpserver.NewError(fiber.StatusBadRequest, "INVALID_FILTER", "Список записей пуст или превышает 1000",
			httpserver.ErrorDetail{Field: "ids", Reason: "empty_or_too_large"})
	case errors.Is(err, domain.ErrNotFound):
		return httpserver.NewError(fiber.StatusNotFound, "NOT_FOUND", "Запись не найдена")
	case errors.Is(err, domain.ErrSourceUnavailable):
		return httpserver.NewError(fiber.StatusServiceUnavailable, "EVGA_SOURCE_UNAVAILABLE", "Источник данных ОБМ ЕВГА недоступен")
	case errors.Is(err, domain.ErrNoticeNotEditable):
		return httpserver.NewError(fiber.StatusUnprocessableEntity, "NOTICE_NOT_EDITABLE",
			"Операция доступна только для уведомления в статусе «Проект создан»")
	case errors.Is(err, domain.ErrAddresseeRequired):
		return httpserver.NewError(fiber.StatusBadRequest, "ADDRESSEE_REQUIRED", err.Error(),
			httpserver.ErrorDetail{Field: "its_cli_id", Reason: "required"})
	case errors.Is(err, domain.ErrNoOwnerDepartment):
		return httpserver.NewError(fiber.StatusForbidden, "ACCESS_DENIED", err.Error())
	case errors.Is(err, application.ErrReadOnlyMode):
		return httpserver.NewError(fiber.StatusForbidden, "EVGA_READ_ONLY", "Модуль ОБМ ЕВГА работает в режиме чтения")
	case errors.Is(err, application.ErrCuratorReadOnly):
		return httpserver.NewError(fiber.StatusForbidden, "ACCESS_DENIED", "Роль не допускает изменения записей")
	case errors.As(err, &te):
		code := "TRANSITION_NOT_ALLOWED"
		if errors.Is(err, domain.ErrTransitionCondition) {
			code = "TRANSITION_CONDITION_FAILED"
		}
		details := []httpserver.ErrorDetail{}
		if te.Field != "" {
			details = append(details, httpserver.ErrorDetail{Field: te.Field, Reason: "required_or_invalid"})
		}
		return httpserver.NewError(fiber.StatusUnprocessableEntity, code, te.Reason, details...)
	default:
		return err
	}
}

// listRegistry — GET /registry (spec FR-4..8).
func (h *Handler) listRegistry(c *fiber.Ctx) error {
	f, err := parseFilter(c)
	if err != nil {
		return err
	}
	page, err := parsePage(c)
	if err != nil {
		return err
	}
	res, scope, err := h.svc.List(c.UserContext(), identityFrom(c), f, page)
	if err != nil {
		return mapErr(err)
	}
	page = page.Normalize()
	items := make([]recordResp, len(res.Items))
	for i, r := range res.Items {
		items[i] = toRecordResp(r)
	}
	return c.JSON(fiber.Map{
		"items":     items,
		"total":     res.Total,
		"page":      page.Page,
		"page_size": page.Size,
		"scope":     toScopeResp(scope),
	})
}

// getRecord — GET /registry/:id (spec FR-9).
func (h *Handler) getRecord(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return httpserver.NewError(fiber.StatusBadRequest, "INVALID_FILTER", "Некорректный идентификатор записи",
			httpserver.ErrorDetail{Field: "id", Reason: "not_an_integer"})
	}
	rec, err := h.svc.Card(c.UserContext(), identityFrom(c), id)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(toCardResp(rec))
}

// exportRegistry — GET /registry/export (spec FR-11).
func (h *Handler) exportRegistry(c *fiber.Ctx) error {
	f, err := parseFilter(c)
	if err != nil {
		return err
	}
	// буфер: ошибка генерации не должна порвать уже отправленный ответ (как в reporter)
	var buf bytes.Buffer
	res, err := h.svc.Export(c.UserContext(), identityFrom(c), f, c.Query("sort"), c.Query("order"), &buf)
	if err != nil {
		return mapErr(err)
	}
	name := "evga-registry-" + time.Now().Format("20060102-150405") + ".xlsx"
	c.Set(fiber.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="`+name+`"`)
	c.Set("X-Evga-Export-Rows", strconv.Itoa(res.Rows))
	if res.Truncated {
		c.Set("X-Evga-Export-Truncated", "true")
	}
	return c.Send(buf.Bytes())
}

// references — GET /references (spec FR-10).
func (h *Handler) references(c *fiber.Ctx) error {
	refs, err := h.svc.References(c.UserContext(), identityFrom(c))
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(toReferencesResp(refs))
}

// --- статусы (спека 008) ---

type statusChangeReq struct {
	StatusID         int64  `json:"status_id"`
	Note             string `json:"note"`
	AmountForVozvrat string `json:"amount_for_vozvrat"`
	Refund           string `json:"refund"`
	ActivityID       *int64 `json:"activity_id"`
}

func (r statusChangeReq) attrs() domain.TransitionAttrs {
	return domain.TransitionAttrs{
		Note:             r.Note,
		AmountForVozvrat: r.AmountForVozvrat,
		Refund:           r.Refund,
		ActivityID:       r.ActivityID,
	}
}

// changeStatus — POST /registry/:id/status (FR-7).
func (h *Handler) changeStatus(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return httpserver.NewError(fiber.StatusBadRequest, "INVALID_FILTER", "Некорректный идентификатор записи",
			httpserver.ErrorDetail{Field: "id", Reason: "not_an_integer"})
	}
	var req statusChangeReq
	if err := c.BodyParser(&req); err != nil {
		return httpserver.NewError(fiber.StatusBadRequest, "INVALID_FILTER", "Некорректное тело запроса",
			httpserver.ErrorDetail{Field: "body", Reason: "invalid_json"})
	}
	if err := h.status.ChangeStatus(c.UserContext(), identityFrom(c), id, req.StatusID, req.attrs()); err != nil {
		return mapErr(err)
	}
	rec, err := h.svc.Card(c.UserContext(), identityFrom(c), id)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(toCardResp(rec))
}

type bulkStatusReq struct {
	IDs []int64 `json:"ids"`
	statusChangeReq
}

// bulkStatus — POST /registry/status/bulk (FR-8).
func (h *Handler) bulkStatus(c *fiber.Ctx) error {
	var req bulkStatusReq
	if err := c.BodyParser(&req); err != nil {
		return httpserver.NewError(fiber.StatusBadRequest, "INVALID_FILTER", "Некорректное тело запроса",
			httpserver.ErrorDetail{Field: "body", Reason: "invalid_json"})
	}
	rep, err := h.status.BulkChangeStatus(c.UserContext(), identityFrom(c), req.IDs, req.StatusID, req.attrs())
	if err != nil {
		return mapErr(err)
	}
	rejections := make([]fiber.Map, len(rep.Rejections))
	for i, r := range rep.Rejections {
		rejections[i] = fiber.Map{"id": r.ID, "reason": r.Reason}
	}
	return c.JSON(fiber.Map{
		"processed":  rep.Processed,
		"rejected":   rep.Rejected,
		"cascaded":   rep.Cascaded,
		"rejections": rejections,
	})
}

// history — GET /registry/:id/history (FR-12); видимость — через Card тем же scope.
func (h *Handler) history(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return httpserver.NewError(fiber.StatusBadRequest, "INVALID_FILTER", "Некорректный идентификатор записи",
			httpserver.ErrorDetail{Field: "id", Reason: "not_an_integer"})
	}
	if _, err := h.svc.Card(c.UserContext(), identityFrom(c), id); err != nil {
		return mapErr(err)
	}
	items, err := h.status.History(c.UserContext(), id)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(fiber.Map{"items": items})
}
