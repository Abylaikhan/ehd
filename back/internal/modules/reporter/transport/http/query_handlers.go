package http

import (
	"bytes"
	"time"

	"github.com/gofiber/fiber/v2"

	"ehd-api/internal/modules/reporter/export"
	"ehd-api/pkg/httpserver"
)

// --- пользовательские (сессия) ---

func (h *Handler) listUserViews(c *fiber.Ctx) error {
	items, err := h.query.ListForUser(c.UserContext(), requesterFrom(c))
	if err != nil {
		return mapErr(err)
	}
	out := make([]fiber.Map, len(items))
	for i, v := range items {
		out[i] = fiber.Map{"slug": v.Slug, "name": v.Name, "description": v.Description}
	}
	return c.JSON(fiber.Map{"items": out})
}

func (h *Handler) userViewMeta(c *fiber.Ctx) error {
	m, err := h.query.Meta(c.UserContext(), requesterFrom(c), c.Params("slug"))
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(toViewMetaResp(m))
}

func (h *Handler) userQuery(c *fiber.Ctx) error {
	var req querySpecReq
	if err := c.BodyParser(&req); err != nil {
		return badRequest("body", "invalid_json")
	}
	res, err := h.query.Query(c.UserContext(), requesterFrom(c), c.Params("slug"), req.toDomain())
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(toQueryResultResp(res))
}

func (h *Handler) userCount(c *fiber.Ctx) error {
	var req querySpecReq
	if err := c.BodyParser(&req); err != nil {
		return badRequest("body", "invalid_json")
	}
	n, err := h.query.Count(c.UserContext(), requesterFrom(c), c.Params("slug"), req.toDomain())
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(fiber.Map{"total_count": n})
}

func (h *Handler) exportView(c *fiber.Ctx) error {
	var req querySpecReq
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return badRequest("body", "invalid_json")
		}
	}
	res, err := h.query.Export(c.UserContext(), requesterFrom(c), c.Params("slug"), req.toDomain())
	if err != nil {
		return mapErr(err)
	}
	// формируем в буфер, чтобы ошибка генерации не порвала уже отправленный ответ
	var buf bytes.Buffer
	if err := export.WriteXLSX(&buf, "Данные", res.Headers, res.Rows); err != nil {
		return httpserver.NewError(fiber.StatusInternalServerError, "EXPORT_FAILED", "Не удалось сформировать файл")
	}
	filename := res.Filename + "_" + time.Now().UTC().Format("2006-01-02") + ".xlsx"
	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Set("Cache-Control", "no-store")
	return c.Send(buf.Bytes())
}

// --- админ-предпросмотр черновика ---

func (h *Handler) previewView(c *fiber.Ctx) error {
	var req querySpecReq
	if err := c.BodyParser(&req); err != nil {
		return badRequest("body", "invalid_json")
	}
	res, err := h.query.PreviewDraft(c.UserContext(), c.Params("id"), req.toDomain())
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(toQueryResultResp(res))
}
