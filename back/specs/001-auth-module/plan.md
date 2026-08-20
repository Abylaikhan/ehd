# План реализации: Auth Module (001-auth-module)

## Техническое решение (по слоям)
- **domain/**: сущности (User, Role, Session, Region, Department, Identity), доменные ошибки, политика пароля, валидация ИИН. Чисто, без инфраструктуры.
- **repository/** (GORM): модели схемы `auth` (+ regions/departments/user_regions/user_departments; sessions с token_hash), реализация портов repo.
- **application/**: `Service` — use cases (register, login, logout, change-password, currentUser, admin users/roles, eds). Определяет порты (интерфейсы repo, Cipher, Verifier, Clock). Реализует `contract.Provider`.
- **eds/**: `Verifier` интерфейс + `StubVerifier` (sandbox); реальный NCALayer — позже.
- **transport/http/** (Fiber): DTO, валидация, хендлеры, session-cookie, middleware `RequireAuth`/`RequireAdmin`.
- **pkg/crypto**: AES-256-GCM для PII + HMAC-SHA256 для ИИН.
- **config**: ключи шифрования/HMAC, TTL сессии/временного пароля, max failed attempts, cookie secure.
- **internal/app**: wiring; seed справочников (регионы/подразделения) для sandbox.

## Затронутый код
- Новое: `pkg/crypto`, `internal/modules/auth/{application,eds}`, `.../transport/http/{dto,middleware,handlers,register}`.
- Изменяется: `config/config.go` (+Auth), `internal/modules/auth/{domain,repository}` (расширение), `internal/app/app.go` (wiring), `docker-compose.dev.yml` + `.env.example` (ключи), `openapi/`.

## Модель данных (дельта к существующей)
- `users`: + `password_change_required bool`, `temp_password_expires_at timestamptz null`.
- `sessions`: + `token_hash varchar(64) uniqueIndex not null` (храним sha256 токена, не сам токен).
- Новые: `regions(id,code,name_ru,name_kk,status)`, `departments(...)`, `user_regions(user_id,region_id)`, `user_departments(user_id,department_id)`.

## Контракты
Session token — случайные 32 байта (hex), в cookie `ehd_session` (HttpOnly/SameSite=Lax/Secure по env) и через `Authorization: Bearer` для API-клиентов. В БД — sha256(token).

## Риски и решения
| Риск | Митигация |
|---|---|
| ЭЦП без NCALayer | `Verifier`-порт + stub; endpoint-контракт финальный |
| Секреты в логах | zap-поля только безопасные; PII/пароли/токены не логируются |
| Fail-open профиля | коды регионов/подразделений только из БД; клиент не влияет |
| Утечка токена из БД | храним sha256, не сам токен |

## Constitution Check
| Принцип | Статус | Комментарий |
|---|---|---|
| 1. Чистая архитектура/границы | PASS | domain чист; GORM в repository; Provider через contract, без сети |
| 2. Фиксированный стек | PASS | Fiber/GORM/zap/viper + stdlib crypto, golang.org/x/crypto/bcrypt |
| 3. Безопасность Query Engine | N/A | модуль не обращается к ClickHouse |
| 4. RBAC/RLS на сервере | PASS | права и коды региона/подразделения только на backend |
| 5. Данные и миграции | PASS | AutoMigrate (sandbox); PII шифруется, ИИН HMAC |
| 6. Наблюдаемость/надёжность | PASS | request_id, единый error-контракт, без секретов |
| 7. Тестирование | PASS | unit: политика пароля, crypto, сессии, lockout; live smoke |
