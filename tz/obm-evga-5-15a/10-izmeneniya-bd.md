# §14. Изменения в базе данных `obm_evga`

⚠ Применяются к **внешней боевой БД** — только согласованными SQL-скриптами (не GORM AutoMigrate).

## EVGA-DB-001. Новые поля

```sql
-- Код и БИН ГУ-отправителя в снимке строк уведомления
-- (для колонки "Код ГУ отправителя" приложения к PDF без обращения к живой витрине)
alter table public.its_risk_notice_5_15a
    add column gu varchar(250),
    add column gu_bin varchar(20);

-- Год последней выдачи номера — для ежегодного сброса счётчика
alter table public.its_departments
    add column notice_num_year int4;
```

## EVGA-DB-002. Доработка справочника статусов

```sql
-- Вывод из оборота неиспользуемых статусов
update public.its_risk_status
   set "in$trash" = 1
 where id in (3, 5, 9, 10);
```

~~Дополнительно: заполнить `title_kk` по всем действующим статусам.~~
**07.09.2026 (ответ Б3): казахские тексты НЕ требуются** — казахская часть уведомления статична,
`title_kk` статусов не заполняем.

## EVGA-DB-003. Журнал изменения статусов

`updated_by`/`updated_at` в `its_tb_5_15a` затираются при каждом обновлении — не дают прослеживаемости. Отдельная таблица:

```sql
create table public.its_tb_5_15a_status_log (
    id                  bigserial primary key,
    "sys$uuid"          varchar(36) default uuid_generate_v4(),
    tb_5_15a_id         int8 not null references public.its_tb_5_15a(id),
    status_from_id      int8 references public.its_risk_status(id),
    status_to_id        int8 not null references public.its_risk_status(id),
    note                text,
    amount_for_vozvrat  numeric(20,2),
    refund              numeric(20,2),
    changed_by          int8 references public.users(id),
    changed_at          timestamptz not null default current_timestamp,
    change_source       varchar(50)   -- 'manual' | 'bulk' | 'system'
);
create index "its_tb_5_15a_status_log$tb_5_15a_id$idx"
    on public.its_tb_5_15a_status_log (tb_5_15a_id);
```

Запись — при каждом изменении статуса, включая автоматические.

## EVGA-DB-004. Защита от повторного включения записи в уведомление

```sql
create unique index its_risk_notice_5_15a_tb_5_15a_id_uidx
    on public.its_risk_notice_5_15a (tb_5_15a_id);
```

**07.09.2026 (ответ А1): повторные уведомления НЕ допускаются.** Существующие дубли в проде —
исторические; индекс применять после чистки (перечень дублей — диагностическим SELECT-ом
из скрипта 004). До чистки запрет обеспечивается программной проверкой при формировании:
запись занята, если входит в действующее (не отозванное) уведомление.

## EVGA-DB-005. Уникальность номера уведомления

```sql
create unique index its_risk_notice_number_uidx
    on public.its_risk_notice (<поле номера>)
 where "in$trash" is null;
```

Поле определяется после уточнения структуры `its_risk_notice` (OQ-01).
