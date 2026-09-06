# Фактическая схема obm_evga — разбор (фаза 0, снято 06.09.2026)

Источник: боевая PostgreSQL 16.1 `obm_evga@10.58.57.39:5432`, схема `public`, **2963 таблицы**
(наш модуль касается ~15). Доступ — только чтение (договорённость; технически учётка шире —
в коде использовать `default_transaction_read_only=on`). Реквизиты — `schema/.env.local` (не в git).

Артефакты: `schema/schema.sql` (полный DDL), `schema/fk_map.txt` (все FK), `schema/rowcounts.txt`,
`schema/refdata.sql` + `schema/sample_*.csv` (ПДн, не в git). Dev-песочница:
`scripts/load_dev_replica.sh` → база `obm_evga` в compose-Postgres (`postgres:5432/obm_evga` изнутри).

## Платформа

Это низкокодовая платформа: у таблиц служебные поля `sys$uuid`, `in$trash` (мягкое удаление),
`created_by/company_id/project_id/staff_position_id`; документооборот — через общий «реестр заявок»
`its_req` с полиморфной привязкой (`entity_id → entities`, `entity_pk`); задачи — общий движок `tasks`.

## Объёмы (прод, 06.09.2026)

| Таблица | Строк | Размер |
|---|---|---|
| `its_tb_5_15a` | 6 558 603 | 9 045 MB |
| `its_risk_notice_5_15a` | 912 284 | 570 MB |
| `its_risk_notice` | 24 421 | 20 MB |
| `its_risk_notice_recepient` | 45 448 | 10 MB |
| `its_out` | 23 051 | 1 896 MB |
| `its_req` | 64 023 | 1 066 MB |
| `its_cli` | 14 325 | 30 MB |
| `users` | 710 | 0.8 MB |
| `files` | 197 405 | 110 MB |

Процесс в проде живой: 24,4 тыс. уведомлений (23 070 «Отправлен объекту аудита»).

## Ключевые ответы на открытые вопросы

### Статус и номер уведомления (OQ-01 ✅)

`its_risk_notice` своих полей статуса/номера **не имеет**. Всё через `req_id → its_req`:
- **статус** = `its_req.stat_id → its_req_stat` (общеплатформенный справочник статусов);
- **номер** = `its_req.docnum`, формат подтверждён: `05/2026/000538`, `16/2026/000721`.

Статусы уведомлений фактически в проде: `1 draft` «Проект создан» (745), `2 approving`
«На согласовании» (567), `3 closed` «Отправлен объекту аудита» (23 070; аналог «Исполнен» из ТЗ),
`15 refused_by_initiator` «Отозван» (38). В справочнике также есть `5 on_signing` «На подписании»
и `12 rework` «Отправлен на доработку» — покрывают сценарий §11.1.

Подписант хранится в шапке: `its_risk_notice.signer_id → users` + `signer_date`.
Прочие поля шапки: `its_organisation_id`, `notice_txt`, `its_departments_id`, `file_id`,
`file_pdf → files`, `its_out_id`, `its_activity_id`, `staff_id`.

### Маршрут и задачи (OQ-02 ✅)

Платформенный движок: **`tasks`** (144 114 строк) + история **`task_hst`** (95 556).
Ключевые поля `tasks`: `entity_id`/`entity_pk` (привязка к сущности), `manager_id` (ответственный),
`type_id`, `status_id`, `step_nn` (порядок этапа), `due_at`, `approve_res_id`, `closed_dt`,
`approve_type_id`, `assigned_by`, `role_id`. `task_hst`: `task_id`, `action_txt`, `txt`, `file_id`.
Есть индексы `idx_tasks_entity_pk` и `(type_id, entity_pk)`.
⚠ Справочники `task_types`/`task_statuses` в БД пустые — семантика `type_id`/`status_id`
задаётся приложением платформы; снять значения по фактическим задачам уведомлений при
проектировании маршрута (фаза 6).

### DDL its_cli / its_out / its_req (OQ-03 ✅ по структуре)

- `its_out`: канцелярские поля `doc_num`, `doc_date`, `subject`, `body`, `sender_cli_id → its_cli`,
  `its_risk_notice_id`, `req_id`, вложения `files jsonb`, `file_id`, подписи **`files_sign text`,
  `second_sign text`, `files_sign_owner`**, `exec_due_time varchar(50)`, номенклатура
  `nomen_id/int_nomen_id`, **ЕНСИ-справочники: `nsi_doc_type_id → its_r_ensi_nsi_doc_type`,
  `nsi_character_id → its_r_ensi_nsi_character`** (это `docKind`/`character` для ЕСЭДО).
- `its_cli`: справочник организаций со своими справочниками (`its_cliref_kind/type`,
  `its_r_cli_stat` и пр.), `file_id`, `acc_manager`. Синхронизация с ЕНСИ — сервис MIND-S-0035.
- `its_req`: `stat_id`, `docnum` (+ `docnum2..5`, `req_num`), `entity_id/entity_pk/entity_uuid`,
  `rejected`, `is_archive`, `planned_final_stat_id`, `reg_scan jsonb`.

### Маппинг пользователей (OQ-11 ✅ реализуем)

`users`: **`iin varchar(250)`** и **`skk_iin varchar(12)`**, `its_departments_id → its_departments`,
`email`, `hr_position_id`. Сопоставление ehd2 ↔ obm_evga по ИИН выполнимо прямым запросом;
какая из двух колонок канонична — проверить по данным (у 710 пользователей).

## Подтверждения ТЗ

- `its_tb_5_15a`: все поля §5/§6 на месте; плюс `its_out_id`, `its_in_id`, `its_risk_id`,
  `is_read`, `exec`, `fl_*` (ГБД ФЛ), `zp_*`. Прямой ссылки «на уведомление» нет —
  связь через снимок `its_risk_notice_5_15a.tb_5_15a_id`.
- Уникальный индекс **`its_tb_5_15a_ppo_pp_idx (ppo_pp, its_risk_profile_id)`** существует (§5).
- `its_risk_notice_5_15a`: оба FK (`tb_5_15a`, `tb_5_15a_id`) существуют — рабочий `tb_5_15a_id`
  (§8.4); **уникального индекса на `tb_5_15a_id` нет** → миграция EVGA-DB-004 актуальна.
- Уникального индекса на `its_req.docnum` нет → EVGA-DB-005 актуальна
  (⚠ `its_req` общий для всех типов документов — индекс делать частичным по типу сущности).
- `its_departments`: `notice_num` есть, **`notice_num_year` нет** → EVGA-DB-001 актуальна;
  реквизиты для подвала письма на месте (`tel_fax`, `post_adress`, `email`, `iik`, `bin`),
  плюс `its_cli_id` (свой адресат ЕСЭДО департамента).
- `its_risk_status`: ровно 12 записей, id/code/title совпадают с §7.1; `in$trash` NULL у всех
  (в т.ч. у выведенных «-») → EVGA-DB-002 актуальна; `title_kk` NULL у всех 12.
- `its_regions`: `code`, `kato`, `region_code` — как в §12.1.

## Замечания для реализации

1. Витрина 6,5 млн строк / 9 ГБ: реестр — только серверная пагинация; фильтры `god`/`mes`,
   `paymentdate`, `its_departments_id` уже частично покрыты FK-индексами, но составные индексы
   под типовые фильтры возможно понадобятся (замер на реальных запросах).
2. Мягкое удаление: везде фильтровать `"in$trash" is null` (записи с `in$trash=1` — корзина).
3. GORM-модели: имена с `$` (`in$trash`, `sys$uuid`) требуют явных `column:"in$trash"` тегов.
4. `its_out` 1,9 ГБ при 23 тыс. строк — тяжёлые `body`/`files_sign`/`second_sign`;
   в списки эти колонки не выбирать.
5. Полный список FK — `schema/fk_map.txt`; в БД связи объявлены, целостность частично держит
   платформа (например, `its_risk_notice.its_out_id` и `its_out.its_risk_notice_id` — взаимные).
