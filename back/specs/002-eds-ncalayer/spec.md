# Спецификация: реальная проверка ЭЦП (NCALayer) — backend

- **ID**: 002-eds-ncalayer
- **Статус**: implemented
- **Требования ТЗ**: REP-FR-001 (вход по ЭЦП через NCALayer), Безопасность (проверка подписи, срока, отзыва, цепочки)
- **Дата**: 2026-08-21

## 1. Цель
Заменить sandbox-заглушку ЭЦП реальной криптографической проверкой CMS-подписи, созданной NCALayer,
против доверенных корней НУЦ РК, с извлечением ИИН/БИН/ФИО.

## 2. Решения (согласовано)
- Крипто: **RSA + ГОСТ** (через `github.com/ddulesov/pkcs7` + `ddulesov/gogost`).
- Отзыв: **OCSP онлайн** (RSA-проверяемо; ГОСТ-ответы OCSP — soft-skip, требуют GOST-провайдера).
- Корни: **тестовая зона pki.gov.kz**, загрузка из `EDS_TRUST_DIR` (не выдуманы, не в репозитории).

## 3. Функциональные требования
| ID | Требование | 
|---|---|
| FR-1 | `Verify(challenge, base64CMS)`: разобрать detached CMS, проверить подпись над challenge |
| FR-2 | Проверить цепочку до доверенного корня НУЦ РК (`EDS_TRUST_DIR`) |
| FR-3 | Проверить срок действия сертификата |
| FR-4 | OCSP-проверка отзыва (online); при `Revoked` — отказ, при недоступности — soft-skip с логом |
| FR-5 | Извлечь ИИН (`SERIALNUMBER`/OID 2.5.4.5), БИН (юрлицо), ФИО (`CN`) |
| FR-6 | Feature-flag `EDS_MODE`: `stub` (sandbox/CI) или `ncalayer` (реальная проверка) |
| FR-7 | В режиме `ncalayer` при пустом trust store сервис НЕ стартует (защита) |

## 4. Архитектура
`eds.Verifier` (интерфейс) → `StubVerifier` | `NCALayerVerifier`. Выбор в `internal/app` по `cfg.EDS.Mode`.
Файлы: `eds/{ncalayer,trust,subject,ocsp,errors}.go`. Ошибка проверки → `domain.ErrEDSVerification` → HTTP 401 `EDS_VERIFICATION_FAILED`.

## 5. Критерии приёмки
- AC-1: `EDS_MODE=ncalayer` + корни в trust → сервис стартует (`eds mode: ncalayer`). ✅
- AC-2: пустой trust → старт запрещён с понятной ошибкой. ✅
- AC-3: невалидная CMS → 401 `EDS_VERIFICATION_FAILED`. ✅
- AC-4: извлечение ИИН/БИН/ФИО из subject — unit-тест на реальном DER. ✅
- AC-5: `EDS_MODE=stub` — прежнее поведение sandbox. ✅

## 6. Ограничения / вне объёма
- Полная e2e-проверка настоящей подписи требует NCALayer + ключ ЭЦП + корни НУЦ РК (нет в sandbox).
  Проверены: разбор сертификата, извлечение subject, trust store, boot-логика, отклонение невалидной CMS.
- ГОСТ-OCSP (проверка подписи OCSP-ответа GOST-издателя) — требует GOST-провайдера; сейчас soft-skip.
