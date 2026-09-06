package http

import "github.com/gofiber/fiber/v2"

// Register монтирует маршруты модуля ОБМ ЕВГА (все — под RequireModule).
// Порядок важен: /registry/export раньше /registry/:id.
func Register(r fiber.Router, h *Handler, guard *Guard) {
	r.Get("/ping", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"module": "evga", "status": "ok"})
	})

	user := r.Group("", guard.RequireModule)
	user.Get("/registry", h.listRegistry)
	user.Get("/registry/export", h.exportRegistry)
	user.Post("/registry/status/bulk", h.bulkStatus)
	user.Get("/registry/:id", h.getRecord)
	user.Post("/registry/:id/status", h.changeStatus)
	user.Get("/registry/:id/history", h.history)
	user.Get("/references", h.references)
}
