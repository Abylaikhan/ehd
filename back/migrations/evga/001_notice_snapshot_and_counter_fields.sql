-- EVGA-DB-001 (ТЗ §14.1). Идемпотентно.
-- Код и БИН ГУ-отправителя в снимке строк уведомления (для PDF без обращения к живой витрине).
alter table public.its_risk_notice_5_15a
    add column if not exists gu     varchar(250),
    add column if not exists gu_bin varchar(20);

-- Год последней выдачи номера — для ежегодного сброса счётчика (ТЗ §12.1 п.3).
alter table public.its_departments
    add column if not exists notice_num_year int4;
