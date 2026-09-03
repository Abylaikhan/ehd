package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"ehd-api/internal/modules/reporter/application"
	"ehd-api/internal/modules/reporter/domain"
	"ehd-api/pkg/httpserver"
)

// Handler — HTTP-хендлеры Reporter Module (источник, интроспекция, представления, запросы, меню).
type Handler struct {
	svc   *application.Service
	views *application.ViewService
	query *application.QueryService
	menu  *application.MenuService
}

func NewHandler(svc *application.Service, views *application.ViewService, query *application.QueryService, menu *application.MenuService) *Handler {
	return &Handler{svc: svc, views: views, query: query, menu: menu}
}

func badRequest(field, reason string) *httpserver.APIError {
	return httpserver.NewError(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "Некорректные данные",
		httpserver.ErrorDetail{Field: field, Reason: reason})
}

// --- управление источником ---

func (h *Handler) createSource(c *fiber.Ctx) error {
	var req sourceReq
	if err := c.BodyParser(&req); err != nil {
		return badRequest("body", "invalid_json")
	}
	ds, err := h.svc.CreateSource(c.UserContext(), application.CreateSourceInput{
		Code: req.Code, Name: req.Name, Host: req.Host, Port: req.Port,
		Protocol: req.Protocol, TLSEnabled: req.TLSEnabled, TLSSkipVerify: req.TLSSkipVerify,
		Username: req.Username, Password: req.Password,
	})
	if err != nil {
		return mapErr(err)
	}
	return c.Status(fiber.StatusCreated).JSON(toSourceResp(ds))
}

func (h *Handler) listSources(c *fiber.Ctx) error {
	list, err := h.svc.ListSources(c.UserContext())
	if err != nil {
		return mapErr(err)
	}
	items := make([]sourceResp, len(list))
	for i, ds := range list {
		items[i] = toSourceResp(ds)
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *Handler) getSource(c *fiber.Ctx) error {
	ds, err := h.svc.GetSource(c.UserContext(), c.Params("id"))
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(toSourceResp(ds))
}

func (h *Handler) updateSource(c *fiber.Ctx) error {
	var req sourceReq
	if err := c.BodyParser(&req); err != nil {
		return badRequest("body", "invalid_json")
	}
	ds, err := h.svc.UpdateSource(c.UserContext(), c.Params("id"), application.UpdateSourceInput{
		CreateSourceInput: application.CreateSourceInput{
			Code: req.Code, Name: req.Name, Host: req.Host, Port: req.Port,
			Protocol: req.Protocol, TLSEnabled: req.TLSEnabled, TLSSkipVerify: req.TLSSkipVerify,
			Username: req.Username, Password: req.Password,
		},
	})
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(toSourceResp(ds))
}

func (h *Handler) testSource(c *fiber.Ctx) error {
	if err := h.svc.TestSource(c.UserContext(), c.Params("id")); err != nil {
		return mapErr(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *Handler) testParams(c *fiber.Ctx) error {
	var req sourceReq
	if err := c.BodyParser(&req); err != nil {
		return badRequest("body", "invalid_json")
	}
	err := h.svc.TestParams(c.UserContext(), application.ConnParams{
		Host: req.Host, Port: req.Port, Protocol: req.Protocol,
		TLSEnabled: req.TLSEnabled, TLSSkipVerify: req.TLSSkipVerify,
		Username: req.Username, Password: req.Password,
	})
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *Handler) activate(c *fiber.Ctx) error {
	if err := h.svc.Activate(c.UserContext(), c.Params("id")); err != nil {
		return mapErr(err)
	}
	return c.JSON(fiber.Map{"status": domain.SourceStatusActive})
}

func (h *Handler) deactivate(c *fiber.Ctx) error {
	if err := h.svc.Deactivate(c.UserContext(), c.Params("id")); err != nil {
		return mapErr(err)
	}
	return c.JSON(fiber.Map{"status": domain.SourceStatusInactive})
}

// --- интроспекция ---

func (h *Handler) databases(c *fiber.Ctx) error {
	dbs, err := h.svc.Databases(c.UserContext(), c.Params("id"))
	if err != nil {
		return mapErr(err)
	}
	items := make([]databaseResp, len(dbs))
	for i, d := range dbs {
		items[i] = databaseResp{Name: d.Name}
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *Handler) tables(c *fiber.Ctx) error {
	tables, err := h.svc.Tables(c.UserContext(), c.Params("id"), c.Params("db"))
	if err != nil {
		return mapErr(err)
	}
	items := make([]tableResp, len(tables))
	for i, t := range tables {
		items[i] = tableResp{Name: t.Name, Engine: t.Engine, Kind: t.Kind}
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *Handler) columns(c *fiber.Ctx) error {
	cols, err := h.svc.Columns(c.UserContext(), c.Params("id"), c.Params("db"), c.Params("table"))
	if err != nil {
		return mapErr(err)
	}
	items := make([]columnResp, len(cols))
	for i, col := range cols {
		items[i] = columnResp{
			Name: col.Name, Type: col.Type, Position: col.Position, Nullable: col.Nullable,
			Comment: col.Comment, InPrimaryKey: col.InPrimaryKey, InSortingKey: col.InSortingKey,
		}
	}
	return c.JSON(fiber.Map{"items": items})
}

// mapErr переводит доменные ошибки Reporter в единый HTTP-контракт.
func mapErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrSourceAlreadyExists):
		return httpserver.NewError(fiber.StatusConflict, "SOURCE_ALREADY_EXISTS", "Источник уже существует")
	case errors.Is(err, domain.ErrSourceNotFound):
		return httpserver.NewError(fiber.StatusNotFound, "SOURCE_NOT_FOUND", "Источник не найден")
	case errors.Is(err, domain.ErrConnectionFailed):
		return httpserver.NewError(fiber.StatusBadGateway, "SOURCE_CONNECTION_FAILED", "Не удалось подключиться к источнику")
	case errors.Is(err, domain.ErrValidation):
		return httpserver.NewError(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "Некорректные данные источника")
	case errors.Is(err, domain.ErrViewNotFound):
		return httpserver.NewError(fiber.StatusNotFound, "VIEW_NOT_FOUND", "Представление не найдено")
	case errors.Is(err, domain.ErrSlugTaken):
		return httpserver.NewError(fiber.StatusConflict, "SLUG_TAKEN", "Slug уже используется")
	case errors.Is(err, domain.ErrSourceInactive):
		return httpserver.NewError(fiber.StatusConflict, "SOURCE_INACTIVE", "Источник не активен")
	case errors.Is(err, domain.ErrTableNotFound):
		return httpserver.NewError(fiber.StatusUnprocessableEntity, "SOURCE_TABLE_NOT_FOUND", "Таблица не найдена в источнике")
	case errors.Is(err, domain.ErrPublishValidation):
		return httpserver.NewError(fiber.StatusUnprocessableEntity, "PUBLISH_VALIDATION", "Представление не готово к публикации: нужны активный источник, ≥1 видимая колонка, валидный slug и ≥1 роль")
	case errors.Is(err, domain.ErrViewValidation):
		return httpserver.NewError(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "Некорректные данные представления")
	case errors.Is(err, domain.ErrAccessDenied):
		return httpserver.NewError(fiber.StatusForbidden, "ACCESS_DENIED", "Нет доступа к представлению")
	case errors.Is(err, domain.ErrSourceUnavailable):
		return httpserver.NewError(fiber.StatusServiceUnavailable, "SOURCE_UNAVAILABLE", "Источник данных недоступен")
	case errors.Is(err, domain.ErrQueryValidation):
		return httpserver.NewError(fiber.StatusUnprocessableEntity, "QUERY_VALIDATION", "Некорректный запрос данных")
	case errors.Is(err, domain.ErrViewNotConfigured):
		return httpserver.NewError(fiber.StatusUnprocessableEntity, "VIEW_NOT_CONFIGURED", "Представление не сконфигурировано для запросов")
	case errors.Is(err, domain.ErrMenuNotFound):
		return httpserver.NewError(fiber.StatusNotFound, "MENU_NOT_FOUND", "Пункт меню не найден")
	case errors.Is(err, domain.ErrMenuCycle):
		return httpserver.NewError(fiber.StatusUnprocessableEntity, "MENU_CYCLE", "Недопустимая вложенность меню (цикл)")
	case errors.Is(err, domain.ErrMenuDepth):
		return httpserver.NewError(fiber.StatusUnprocessableEntity, "MENU_DEPTH", "Превышена максимальная глубина меню (3)")
	case errors.Is(err, domain.ErrMenuHasChildren):
		return httpserver.NewError(fiber.StatusConflict, "MENU_HAS_CHILDREN", "Сначала удалите вложенные пункты")
	case errors.Is(err, domain.ErrMenuValidation):
		return httpserver.NewError(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "Некорректные данные пункта меню")
	case errors.Is(err, domain.ErrExportBusy):
		return httpserver.NewError(fiber.StatusTooManyRequests, "EXPORT_BUSY", "Экспорт уже выполняется, повторите позже")
	case errors.Is(err, domain.ErrExportTooLarge):
		return httpserver.NewError(fiber.StatusRequestEntityTooLarge, "EXPORT_TOO_LARGE", "Слишком большой набор для экспорта; уточните фильтры")
	default:
		return httpserver.NewError(fiber.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
	}
}
