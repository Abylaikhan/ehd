package http

import (
	"github.com/gofiber/fiber/v2"

	"ehd-api/internal/modules/reporter/application"
	"ehd-api/internal/modules/reporter/domain"
)

func (h *Handler) createView(c *fiber.Ctx) error {
	var req createViewReq
	if err := c.BodyParser(&req); err != nil {
		return badRequest("body", "invalid_json")
	}
	v, err := h.views.CreateView(c.UserContext(), application.CreateViewInput{
		Name: req.Name, Slug: req.Slug, Description: req.Description,
		DataSourceID: req.DataSourceID, Database: req.Database, Table: req.Table,
	})
	if err != nil {
		return mapErr(err)
	}
	return c.Status(fiber.StatusCreated).JSON(toViewResp(v))
}

func (h *Handler) listViews(c *fiber.Ctx) error {
	list, err := h.views.ListViews(c.UserContext())
	if err != nil {
		return mapErr(err)
	}
	items := make([]viewResp, len(list))
	for i, v := range list {
		items[i] = toViewResp(v)
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *Handler) getView(c *fiber.Ctx) error {
	d, err := h.views.GetView(c.UserContext(), c.Params("id"))
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(toViewDetailResp(d))
}

func (h *Handler) updateView(c *fiber.Ctx) error {
	var req updateViewReq
	if err := c.BodyParser(&req); err != nil {
		return badRequest("body", "invalid_json")
	}
	v, err := h.views.UpdateView(c.UserContext(), c.Params("id"), application.UpdateViewInput{
		Name: req.Name, Slug: req.Slug, Description: req.Description,
		PageSizeDefault: req.PageSizeDefault, PageSizeMin: req.PageSizeMin, PageSizeMax: req.PageSizeMax,
		DefaultSortColumn: req.DefaultSortColumn, DefaultSortDir: req.DefaultSortDir,
		ExportRowLimit: req.ExportRowLimit, RowScopeMode: req.RowScopeMode,
	})
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(toViewResp(v))
}

func (h *Handler) deleteView(c *fiber.Ctx) error {
	if err := h.views.DeleteView(c.UserContext(), c.Params("id")); err != nil {
		return mapErr(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *Handler) updateViewColumns(c *fiber.Ctx) error {
	var req columnsReq
	if err := c.BodyParser(&req); err != nil {
		return badRequest("body", "invalid_json")
	}
	if err := h.views.UpdateColumns(c.UserContext(), c.Params("id"), req.toInput()); err != nil {
		return mapErr(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *Handler) refreshViewColumns(c *fiber.Ctx) error {
	if err := h.views.RefreshColumns(c.UserContext(), c.Params("id")); err != nil {
		return mapErr(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *Handler) setViewPermissions(c *fiber.Ctx) error {
	var req permissionsReq
	if err := c.BodyParser(&req); err != nil {
		return badRequest("body", "invalid_json")
	}
	if err := h.views.SetPermissions(c.UserContext(), c.Params("id"), req.RoleCodes); err != nil {
		return mapErr(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *Handler) publishView(c *fiber.Ctx) error {
	v, err := h.views.Publish(c.UserContext(), c.Params("id"))
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(toViewResp(v))
}

func (h *Handler) disableView(c *fiber.Ctx) error {
	if err := h.views.Disable(c.UserContext(), c.Params("id")); err != nil {
		return mapErr(err)
	}
	return c.JSON(fiber.Map{"status": domain.ViewStatusDisabled})
}
