// Package http — маршруты Auth Module под /api/v1/auth.
package http

import "github.com/gofiber/fiber/v2"

// Register монтирует маршруты Auth Module.
func Register(r fiber.Router, h *Handler) {
	r.Get("/ping", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"module": "auth", "status": "ok"})
	})

	// публичные
	r.Post("/register", h.register)
	r.Post("/login", h.login)
	r.Post("/eds/challenge", h.edsChallenge)
	r.Post("/eds/verify", h.edsVerify)

	// пользовательские (активная сессия)
	authed := r.Group("", h.requireAuth)
	authed.Get("/me", h.me)
	authed.Post("/logout", h.logout)
	authed.Post("/change-password", h.changePassword)

	// административные
	admin := r.Group("/admin", h.requireAuth, h.requireAdmin)
	admin.Get("/users", h.adminListUsers)
	admin.Get("/users/:id", h.adminGetUser)
	admin.Patch("/users/:id", h.adminUpdateUser)
	admin.Post("/users/:id/unlock", h.adminUnlockUser)
	admin.Post("/users/:id/temp-password", h.adminTempPassword)
	admin.Get("/roles", h.adminListRoles)
	admin.Post("/roles", h.adminCreateRole)
	admin.Get("/regions", h.adminListRegions)
	admin.Get("/departments", h.adminListDepartments)
}
