package querybuilder

import "errors"

var (
	// ErrUnsafeIdentifier — идентификатор не прошёл проверку (потенциальная инъекция).
	ErrUnsafeIdentifier = errors.New("небезопасный идентификатор")
	// ErrNoColumns — пустой список SELECT-колонок (SELECT * запрещён).
	ErrNoColumns = errors.New("не заданы колонки выборки")
	// ErrBadFilter — некорректный фильтр (оператор/значения).
	ErrBadFilter = errors.New("некорректный фильтр")
)
