# План реализации: Reporter UI — Каталог и таблица

- **Спека**: [spec.md](./spec.md) · **Дата**: 2026-08-24

## Архитектура
Nuxt Layer `reporter`. Серверное состояние — `useAsyncData` + `useApi` (slices 005/006 API). Переиспользуемые состояния — в `base`. Права/капабилити — только из API (Принцип 4).

## Файлы
**shared/api** (`shared/api/types.ts`)
- Добавить: `UserViewListItem`, `ColumnMeta`, `ViewMeta`, `FilterSpec`, `QuerySpec`, `ResultColumn`, `QueryResult`, `CountResult`.

**base** (`layers/base/components/`)
- `EmptyState.vue` — иконка + заголовок + подсказка (props: icon, title, hint).
- `ErrorState.vue` — иконка + сообщение + опц. кнопка «Повторить» (props: title, message, retry?).

**reporter** (`layers/reporter/`)
- `composables/useReporterViews.ts` — `list()`, `meta(slug)`, `query(slug, spec)`, `count(slug, spec)`, `exportView(slug, spec)` (blob + имя из `Content-Disposition`, скачивание).
- `utils/format.ts` — `formatCell(value, displayType)` (дата/датавремя → локаль, null → «—»); `filenameFromDisposition(header)`.
- `utils/format.test.ts` — unit-тесты форматирования и парсинга имени.
- `pages/reporter/index.vue` — каталог (заменить заглушку): `useAsyncData` → `list()`; Empty/Error; ссылки на `/reporter/{slug}`; `middleware: 'auth'`.
- `pages/reporter/[slug].vue` — таблица: `useAsyncData` (meta) + `useAsyncData` (первая страница + count с `watch` на поиск/page_size); load-more через курсор (append); экспорт; состояния; синк поиск/page_size в URL; `middleware: 'auth'`.

## Ключевые решения
- **Keyset load-more**: `useAsyncData` грузит первую страницу + count (реактивно на `search`/`pageSize`); догрузка — ручной append в `extraRows` по `next_cursor`; смена поиска/размера сбрасывает `extraRows` и курсор (watch на data).
- **Экспорт**: `useApi().raw(url,{method:'POST',body,responseType:'blob'})` → `Blob` + `Content-Disposition`; создаём objectURL, кликаем скрытую ссылку, revoke. Только на клиенте, по кнопке.
- **Состояния из ошибок**: `apiErrorCode(error)` → `ACCESS_DENIED`→403-state, `VIEW_NOT_FOUND`/404→not-found, `SOURCE_UNAVAILABLE`→source-state, иначе Error. 401 обрабатывает middleware (редирект на `/login`).
- **URL-state**: `search`, `page_size` в query string через `useRoute`/`router.replace`.
- **Форматирование**: `display_type` date/datetime → `Intl.DateTimeFormat('ru')`; number/money — как есть (расширенные форматы — позже); null/undefined → «—».

## Компоненты PrimeVue
`DataTable`, `Column`, `InputText` (поиск), `Button`, `Card`, `Select` (page_size), `Tag`/`Message` (состояния), `useToast`/`Toast` (экспорт-ошибки), `ProgressSpinner`.

## Тестирование (Принцип 7)
- Unit (vitest, как `layers/auth/utils/validators.test.ts`): `formatCell` по типам + null; `filenameFromDisposition`.
- `pnpm build` (SSR) без ошибок.
- Визуально на стенде (analyst1/Analyst123): каталог → `demo-transactions` → таблица 166 строк (RLS), поиск, «Показать ещё», экспорт XLSX; проверить Loading/Empty/Error/403 (norole1)/404.

## Риски
- SSR-сериализация первой страницы (useAsyncData) vs клиентская догрузка — append держим в отдельном ref, не в useAsyncData.
- Значения ClickHouse (decimal как строка, дата RFC3339) — `formatCell` покрывает; проверка на demo.
- Нет `@nuxt/test-utils` в devDeps — ограничиваемся unit-тестами утилит + build; компонентные тесты вне слайса.
