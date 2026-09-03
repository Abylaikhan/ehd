package http

import (
	"github.com/gofiber/fiber/v2"

	"ehd-api/internal/modules/reporter/domain"
)

// --- админ: пункты меню ---

func (h *Handler) adminListMenu(c *fiber.Ctx) error {
	items, err := h.menu.AdminList(c.UserContext())
	if err != nil {
		return mapErr(err)
	}
	out := make([]menuItemResp, len(items))
	for i, it := range items {
		out[i] = toMenuItemResp(it)
	}
	return c.JSON(fiber.Map{"items": out})
}

func (h *Handler) adminCreateMenu(c *fiber.Ctx) error {
	var req menuItemReq
	if err := c.BodyParser(&req); err != nil {
		return badRequest("body", "invalid_json")
	}
	it, err := h.menu.Create(c.UserContext(), req.toInput())
	if err != nil {
		return mapErr(err)
	}
	return c.Status(fiber.StatusCreated).JSON(toMenuItemResp(it))
}

func (h *Handler) adminUpdateMenu(c *fiber.Ctx) error {
	var req menuItemReq
	if err := c.BodyParser(&req); err != nil {
		return badRequest("body", "invalid_json")
	}
	it, err := h.menu.Update(c.UserContext(), c.Params("id"), req.toInput())
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(toMenuItemResp(it))
}

func (h *Handler) adminDeleteMenu(c *fiber.Ctx) error {
	if err := h.menu.Delete(c.UserContext(), c.Params("id")); err != nil {
		return mapErr(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

// --- пользователь: разрешённое дерево навигации ---

func (h *Handler) userNavigation(c *fiber.Ctx) error {
	nodes, err := h.menu.Navigation(c.UserContext(), requesterFrom(c))
	if err != nil {
		return mapErr(err)
	}
	if nodes == nil {
		nodes = []domain.MenuNode{}
	}
	return c.JSON(fiber.Map{"items": nodes})
}
