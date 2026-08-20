package http

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"ehd-api/internal/modules/auth/contract"
	"ehd-api/internal/modules/auth/domain"
	"ehd-api/pkg/httpserver"
)

const identityKey = "identity"

// tokenFromRequest — cookie ehd_session или заголовок Authorization: Bearer.
func tokenFromRequest(c *fiber.Ctx) string {
	if v := c.Cookies(sessionCookie); v != "" {
		return v
	}
	if h := c.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

// requireAuth — общий middleware ЕХД: активная сессия и статус пользователя (REP-FR-002).
func (h *Handler) requireAuth(c *fiber.Ctx) error {
	token := tokenFromRequest(c)
	if token == "" {
		return httpserver.NewError(fiber.StatusUnauthorized, "UNAUTHENTICATED", "Требуется аутентификация")
	}
	id, err := h.svc.CurrentUser(c.UserContext(), token)
	if err != nil {
		if errors.Is(err, domain.ErrUserBlocked) {
			return httpserver.NewError(fiber.StatusForbidden, "USER_BLOCKED", "Учётная запись заблокирована")
		}
		return httpserver.NewError(fiber.StatusUnauthorized, "UNAUTHENTICATED", "Сессия недействительна или истекла")
	}
	c.Locals(identityKey, id)
	return c.Next()
}

// requireAdmin — доступ только администратору.
func (h *Handler) requireAdmin(c *fiber.Ctx) error {
	id, ok := identityFrom(c)
	if !ok || !id.IsAdmin {
		return httpserver.NewError(fiber.StatusForbidden, "ACCESS_DENIED", "Недостаточно прав")
	}
	return c.Next()
}

func identityFrom(c *fiber.Ctx) (contract.Identity, bool) {
	id, ok := c.Locals(identityKey).(contract.Identity)
	return id, ok
}
