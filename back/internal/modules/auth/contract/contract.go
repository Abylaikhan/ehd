// Package contract — внутренний интерфейс Auth Module для других модулей ЕХД.
// По ТЗ между модулями нет сетевых вызовов: Reporter получает пользователя,
// роли и профиль доступа через этот интерфейс внутри одного процесса.
package contract

import "context"

// Identity — доверенный контекст пользователя для авторизации в модулях.
type Identity struct {
	UserID          string
	Login           string
	IsAdmin         bool
	HasPassword     bool // задан ли пароль (у входа по ЭЦП пароля может не быть)
	RoleCodes       []string
	RegionCodes     []string
	DepartmentCodes []string
}

type Provider interface {
	// CurrentUser возвращает пользователя по активной сессии.
	CurrentUser(ctx context.Context, sessionID string) (Identity, error)
	// UserIIN возвращает расшифрованный ИИН пользователя и признак его подтверждённости.
	// ИИН не должен попадать в логи и ответы API — только для внутрипроцессного
	// сопоставления с внешними системами (модуль ЕВГА, спека 007 FR-3).
	UserIIN(ctx context.Context, userID string) (iin string, verified bool, err error)
}
