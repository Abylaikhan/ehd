# Спецификация: вход по ЭЦП через NCALayer — frontend

- **ID**: 002-eds-ncalayer
- **Статус**: implemented
- **Требования ТЗ**: REP-FR-001 (вход по ЭЦП через NCALayer)
- **Дата**: 2026-08-21
- **Зависит от**: back/specs/002-eds-ncalayer (эндпоинты `/auth/eds/*`)

## 1. Цель
Подключить реальный клиент NCALayer: подписать серверный challenge и отправить CMS на проверку бэкенду.

## 2. Функциональные требования
| ID | Требование |
|---|---|
| FR-1 | `useNcaLayer().signChallenge(base64)` — connect к `wss://127.0.0.1:13579`, `basicsSignCMS` → base64 CMS |
| FR-2 | Экран входа: режим `edsMode` (`stub`|`ncalayer`) из публичного runtime-config (`NUXT_PUBLIC_EDS_MODE`) |
| FR-3 | `ncalayer`: кнопка «Подписать через NCALayer» → challenge → подпись → `/eds/verify` |
| FR-4 | `stub`: ручной ввод (ИИН обязателен) — прежнее sandbox-поведение |
| FR-5 | Понятные ошибки: NCALayer не запущен → сообщение + ссылка на установку; отмена; отказ проверки |
| FR-6 | `ncalayer-js-client` импортируется только в браузере (не в SSR-бандл) |

## 3. Реализация
- `layers/auth/composables/useNcaLayer.ts` — обёртка + `ncaErrorMessage`.
- `layers/auth/pages/login.vue` — диалог ЭЦП, ветка по `edsMode`.
- `nuxt.config`: `runtimeConfig.public.edsMode`. Синхронизация с бэкендом через общий `EDS_MODE` (compose).

## 4. Критерии приёмки
- AC-1: `NUXT_PUBLIC_EDS_MODE=ncalayer` → диалог показывает поток NCALayer (не ручную форму). ✅ (build + runtime)
- AC-2: NCALayer не запущен → понятное сообщение со ссылкой на установку.
- AC-3: `stub` — прежний ручной ввод работает end-to-end. ✅ (проверено в браузере)
- AC-4: `pnpm build` без ошибок; библиотека не попадает в SSR. ✅

## 5. Ограничения
- Реальная подпись требует установленного NCALayer + ключа ЭЦП — недоступно в sandbox/CI.
  Проверено: сборка, установка клиента, режимная логика, рантайм-флаг.
