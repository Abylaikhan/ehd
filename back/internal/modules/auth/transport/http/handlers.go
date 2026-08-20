package http

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"ehd-api/internal/modules/auth/application"
	"ehd-api/internal/modules/auth/domain"
	"ehd-api/pkg/httpserver"
)

// Handler — HTTP-хендлеры Auth Module.
type Handler struct {
	svc          *application.Service
	cookieSecure bool
}

func NewHandler(svc *application.Service, cookieSecure bool) *Handler {
	return &Handler{svc: svc, cookieSecure: cookieSecure}
}

func (h *Handler) setSessionCookie(c *fiber.Ctx, token string, exp time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  exp,
		HTTPOnly: true,
		Secure:   h.cookieSecure,
		SameSite: "Lax",
	})
}

func (h *Handler) clearSessionCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name: sessionCookie, Value: "", Path: "/",
		Expires: time.Unix(0, 0), HTTPOnly: true, Secure: h.cookieSecure, SameSite: "Lax",
	})
}

func badRequest(field, reason string) *httpserver.APIError {
	return httpserver.NewError(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "Некорректные данные",
		httpserver.ErrorDetail{Field: field, Reason: reason})
}

// --- public ---

func (h *Handler) register(c *fiber.Ctx) error {
	var req registerReq
	if err := c.BodyParser(&req); err != nil {
		return badRequest("body", "invalid_json")
	}
	if req.Login == "" {
		return badRequest("login", "required")
	}
	if req.Email == "" {
		return badRequest("email", "required")
	}
	if req.FullName == "" {
		return badRequest("full_name", "required")
	}
	id, err := h.svc.Register(c.UserContext(), application.RegisterInput{
		Login: req.Login, Password: req.Password, IIN: req.IIN,
		FullName: req.FullName, Email: req.Email, Phone: req.Phone,
	})
	if err != nil {
		return h.mapErr(err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"user_id": id, "status": domain.UserStatusPending})
}

func (h *Handler) login(c *fiber.Ctx) error {
	var req loginReq
	if err := c.BodyParser(&req); err != nil {
		return badRequest("body", "invalid_json")
	}
	res, err := h.svc.Login(c.UserContext(), req.Login, req.Password)
	if err != nil {
		return h.mapErr(err)
	}
	h.setSessionCookie(c, res.Token, res.ExpiresAt)
	return c.JSON(loginResp{UserID: res.UserID, Login: res.Login, ExpiresAt: res.ExpiresAt, PasswordChangeRequired: res.PasswordChangeRequired})
}

func (h *Handler) edsChallenge(c *fiber.Ctx) error {
	nonce, err := h.svc.EDSChallenge()
	if err != nil {
		return h.mapErr(err)
	}
	return c.JSON(fiber.Map{"challenge": nonce})
}

func (h *Handler) edsVerify(c *fiber.Ctx) error {
	var req edsVerifyReq
	if err := c.BodyParser(&req); err != nil {
		return badRequest("body", "invalid_json")
	}
	res, err := h.svc.EDSVerify(c.UserContext(), req.Challenge, req.SignedData)
	if err != nil {
		return h.mapErr(err)
	}
	h.setSessionCookie(c, res.Token, res.ExpiresAt)
	return c.JSON(loginResp{UserID: res.UserID, Login: res.Login, ExpiresAt: res.ExpiresAt, PasswordChangeRequired: res.PasswordChangeRequired})
}

// --- user ---

func (h *Handler) changePassword(c *fiber.Ctx) error {
	id, _ := identityFrom(c)
	var req changePasswordReq
	if err := c.BodyParser(&req); err != nil {
		return badRequest("body", "invalid_json")
	}
	if err := h.svc.ChangePassword(c.UserContext(), id.UserID, req.OldPassword, req.NewPassword); err != nil {
		return h.mapErr(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) me(c *fiber.Ctx) error {
	id, _ := identityFrom(c)
	return c.JSON(meResp{
		UserID: id.UserID, Login: id.Login, IsAdmin: id.IsAdmin,
		Roles: emptyIfNil(id.RoleCodes), RegionCodes: emptyIfNil(id.RegionCodes), DepartmentCodes: emptyIfNil(id.DepartmentCodes),
	})
}

func (h *Handler) logout(c *fiber.Ctx) error {
	if token := tokenFromRequest(c); token != "" {
		_ = h.svc.Logout(c.UserContext(), token)
	}
	h.clearSessionCookie(c)
	return c.SendStatus(fiber.StatusNoContent)
}

// --- admin ---

func (h *Handler) adminListUsers(c *fiber.Ctx) error {
	page := atoiDefault(c.Query("page"), 1)
	pageSize := clamp(atoiDefault(c.Query("page_size"), 50), 1, 200)
	views, total, err := h.svc.ListUsers(c.UserContext(), c.Query("status"), c.Query("q"), pageSize, (page-1)*pageSize)
	if err != nil {
		return h.mapErr(err)
	}
	items := make([]userView, 0, len(views))
	for _, v := range views {
		items = append(items, toUserView(v))
	}
	return c.JSON(fiber.Map{"items": items, "total_count": total, "page": page, "page_size": pageSize})
}

func (h *Handler) adminGetUser(c *fiber.Ctx) error {
	v, err := h.svc.GetUser(c.UserContext(), c.Params("id"))
	if err != nil {
		return h.mapErr(err)
	}
	return c.JSON(toUserView(v))
}

func (h *Handler) adminUpdateUser(c *fiber.Ctx) error {
	var req updateUserReq
	if err := c.BodyParser(&req); err != nil {
		return badRequest("body", "invalid_json")
	}
	err := h.svc.UpdateUser(c.UserContext(), c.Params("id"), application.UpdateUserInput{
		IINVerified: req.IINVerified, Status: req.Status,
		RoleIDs: req.RoleIDs, RegionIDs: req.RegionIDs, DepartmentIDs: req.DepartmentIDs,
	})
	if err != nil {
		return h.mapErr(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) adminUnlockUser(c *fiber.Ctx) error {
	if err := h.svc.UnlockUser(c.UserContext(), c.Params("id")); err != nil {
		return h.mapErr(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) adminTempPassword(c *fiber.Ctx) error {
	temp, err := h.svc.SetTempPassword(c.UserContext(), c.Params("id"))
	if err != nil {
		return h.mapErr(err)
	}
	return c.JSON(fiber.Map{"temporary_password": temp})
}

func (h *Handler) adminListRoles(c *fiber.Ctx) error {
	roles, err := h.svc.ListRoles(c.UserContext())
	if err != nil {
		return h.mapErr(err)
	}
	items := make([]roleView, 0, len(roles))
	for _, r := range roles {
		items = append(items, roleView{ID: r.ID, Code: r.Code, NameRu: r.NameRu, NameKk: r.NameKk, Status: r.Status})
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *Handler) adminCreateRole(c *fiber.Ctx) error {
	var req createRoleReq
	if err := c.BodyParser(&req); err != nil {
		return badRequest("body", "invalid_json")
	}
	if req.Code == "" || req.NameRu == "" {
		return badRequest("code", "required")
	}
	role, err := h.svc.CreateRole(c.UserContext(), req.Code, req.NameRu, req.NameKk)
	if err != nil {
		return h.mapErr(err)
	}
	return c.Status(fiber.StatusCreated).JSON(roleView{ID: role.ID, Code: role.Code, NameRu: role.NameRu, NameKk: role.NameKk, Status: role.Status})
}

func (h *Handler) adminListRegions(c *fiber.Ctx) error {
	refs, err := h.svc.ListRegions(c.UserContext())
	if err != nil {
		return h.mapErr(err)
	}
	return c.JSON(fiber.Map{"items": toReferenceViews(refs)})
}

func (h *Handler) adminListDepartments(c *fiber.Ctx) error {
	refs, err := h.svc.ListDepartments(c.UserContext())
	if err != nil {
		return h.mapErr(err)
	}
	return c.JSON(fiber.Map{"items": toReferenceViews(refs)})
}

// --- helpers ---

func (h *Handler) mapErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		return httpserver.NewError(fiber.StatusUnauthorized, "INVALID_CREDENTIALS", "Неверный логин или пароль")
	case errors.Is(err, domain.ErrUserBlocked):
		return httpserver.NewError(fiber.StatusForbidden, "USER_BLOCKED", "Учётная запись заблокирована")
	case errors.Is(err, domain.ErrUserNotActive):
		return httpserver.NewError(fiber.StatusForbidden, "USER_NOT_ACTIVE", "Учётная запись не активна")
	case errors.Is(err, domain.ErrIINTaken):
		return httpserver.NewError(fiber.StatusConflict, "IIN_TAKEN", "ИИН уже зарегистрирован")
	case errors.Is(err, domain.ErrLoginTaken):
		return httpserver.NewError(fiber.StatusConflict, "LOGIN_TAKEN", "Логин уже занят")
	case errors.Is(err, domain.ErrWeakPassword):
		return httpserver.NewError(fiber.StatusUnprocessableEntity, "WEAK_PASSWORD", "Пароль не соответствует политике")
	case errors.Is(err, domain.ErrInvalidIIN):
		return httpserver.NewError(fiber.StatusUnprocessableEntity, "INVALID_IIN", "Некорректный ИИН")
	case errors.Is(err, domain.ErrTempPasswordExpired):
		return httpserver.NewError(fiber.StatusForbidden, "TEMP_PASSWORD_EXPIRED", "Срок временного пароля истёк")
	case errors.Is(err, domain.ErrSessionInvalid):
		return httpserver.NewError(fiber.StatusUnauthorized, "UNAUTHENTICATED", "Сессия недействительна или истекла")
	case errors.Is(err, domain.ErrChallengeInvalid):
		return httpserver.NewError(fiber.StatusBadRequest, "EDS_CHALLENGE_INVALID", "ЭЦП-challenge недействителен или истёк")
	case errors.Is(err, domain.ErrLastAdmin):
		return httpserver.NewError(fiber.StatusConflict, "LAST_ADMIN", "Нельзя заблокировать/разжаловать последнего администратора")
	case errors.Is(err, domain.ErrNotFound):
		return httpserver.NewError(fiber.StatusNotFound, "NOT_FOUND", "Не найдено")
	default:
		return httpserver.NewError(fiber.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
	}
}

func toUserView(v application.AdminUserView) userView {
	return userView{
		ID: v.ID, Login: v.Login, Email: v.Email, FullName: v.FullName, IINMasked: v.IINMasked,
		IINVerified: v.IINVerified, Status: v.Status, FailedAttempts: v.FailedAttempts,
		Roles: emptyIfNil(v.Roles), RegionCodes: emptyIfNil(v.RegionCodes), DepartmentCodes: emptyIfNil(v.DepartmentCodes),
		CreatedAt: v.CreatedAt,
	}
}

func toReferenceViews(refs []domain.Reference) []referenceView {
	items := make([]referenceView, 0, len(refs))
	for _, r := range refs {
		items = append(items, referenceView{ID: r.ID, Code: r.Code, NameRu: r.NameRu, NameKk: r.NameKk, Status: r.Status})
	}
	return items
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
