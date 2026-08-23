# План реализации: Auth UI (001-auth-ui)

## Техническое решение
- **shared/api/types.ts** — типы-зеркало OpenAPI (Me, LoginResponse, UserView, Role, Reference, запросы, ApiError).
- **layers/base**: `useApi` (форвард cookie на SSR, credentials include); `useApiError` (маппинг единого error-контракта). Расширить AppShell (пункты admin по is_admin, logout).
- **layers/auth**:
  - `stores/session.ts` (Pinia): `user: Me|null`, `isAuthenticated`, `isAdmin`, `fetch()`, `logout()`.
  - `composables/useAuth.ts`: `login/register/changePassword/edsLogin`.
  - `utils/validators.ts`: чистые валидаторы (пароль/ИИН/email) + unit-тесты (Vitest).
  - `pages/`: `login.vue`, `register.vue`, `change-password.vue`.
  - `middleware/`: `auth`, `admin`, `guest`.
  - `plugins/session.ts`: загрузка `/me` при инициализации (SSR-friendly).
- **layers/reporter**:
  - `pages/reporter/admin/users/index.vue`: ServerDataTable, поиск/фильтр, действия, Drawer-карточка (проверка ИИН, роли/регионы/подразделения, разблокировка, временный пароль).
  - `pages/reporter/admin/roles.vue`: список + создание роли.

## Затронутый код
- Новое: `shared/api`, `layers/auth/{composables,utils,middleware,plugins}`, `layers/auth/pages/{register,change-password}.vue`, `layers/reporter/pages/reporter/admin/**`.
- Изменяется: `useApi.ts`, `stores/session.ts`, `pages/login.vue`, `AppShell.vue`, `package.json` (vitest).

## Потоки данных
Страница/composable → `useApi()` → `/api/...` (браузер: nitro-прокси; SSR: apiBase + форвард Cookie) → Auth Module. Сессия в браузере — через HttpOnly cookie `ehd_session` (ставит/чистит backend).

## Риски и решения
| Риск | Митигация |
|---|---|
| SSR не видит cookie | `useRequestHeaders(['cookie'])` в useApi на сервере |
| Фронт «расширяет» права | пункты/действия скрываются по is_admin, но backend авторизует повторно |
| ЭЦП без NCALayer | dev-диалог подписи (sandbox), помечен явно |

## Constitution Check
| Принцип | Статус | Комментарий |
|---|---|---|
| 1. Nuxt Layers | PASS | auth в layers/auth, admin в layers/reporter, общее в base |
| 2. PrimeVue-only + токены | PASS | только PrimeVue, `--ehd-*`, без inline-стилей |
| 3. Сервер-стейт / Pinia scope | PASS | данные через useApi/useAsyncData; в Pinia только сессия |
| 4. Права на бэкенде | PASS | capability из /me; backend повторно авторизует |
| 5. Контракт API | PASS | типы из shared/api (зеркало openapi.yaml) |
| 6. UX и доступность | PASS | Loading/Empty/Error/Access denied; focus/aria; ru |
| 7. Тестирование | PASS | Vitest (валидаторы); pnpm build; live + скриншот |
