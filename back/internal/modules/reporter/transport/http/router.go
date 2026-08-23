// Package http — маршруты Reporter Module под /api/v1/reporter.
package http

import "github.com/gofiber/fiber/v2"

// Register монтирует маршруты Reporter Module.
func Register(r fiber.Router, h *Handler, guard *Guard) {
	r.Get("/ping", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"module": "reporter", "status": "ok"})
	})

	// административные (активная сессия + роль администратора)
	admin := r.Group("/admin", guard.RequireAdmin)
	sources := admin.Group("/sources")

	sources.Post("", h.createSource)
	sources.Get("", h.listSources)
	sources.Post("/test", h.testParams) // тест по параметрам без сохранения

	sources.Get("/:id", h.getSource)
	sources.Patch("/:id", h.updateSource)
	sources.Post("/:id/test", h.testSource)
	sources.Post("/:id/activate", h.activate)
	sources.Post("/:id/deactivate", h.deactivate)

	// интроспекция структуры
	sources.Get("/:id/databases", h.databases)
	sources.Get("/:id/databases/:db/tables", h.tables)
	sources.Get("/:id/databases/:db/tables/:table/columns", h.columns)
}
