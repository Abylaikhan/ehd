// Package http — маршруты Reporter Module под /api/v1/reporter.
package http

import "github.com/gofiber/fiber/v2"

func Register(r fiber.Router) {
	r.Get("/ping", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"module": "reporter", "status": "ok"})
	})
}
