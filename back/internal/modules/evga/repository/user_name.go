package repository

import "fmt"

// ФИО сотрудника obm_evga (спека 022 FR-1).
//
// В боевой БД `users.short_name` заполнен у 1 из 715 пользователей, поэтому везде, где
// мы показываем сотрудника (участники маршрута, инициатор уведомления, подписант и
// исполнитель в PDF, автор перехода в журнале статусов), выводился логин.
// Справочник платформы `users_import` покрывает 473 из 664 сотрудников департаментов
// по связке `usr_user_login = users.login`. Таблица существующая — схема obm_evga не меняется.
//
// Уникальность логина в `users_import` схемой не гарантирована (нет PK/UNIQUE),
// поэтому связка — LATERAL с `limit 1`: дубль логина не размножит строки основного запроса.

// fioJoin — LATERAL-связка справочника ФИО для алиаса таблицы `users`.
// Добавляется в запрос ПОСЛЕ join-а самой `users`: LATERAL видит только предшествующие элементы FROM.
func fioJoin(userAlias, outAlias string) string {
	return fmt.Sprintf(`left join lateral (
		select trim(concat_ws(' ', ui.usr_user_last_name, ui.usr_user_first_name, ui.usr_user_father_name)) as fio
		  from users_import ui
		 where ui.usr_user_login = %s.login
		 limit 1
	) %s on true`, userAlias, outAlias)
}

// fioExpr — отображаемое имя: ФИО → short_name → login.
// Откат гарантирует, что ни для одного пользователя результат не станет хуже прежнего.
func fioExpr(userAlias, outAlias string) string {
	return fmt.Sprintf(`coalesce(nullif(%s.fio, ''), nullif(%s.short_name, ''), %s.login, '')`,
		outAlias, userAlias, userAlias)
}
