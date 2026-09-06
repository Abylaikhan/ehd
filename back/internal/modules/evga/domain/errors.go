package domain

import "errors"

var (
	// ErrInvalidSort — поле/направление сортировки вне whitelist (spec FR-7, INVALID_FILTER).
	ErrInvalidSort = errors.New("недопустимое поле или направление сортировки")
	// ErrNotFound — запись не существует или вне видимости пользователя (spec, карточка).
	ErrNotFound = errors.New("запись не найдена")
	// ErrSourceUnavailable — внешняя БД obm_evga недоступна (spec, EVGA_SOURCE_UNAVAILABLE).
	ErrSourceUnavailable = errors.New("источник obm_evga недоступен")
	// ErrBulkLimit — bulk-запрос пуст или превышает предел (спека 008 FR-8, 400).
	ErrBulkLimit = errors.New("список записей пуст или превышает допустимый размер")
)
