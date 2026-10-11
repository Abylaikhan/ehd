package http

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"ehd-api/pkg/httpserver"
)

// setUserDeptReq — назначение департамента ДВГА пользователю (спека 016 FR-3).
// department_id = null снимает привязку (возврат к резолву по ИИН).
type setUserDeptReq struct {
	DepartmentID *int64 `json:"department_id"`
}

func userIDParam(c *fiber.Ctx) (string, error) {
	uid := strings.TrimSpace(c.Params("uid"))
	if uid == "" {
		return "", httpserver.NewError(fiber.StatusBadRequest, "INVALID_FILTER", "Не указан пользователь",
			httpserver.ErrorDetail{Field: "uid", Reason: "required"})
	}
	return uid, nil
}

// adminUsersDepartments — POST /admin/user-departments (спека 020 FR-2):
// эффективные департаменты пачкой для колонки в списке пользователей.
func (h *Handler) adminUsersDepartments(c *fiber.Ctx) error {
	var req struct {
		UserIDs []string `json:"user_ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return httpserver.NewError(fiber.StatusBadRequest, "INVALID_FILTER", "Некорректное тело запроса",
			httpserver.ErrorDetail{Field: "body", Reason: "invalid_json"})
	}
	m, err := h.svc.UsersDepartments(c.UserContext(), identityFrom(c), req.UserIDs)
	if err != nil {
		return mapErr(err)
	}
	items := make(map[string]fiber.Map, len(m))
	for uid, d := range m {
		items[uid] = fiber.Map{
			"department_id": d.DepartmentID, "department_title": d.DepartmentTitle, "source": d.Source,
		}
	}
	return c.JSON(fiber.Map{"items": items})
}

// adminGetUserDept — GET /admin/users/:uid/department (спека 016 FR-3).
func (h *Handler) adminGetUserDept(c *fiber.Ctx) error {
	uid, err := userIDParam(c)
	if err != nil {
		return err
	}
	deptID, err := h.svc.UserDepartment(c.UserContext(), identityFrom(c), uid)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(fiber.Map{"department_id": deptID})
}

// adminSetUserDept — PUT /admin/users/:uid/department (спека 016 FR-3/FR-4).
func (h *Handler) adminSetUserDept(c *fiber.Ctx) error {
	uid, err := userIDParam(c)
	if err != nil {
		return err
	}
	var req setUserDeptReq
	if err := c.BodyParser(&req); err != nil {
		return httpserver.NewError(fiber.StatusBadRequest, "INVALID_FILTER", "Некорректное тело запроса",
			httpserver.ErrorDetail{Field: "body", Reason: "invalid_json"})
	}
	if err := h.svc.SetUserDepartment(c.UserContext(), identityFrom(c), uid, req.DepartmentID); err != nil {
		return mapErr(err)
	}
	return c.JSON(fiber.Map{"department_id": req.DepartmentID})
}
