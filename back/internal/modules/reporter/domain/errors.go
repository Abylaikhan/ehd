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
)
