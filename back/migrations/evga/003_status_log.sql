-- EVGA-DB-003 (ТЗ §14.3). Журнал изменения статусов записей витрины. Идемпотентно.
-- updated_by/updated_at витрины затираются при каждом обновлении и прослеживаемости не дают.
create table if not exists public.its_tb_5_15a_status_log (
    id                  bigserial primary key,
    "sys$uuid"          varchar(36) default public.uuid_generate_v4(),
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
create index if not exists "its_tb_5_15a_status_log$tb_5_15a_id$idx"
    on public.its_tb_5_15a_status_log (tb_5_15a_id);
