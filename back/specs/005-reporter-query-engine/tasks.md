# Задачи: Reporter — Query Engine

- **Спека**: [spec.md](./spec.md) · **План**: [plan.md](./plan.md) · **Дата**: 2026-08-24

| # | Задача | Требование | Файлы | Статус |
|---|---|---|---|---|
| T1 | Domain: QuerySpec/Filter/операторы/AllowedOperators/QueryResult | FR-5..8 | `reporter/domain/query.go` | ☑ |
| T2 | Domain: RowScope + правила применимости | FR-11..14 | `reporter/domain/rowscope.go` | ☑ |
| T3 | Domain: поля keyset/row-scope в DataView и Snapshot | FR-9,FR-11 | `reporter/domain/data_view.go`, `snapshot.go` | ☑ |
| T4 | querybuilder: safeIdent + backtick | Принцип 3 | `reporter/querybuilder/ident.go` | ☑ |
| T5 | querybuilder: операторы → SQL + типизация значений | FR-6,FR-7 | `reporter/querybuilder/operators.go` | ☑ |
| T6 | querybuilder: Build/BuildCount (SELECT, WHERE, RLS, keyset, LIMIT) | FR-4,7,9,11,15 | `reporter/querybuilder/builder.go` | ☑ |
| T7 | Repository: 4 поля в модели + GetViewBySlug | FR-1 | `reporter/repository/models.go`, `data_view_repo.go` | ☑ |
| T8 | ViewService: keyset из PK при create; новые поля в snapshot при publish | FR-9 | `reporter/application/view_service.go` | ☑ |
| T9 | SourceConn.Query/ScalarUint64 + chsource с named-параметрами и query settings | FR-16 | `reporter/application/query_service.go`, `chsource/connector.go` | ☑ |
| T10 | QueryService: резолв slug, RBAC, валидация QuerySpec, RowScope, exec, cursor, count, preview | FR-1..18 | `reporter/application/query_service.go` | ☑ |
| T11 | Transport: guard.RequireAuth; query DTO/handlers; user-маршруты + admin preview | FR-1,2,17,18 | `reporter/transport/http/{guard,query_dto,query_handlers,router}.go` | ☑ |
| T12 | Wiring: собрать QueryService, зарегистрировать | все | `internal/app/app.go` | ☑ |
| T13 | Unit-тесты querybuilder (нет `*`, параметры, операторы, keyset, RLS-варианты, safeIdent) + application (RBAC, page_size) | AC-3..8 | `reporter/querybuilder/*_test.go`, `reporter/application/query_service_test.go` | ☑ |
| T14 | E2E: метаданные; страница+cursor; count; RLS юзер vs админ; NULL исключён; инъекции; preview | AC-1..10 | — | ☑ |
| T15 | DoD: build/vet/gofmt/test; graphify; спека implemented; залив в main | Принцип 7 | — | ☑ |

## Definition of Done
- `go build/vet/gofmt` чисто; `go test ./...` зелёно, включая **негативные security-тесты** (whitelist, операторы, RLS оба режима, скрытые колонки, инъекции).
- E2E на стенде: реальный вывод строк demo-представления с RLS; `total_count` согласован; `/readyz` зелёный.
- Каждое REP-FR трассировано к тесту; graphify обновлён; спека `implemented`; коммит в `main`.
