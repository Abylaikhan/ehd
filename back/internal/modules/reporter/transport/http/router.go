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

	// представления (Data View)
	views := admin.Group("/views")
	views.Post("", h.createView)
	views.Get("", h.listViews)
	views.Get("/:id", h.getView)
	views.Patch("/:id", h.updateView)
	views.Delete("/:id", h.deleteView)
	views.Put("/:id/columns", h.updateViewColumns)
	views.Post("/:id/columns/refresh", h.refreshViewColumns)
	views.Put("/:id/permissions", h.setViewPermissions)
	views.Post("/:id/publish", h.publishView)
	views.Post("/:id/disable", h.disableView)
	views.Post("/:id/preview", h.previewView) // админ-предпросмотр черновика

	// меню/навигация (админ)
	menu := admin.Group("/menu")
	menu.Get("", h.adminListMenu)
	menu.Post("", h.adminCreateMenu)
	menu.Patch("/:id", h.adminUpdateMenu)
	menu.Delete("/:id", h.adminDeleteMenu)

	// пользовательские представления (активная сессия, RBAC по ролям snapshot)
	user := r.Group("", guard.RequireAuth)
	user.Get("/views", h.listUserViews)
	user.Get("/views/:slug", h.userViewMeta)
	user.Post("/views/:slug/query", h.userQuery)
	user.Post("/views/:slug/count", h.userCount)
	user.Post("/views/:slug/export", h.exportView)
	user.Get("/navigation", h.userNavigation)
}
