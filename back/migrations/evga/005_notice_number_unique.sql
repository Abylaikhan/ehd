-- EVGA-DB-005 (ТЗ §14.5). Уникальность номера уведомления.
-- По факту схемы (tz/obm-evga-5-15a/14-shema-bd.md): номер хранится в its_req.docnum,
-- формат «11/2026/001097». its_req общий для всех типов документов платформы, поэтому
-- индекс частичный — только по формату номера уведомления и живым записям.
-- Диагностика дублей ПЕРЕД применением (на проде обязательно):
--   select docnum, count(*) from public.its_req
--    where "in$trash" is null and docnum ~ '^[0-9]{2}/[0-9]{4}/[0-9]{6}$'
--    group by 1 having count(*) > 1 limit 20;
create unique index if not exists its_req_notice_docnum_uidx
    on public.its_req (docnum)
 where "in$trash" is null
   and docnum ~ '^[0-9]{2}/[0-9]{4}/[0-9]{6}$';
