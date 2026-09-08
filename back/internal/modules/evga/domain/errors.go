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
	// ErrNoticeNotEditable — операция допустима только для статуса «Проект создан» (спека 009 FR-9).
	ErrNoticeNotEditable = errors.New("уведомление недоступно для изменения в текущем статусе")
	// ErrAddresseeRequired — у группы не выбран адресат (спека 009 FR-4, EVGA-FR-034).
	ErrAddresseeRequired = errors.New("у каждой группы должен быть выбран адресат из справочника ЕСЭДО")
	// ErrNoOwnerDepartment — создающий пользователь не привязан к департаменту ДВГА
	// (уведомление обязано принадлежать департаменту, EVGA-FR-037).
	ErrNoOwnerDepartment = errors.New("создание уведомлений доступно аудитору, сопоставленному с департаментом ДВГА")
)
