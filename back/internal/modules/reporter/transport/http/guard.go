package http

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"ehd-api/internal/modules/auth/contract"
	"ehd-api/pkg/httpserver"
)

// sessionCookie — имя cookie сессии ЕХД (единое с Auth Module).
const sessionCookie = "ehd_session"

// identityKey — ключ доверенной личности в Locals (локальный для reporter).
const identityKey = "reporter_identity"

// Guard — авторизация маршрутов Reporter через межмодульный интерфейс Auth (без сети).
type Guard struct{ provider contract.Provider }

func NewGuard(p contract.Provider) *Guard { return &Guard{provider: p} }

// tokenFromRequest достаёт токен сессии из cookie ehd_session или заголовка Bearer.
func tokenFromRequest(c *fiber.Ctx) string {
	if v := c.Cookies(sessionCookie); v != "" {
		return v
	}
	if h := c.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

// RequireAdmin пропускает только администратора с активной сессией (REP-FR-002, RBAC на backend).
func (g *Guard) RequireAdmin(c *fiber.Ctx) error {
	token := tokenFromRequest(c)
	if token == "" {
		return httpserver.NewError(fiber.StatusUnauthorized, "UNAUTHENTICATED", "Требуется аутентификация")
	}
	id, err := g.provider.CurrentUser(c.UserContext(), token)
	if err != nil {
		return httpserver.NewError(fiber.StatusUnauthorized, "UNAUTHENTICATED", "Сессия недействительна или истекла")
	}
	if !id.IsAdmin {
		return httpserver.NewError(fiber.StatusForbidden, "ACCESS_DENIED", "Недостаточно прав")
	}
	c.Locals(identityKey, id)
	return c.Next()
}
