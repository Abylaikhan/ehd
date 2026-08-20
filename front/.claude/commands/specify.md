---
description: Создать спецификацию экрана/фичи фронтенда (spec.md) по описанию
argument-hint: <короткое описание фичи>
---

Ты пишешь **спецификацию** (не реализацию) для фронтенда ЕХД.

Фича: $ARGUMENTS

Порядок:

1. Изучи контекст только через инструменты (Grep/Glob запрещены):
   - `graphify_freshness`; при устаревании — `graphify_build update=true`.
   - `graphify_locate` / `graphify_query` — существующие компоненты, composables, layers, маршруты.
   - Требования: `../.tz_extracted.txt` (`REP-FR/BR`, Приложение D — реестр экранов и компонентов PrimeVue).
2. Определи `NNN` (просмотри `specs/` через Read) и создай `specs/NNN-<slug>/spec.md` по `.specify/templates/spec-template.md`.
3. Заполни: цель, пользовательские сценарии, маршрут(ы), UX-поведение, требуемые состояния (Loading/Empty/Error/Access denied/…), права/capability из API, данные (поля из контракта, не выдуманные), критерии приёмки.
4. Запрещено в спеке: детали реализации (имена компонентов, CSS, выбор библиотек). Только «что» и «зачем».
5. Неоднозначности — `[NEEDS CLARIFICATION: …]`.

Соответствие `.specify/memory/constitution.md` обязательно. Утверждения о коде подтверждай Graphify/Read.
