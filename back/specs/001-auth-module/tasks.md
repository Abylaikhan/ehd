# Задачи: Auth Module (001-auth-module)

Порядок = порядок исполнения. `[P]` — параллелизуемые.

| ID | Задача | Тип | Трассировка | Файлы | Готово когда |
|---|---|---|---|---|---|
| T001 | pkg/crypto: AES-GCM + HMAC, unit-тест roundtrip | test+impl | FR-9 | pkg/crypto/crypto.go, crypto_test.go | go test зелёный |
| T002 | domain: политика пароля, валидация ИИН, ошибки, unit-тест | test+impl | FR-5,FR-9 | domain/{password,iin,errors}.go, *_test.go | go test зелёный |
| T003 | config: Auth (ключи, TTL, max attempts, cookie) | impl | FR-6,FR-8 | config/config.go | go build |
| T004 | repository: модели (users+, sessions+token_hash, regions/departments/связки) + AutoMigrate | impl | FR-9,FR-12 | repository/models.go, migrate.go | AutoMigrate ok |
| T005 | repository: repo-реализации (user/role/session/reference) | impl | FR-1,FR-7,FR-8,FR-10 | repository/*_repo.go | go build |
| T006 | eds: Verifier + StubVerifier | impl | FR-3 | eds/verifier.go | go build |
| T007 | application: порты + Service (register/login/logout/change-password/currentUser) | impl | FR-1..FR-9 | application/*.go | go build |
| T008 | application: admin (users list/patch/unlock/temp-password, roles, reference) | impl | FR-10,FR-11 | application/admin.go | go build |
| T009 | application: unit-тесты (lockout, session expiry, temp-pw) | test | FR-7,FR-8,FR-6 | application/service_test.go | go test зелёный |
| T010 | transport: dto + middleware RequireAuth/RequireAdmin | impl | FR-4 | transport/http/{dto,middleware}.go | go build |
| T011 | transport: хендлеры + Register (все маршруты) | impl | все FR | transport/http/{handlers,router}.go | go build |
| T012 | wiring в internal/app + seed справочников | impl | FR-12,FR-13 | internal/app/app.go, repository/seed_reference.go | go build |
| T013 | env: ключи в .env.example и docker-compose | impl | FR-9 | .env.example, docker-compose.dev.yml | стек стартует |
| T014 | OpenAPI: пути /auth | docs | раздел 10 | openapi/openapi.yaml | валидный YAML |
| T015 | Проверка: go build/vet/test, live smoke (register→login→me→change-pw→admin) | verify | AC-1..AC-6 | — | сценарии зелёные |

## Обязательные негативные/security-проверки
- Повторный ИИН → 409; неверный пароль ×3 → блок; не-админ на /auth/admin/* → 403; /auth/me без сессии → 401; ИИН/ФИО/телефон в БД зашифрованы.

## Статус выполнения (2026-08-21)
Все задачи T001–T015 выполнены. `go build/vet/gofmt/test` — зелёно (unit: crypto, domain, application). Живой сценарий — 26/26 проверок (register/dup/weak/admin/RBAC/assign/profile/lockout/unlock/temp-password/EDS/PII at rest). OpenAPI `openapi/openapi.yaml` (14 путей). Реальная валидация ЭЦП-сертификата (NCALayer) — вынесена в Phase 2 (интерфейс `eds.Verifier` + `StubVerifier`).
