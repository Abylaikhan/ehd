# Задачи: <название фичи> (NNN-<slug>)

Порядок исполнения = порядок в списке. `[P]` — можно параллелить. TDD: тест раньше реализации.

| ID | Задача | Тип | Трассировка (REP-FR/BR) | Затронутые файлы (graphify_impact) | Готово когда |
|---|---|---|---|---|---|
| T001 | Контракт-тест эндпоинта … | test | REP-FR-### | … | тест падает по нужной причине |
| T002 | Доменная сущность … | impl | REP-FR-### | internal/modules/…/domain/… | go build ok |
| T003 | Негативный тест: скрытая колонка не в SELECT | test | REP-BR-007 | … | тест зелёный |
| … | | | | | |

## Порядок по умолчанию
contracts/tests → domain → repository (GORM) → application → transport → wiring (`internal/app`) → integration tests → docs/OpenAPI.

## Обязательные security-задачи
- Негативные: запрещённая колонка/оператор, скрытое поле в ответе/логе/XLSX, оба режима row scope.
