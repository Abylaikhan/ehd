# Задачи: Auth UI (001-auth-ui)

| ID | Задача | Тип | Трассировка | Файлы | Готово когда |
|---|---|---|---|---|---|
| T001 | Типы API (зеркало OpenAPI) | types | FR-5 | shared/api/types.ts | build ok |
| T002 | useApi: форвард cookie на SSR + credentials | composable | FR-5 | layers/base/composables/useApi.ts | /me работает в SSR |
| T003 | Валидаторы + Vitest | test | FR-1,FR-4 | layers/auth/utils/validators.ts, *.test.ts | vitest зелёный |
| T004 | Session store (Pinia) | store | FR-5,FR-6 | layers/auth/stores/session.ts | build ok |
| T005 | Плагин загрузки /me | plugin | FR-5 | layers/auth/plugins/session.ts | сессия на SSR |
| T006 | Middleware auth/admin/guest | middleware | FR-6 | layers/auth/middleware/*.ts | редиректы работают |
| T007 | useAuth (login/register/changePassword/eds) | composable | FR-1..4 | layers/auth/composables/useAuth.ts | build ok |
| T008 | Страница login (+ ЭЦП dev-диалог) | page | FR-1,FR-2,FR-3 | layers/auth/pages/login.vue | вход работает |
| T009 | Страница register | page | FR-4 | layers/auth/pages/register.vue | pending-сообщение |
| T010 | Страница change-password | page | FR-2 | layers/auth/pages/change-password.vue | смена работает |
| T011 | AppShell: admin-пункты по is_admin + logout | component | FR-6,FR-10 | layers/base/components/AppShell.vue | пункты/выход |
| T012 | Реестр пользователей + карточка (Drawer) | page | FR-7,FR-8 | layers/reporter/pages/reporter/admin/users/index.vue | список/назначение/действия |
| T013 | Роли: список + создание | page | FR-9 | layers/reporter/pages/reporter/admin/roles.vue | создание роли |
| T014 | Проверка: pnpm build, vitest, live smoke, скриншоты 1280 | verify | AC-1..6 | — | зелёно |

## Обязательные состояния/проверки
- Формы: валидация, ошибка, submit-блок. Списки: Loading/Empty/Error/Access denied.
- Guard: не-админ на /reporter/admin/* → редирект/403; logout → /login.

## Статус выполнения (2026-08-21)
Все задачи T001–T014 выполнены. Vitest 7/7 (валидаторы). Живой сценарий через фронт — 8/8:
proxy login/me, **SSR /reporter/admin/users под сессией с пробросом cookie**, guard-редиректы
(неаутентифицированный → /login 302; guest при сессии → / 302), logout. Экраны login/register сняты
на 1280. Ключевое решение — форвард входящего Cookie в `useApi` на SSR. Реальный NCALayer-мост ЭЦП — Phase 2.
