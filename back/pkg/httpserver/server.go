package httpserver

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"ehd-api/config"
)

const requestIDKey = "request_id"

// New собирает fiber.App с общими middleware и единым обработчиком ошибок.
func New(cfg *config.Config, log *zap.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:               cfg.App.Name,
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          150 * time.Second, // запас под потоковый XLSX-экспорт (до 120 c по ТЗ)
		IdleTimeout:           60 * time.Second,
		DisableStartupMessage: true,
		ErrorHandler:          errorHandler(log),
	})

	// request_id для каждого запроса (требование ТЗ: все ответы содержат request_id)
	app.Use(requestid.New(requestid.Config{
		Header:     "X-Request-Id",
		ContextKey: requestIDKey,
		Generator:  func() string { return "req_" + uuid.NewString() },
	}))
	app.Use(requestLogger(log))
	app.Use(recover.New(recover.Config{EnableStackTrace: true}))

	return app
}

func requestID(c *fiber.Ctx) string {
	if v, ok := c.Locals(requestIDKey).(string); ok {
		return v
	}
	return ""
}

func requestLogger(log *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		// ErrorHandler выставляет статус ПОЗЖЕ этого middleware, поэтому при ошибке
		// берём фактический статус из возвращённой ошибки, а не из ещё не записанного ответа.
		status := c.Response().StatusCode()
		if err != nil {
			var apiErr *APIError
			if errors.As(err, &apiErr) {
				status = apiErr.Status
			} else {
				status = fiber.StatusInternalServerError
			}
		}
		log.Info("http",
			zap.String("request_id", requestID(c)),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", status),
			zap.Int64("duration_ms", time.Since(start).Milliseconds()),
		)
		return err
	}
}

func errorHandler(log *zap.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		reqID := requestID(c)

		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			var fe *fiber.Error
			if errors.As(err, &fe) {
				apiErr = NewError(fe.Code, codeForStatus(fe.Code), messageForStatus(fe.Code))
			} else {
				log.Error("unhandled error", zap.String("request_id", reqID), zap.Error(err))
				apiErr = NewError(fiber.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
			}
		}

		return c.Status(apiErr.Status).JSON(errorResponse{
			RequestID: reqID,
			Error:     errorBody{Code: apiErr.Code, Message: apiErr.Message, Details: apiErr.Details},
		})
	}
}

func codeForStatus(status int) string {
	switch status {
	case fiber.StatusNotFound:
		return "NOT_FOUND"
	case fiber.StatusMethodNotAllowed:
		return "METHOD_NOT_ALLOWED"
	default:
		return "HTTP_ERROR"
	}
}

func messageForStatus(status int) string {
	switch status {
	case fiber.StatusNotFound:
		return "Ресурс не найден"
	case fiber.StatusMethodNotAllowed:
		return "Метод не поддерживается"
	default:
		return "Ошибка запроса"
	}
}
