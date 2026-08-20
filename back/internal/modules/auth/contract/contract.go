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
}
