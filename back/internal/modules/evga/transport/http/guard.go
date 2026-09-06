// Package http — маршруты модуля ОБМ ЕВГА под /api/v1/evga (спека 007).
package http

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"ehd-api/internal/modules/auth/contract"
	"ehd-api/internal/modules/evga/application"
	"ehd-api/pkg/httpserver"
)

// sessionCookie — имя cookie сессии ЕХД (единое с Auth Module).
const sessionCookie = "ehd_session"

// identityKey — ключ доверенной личности в Locals (локальный для evga).
const identityKey = "evga_identity"

// Guard — авторизация маршрутов ЕВГА через auth/contract (без сети).
type Guard struct{ provider contract.Provider }

func NewGuard(p contract.Provider) *Guard { return &Guard{provider: p} }

func tokenFromRequest(c *fiber.Ctx) string {
	if v := c.Cookies(sessionCookie); v != "" {
		return v
	}
	if h := c.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

// RequireModule пропускает пользователей с ролью модуля: evga_auditor, evga_curator или админ.
func (g *Guard) RequireModule(c *fiber.Ctx) error {
	token := tokenFromRequest(c)
	if token == "" {
		return httpserver.NewError(fiber.StatusUnauthorized, "UNAUTHENTICATED", "Требуется аутентификация")
	}
	id, err := g.provider.CurrentUser(c.UserContext(), token)
	if err != nil {
		return httpserver.NewError(fiber.StatusUnauthorized, "UNAUTHENTICATED", "Сессия недействительна или истекла")
	}
	if !application.HasModuleAccess(id) {
		return httpserver.NewError(fiber.StatusForbidden, "ACCESS_DENIED", "Нет доступа к модулю ОБМ ЕВГА")
	}
	c.Locals(identityKey, id)
	return c.Next()
}

// identityFrom — доверенная личность из Locals (после RequireModule).
func identityFrom(c *fiber.Ctx) contract.Identity {
	id, _ := c.Locals(identityKey).(contract.Identity)
	return id
}
