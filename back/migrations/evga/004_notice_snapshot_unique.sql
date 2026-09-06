-- EVGA-DB-004 (ТЗ §14.4). Одна запись витрины — не более чем в одном уведомлении.
-- ⚠ Зависит от OQ-08: если повторные уведомления допустимы — заменить на частичный индекс.
-- Диагностика дублей ПЕРЕД применением (на проде обязательно):
--   select tb_5_15a_id, count(*) from public.its_risk_notice_5_15a
--    where tb_5_15a_id is not null group by 1 having count(*) > 1 limit 20;
create unique index if not exists its_risk_notice_5_15a_tb_5_15a_id_uidx
    on public.its_risk_notice_5_15a (tb_5_15a_id);
