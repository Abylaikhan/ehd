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

	// уведомления (спека 009); порядок: preview раньше :id
	user.Post("/notices/preview", h.noticePreview)
	user.Post("/notices", h.noticeCreate)
	user.Get("/notices", h.noticeList)
	user.Get("/notices/:id", h.noticeGet)
	user.Delete("/notices/:id", h.noticeDelete)
	user.Get("/cli", h.cliSearch)

	// согласование и маршрут (спека 010)
	user.Get("/notices/:id/route", h.routeGet)
	user.Put("/notices/:id/route", h.routePut)
	user.Get("/notices/:id/participants", h.deptUsers)
	user.Post("/notices/:id/submit", h.noticeSubmit)
	user.Post("/notices/:id/approve", h.noticeApprove)
	user.Post("/notices/:id/reject", h.noticeReject)

	// исходящее и регистрация (спека 011)
	user.Post("/notices/:id/outgoing", h.noticeOutgoing)
	user.Post("/notices/:id/register", h.noticeRegister)
	user.Get("/notices/:id/out", h.noticeOut)
	user.Get("/notices/:id/pdf", h.noticePDF)
}
