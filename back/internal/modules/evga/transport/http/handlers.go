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
type Handler struct{ svc *application.Service }

func NewHandler(svc *application.Service) *Handler { return &Handler{svc: svc} }

// mapErr — доменные ошибки → единый контракт ошибок (spec, раздел 4).
func mapErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidSort):
		return httpserver.NewError(fiber.StatusBadRequest, "INVALID_FILTER", "Недопустимая сортировка",
			httpserver.ErrorDetail{Field: "sort", Reason: "not_in_whitelist"})
	case errors.Is(err, domain.ErrNotFound):
		return httpserver.NewError(fiber.StatusNotFound, "NOT_FOUND", "Запись не найдена")
	case errors.Is(err, domain.ErrSourceUnavailable):
		return httpserver.NewError(fiber.StatusServiceUnavailable, "EVGA_SOURCE_UNAVAILABLE", "Источник данных ОБМ ЕВГА недоступен")
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
