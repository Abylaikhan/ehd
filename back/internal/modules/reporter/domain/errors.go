package domain

import "errors"

// Доменные ошибки Reporter — источник данных.
var (
	// ErrSourceAlreadyExists — попытка создать второй источник (MVP: один источник, REP-BR-001).
	ErrSourceAlreadyExists = errors.New("источник уже существует")
	// ErrSourceNotFound — источник по id не найден.
	ErrSourceNotFound = errors.New("источник не найден")
	// ErrConnectionFailed — не удалось подключиться/выполнить SELECT 1 (REP-FR-011).
	ErrConnectionFailed = errors.New("не удалось подключиться к источнику")
	// ErrValidation — некорректные входные данные источника.
	ErrValidation = errors.New("ошибка валидации источника")

	// --- Data View ---

	// ErrViewNotFound — представление по id не найдено.
	ErrViewNotFound = errors.New("представление не найдено")
	// ErrSlugTaken — slug уже используется другим представлением.
	ErrSlugTaken = errors.New("slug уже используется")
	// ErrViewValidation — некорректные данные представления/колонок.
	ErrViewValidation = errors.New("ошибка валидации представления")
	// ErrPublishValidation — представление не готово к публикации (REP-FR-041).
	ErrPublishValidation = errors.New("представление не готово к публикации")
	// ErrTableNotFound — выбранная таблица отсутствует в источнике.
	ErrTableNotFound = errors.New("таблица не найдена в источнике")
	// ErrSourceInactive — источник не активен, представление к нему нельзя создать/опубликовать.
	ErrSourceInactive = errors.New("источник не активен")

	// --- Query Engine ---

	// ErrAccessDenied — у пользователя нет роли, дающей доступ к представлению (REP-FR-053).
	ErrAccessDenied = errors.New("нет доступа к представлению")
	// ErrSourceUnavailable — источник недоступен/деактивирован во время запроса.
	ErrSourceUnavailable = errors.New("источник недоступен")
	// ErrQueryValidation — QuerySpec нарушает whitelist/операторы/типы (Принцип 3).
	ErrQueryValidation = errors.New("некорректный запрос данных")
	// ErrViewNotConfigured — у представления не задан стабильный ключ для keyset-пагинации.
	ErrViewNotConfigured = errors.New("представление не сконфигурировано для запросов")

	// --- Export ---

	// ErrExportBusy — в системе уже выполняется экспорт (одновременно допустим один).
	ErrExportBusy = errors.New("экспорт уже выполняется")
	// ErrExportTooLarge — набор превышает жёсткий лимит выгрузки (100 000 строк).
	ErrExportTooLarge = errors.New("слишком большой набор для экспорта")
)
