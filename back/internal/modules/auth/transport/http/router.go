// Package http — маршруты Auth Module под /api/v1/auth.
package http

import "github.com/gofiber/fiber/v2"

func Register(r fiber.Router) {
	r.Get("/ping", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"module": "auth", "status": "ok"})
	})
}
