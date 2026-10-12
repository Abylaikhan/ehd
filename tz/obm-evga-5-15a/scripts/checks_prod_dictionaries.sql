-- Проверка боевой obm_evga: справочники для автоадресатов (замечание 3) и аудит схемы.
-- Выполнено 13.10.2026, результаты — в voprosy-analitiku-6.md.
--
-- ТОЛЬКО SELECT. Запускать с принудительным read-only, не полагаясь на сервер:
--   на боевой БД default_transaction_read_only = OFF.
--   PGOPTIONS="-c default_transaction_read_only=on" psql -h 10.58.57.39 -U <ro> -d obm_evga -f <этот файл>
-- Роль abdd имеет только SELECT на таблицах модуля — проверено запросом 6.

\pset pager off

\echo ==== 1. Применены ли миграции EVGA-DB-001/003/004 ====
select 'its_risk_notice_5_15a.gu' as obj,
       exists(select 1 from information_schema.columns
               where table_schema='public' and table_name='its_risk_notice_5_15a' and column_name='gu') as est
union all select 'its_risk_notice_5_15a.gu_bin',
       exists(select 1 from information_schema.columns
               where table_schema='public' and table_name='its_risk_notice_5_15a' and column_name='gu_bin')
union all select 'its_departments.notice_num_year',
       exists(select 1 from information_schema.columns
               where table_schema='public' and table_name='its_departments' and column_name='notice_num_year')
union all select 'its_tb_5_15a_status_log (таблица)',
       to_regclass('public.its_tb_5_15a_status_log') is not null
union all select 'users_import (таблица)',
       to_regclass('public.users_import') is not null;
-- Факт 13.10.2026: gu, gu_bin, notice_num_year, its_tb_5_15a_status_log — ОТСУТСТВУЮТ.

\echo ==== 2. NOT NULL без DEFAULT в таблицах записи (риск падения INSERT) ====
select table_name, column_name, data_type
  from information_schema.columns
 where table_schema='public'
   and table_name in ('its_req','its_risk_notice','its_risk_notice_5_15a',
                      'its_risk_notice_recepient','its_out')
   and is_nullable='NO' and column_default is null
 order by 1, ordinal_position;
-- Факт: 0 строк. Других сюрпризов уровня gu/gu_bin нет.

\echo ==== 3. Триггеры на таблицах записи ====
select t.tgrelid::regclass as tbl, t.tgname, p.proname as func
  from pg_trigger t join pg_proc p on p.oid = t.tgfoid
 where not t.tgisinternal
   and t.tgrelid::regclass::text in ('its_req','its_risk_notice','its_risk_notice_5_15a',
                                     'its_risk_notice_recepient','its_out','its_tb_5_15a')
 order by 1, 2;
-- Факт: только updated_at и каскад при удалении its_req. Поведение записи не меняют.

\echo ==== 4. Пригодность справочника адресатов its_cli для связки по коду/БИН ====
select count(*) as vsego,
       count(*) filter (where coalesce(bin_iin,'') <> '') as s_bin_iin,
       count(*) filter (where coalesce(code,'')    <> '') as s_code,
       count(*) filter (where coalesce(kod,'')     <> '') as s_kod,
       count(*) filter (where coalesce(short_title,'') <> '') as s_short_title
  from its_cli;
-- Факт: 14325 / 18 / 0 / 0 / 0 — связать можно ТОЛЬКО по title.

\echo ==== 5. Справочник подчинённости: заполненность уровней ====
select count(*) as strok,
       count(*) filter (where coalesce(bin_gu1,'')<>'') as l1,
       count(*) filter (where coalesce(bin_gu2,'')<>'') as l2,
       count(*) filter (where coalesce(bin_gu3,'')<>'') as l3,
       count(*) filter (where coalesce(bin_gu4,'')<>'') as l4,
       count(*) filter (where coalesce(bin_gu5,'')<>'') as l5,
       count(*) filter (where coalesce(bin_gu6,'')<>'') as l6
  from spr_hierarchy_reference;
-- Факт: 25296 / 25296 / 24259 / 20173 / 8355 / 257 / 65.

\echo ==== 6. Покрытие: БИН ГУ витрины найден в иерархии ====
with gu as (select distinct gu_bin from its_tb_5_15a where coalesce(gu_bin,'') <> '')
select (select count(*) from gu) as gu_bin_vsego,
       count(*) filter (where found) as naydeno
  from (select exists(select 1 from spr_hierarchy_reference s
                       where g.gu_bin in (s.bin_gu1,s.bin_gu2,s.bin_gu3,s.bin_gu4,s.bin_gu5,s.bin_gu6)) as found
          from gu g) z;
-- Факт: 5859 из 6145 = 95.3%.

\echo ==== 7. Фактический выбор адресатов: сравнение с иерархией ====
-- Отвечает на вопрос 2 документа №6: что аудиторы понимают под «вышестоящей».
with chosen as (
  select t.gu_bin, max(t.sendername) as gu_name, c.title as cli_title, count(*) as raz
    from its_risk_notice n
    join its_risk_notice_5_15a sn on sn.its_risk_notice_id = n.id and sn."in$trash" is null
    join its_tb_5_15a t on t.id = sn.tb_5_15a_id
    join its_risk_notice_recepient r on r.its_risk_notice_id = n.id and r."in$trash" is null
    join its_cli c on c.id = r.its_cli_id
   where n."in$trash" is null and coalesce(t.gu_bin,'') <> '' and coalesce(c.title,'') <> ''
   group by t.gu_bin, c.title
)
select ch.gu_name as gu_vitriny, p.self_name as gu_po_ierarhii,
       p.parent_name as vyshestoyashchaya_po_spravochniku,
       ch.cli_title as fakticheskiy_adresat, ch.raz
  from chosen ch
  left join lateral (
    select case when s.bin_gu6=ch.gu_bin then s.name_gu5 when s.bin_gu5=ch.gu_bin then s.name_gu4
                when s.bin_gu4=ch.gu_bin then s.name_gu3 when s.bin_gu3=ch.gu_bin then s.name_gu2
                when s.bin_gu2=ch.gu_bin then s.name_gu1 end as parent_name,
           case when s.bin_gu6=ch.gu_bin then s.name_gu6 when s.bin_gu5=ch.gu_bin then s.name_gu5
                when s.bin_gu4=ch.gu_bin then s.name_gu4 when s.bin_gu3=ch.gu_bin then s.name_gu3
                when s.bin_gu2=ch.gu_bin then s.name_gu2 when s.bin_gu1=ch.gu_bin then s.name_gu1 end as self_name
      from spr_hierarchy_reference s
     where ch.gu_bin in (s.bin_gu1,s.bin_gu2,s.bin_gu3,s.bin_gu4,s.bin_gu5,s.bin_gu6)
     limit 1) p on true
 where p.parent_name is not null
 order by ch.raz desc
 limit 10;
-- Факт: у отдела образования Сайрамского района (11780 уведомлений) иерархия даёт
-- «Аппарат акима района», а аудиторы выбирали «Управление образования области».
-- Нечёткое сопоставление по названию даёт ~49% совпадений по числу уведомлений.

\echo ==== 8. Устойчивость выбора адресата у одного ГУ ====
with pair as (
  select t.gu, r.its_cli_id, count(*) as raz
    from its_risk_notice n
    join its_risk_notice_5_15a sn on sn.its_risk_notice_id = n.id and sn."in$trash" is null
    join its_tb_5_15a t on t.id = sn.tb_5_15a_id
    join its_risk_notice_recepient r on r.its_risk_notice_id = n.id and r."in$trash" is null
   where n."in$trash" is null and coalesce(t.gu,'') <> ''
   group by t.gu, r.its_cli_id
), agg as (select gu, sum(raz) as vsego, max(raz) as top_raz from pair group by gu)
select width_bucket(top_raz::numeric / vsego, 0, 1.0001, 5) as kvintil_doli_top_adresata,
       count(*) as skolko_gu
  from agg group by 1 order by 1;
-- Факт: у 2927 ГУ доминирующий адресат покрывает 80-100% уведомлений,
-- у 1699 — только 40-60%. Полный автомат без подтверждения аудитором небезопасен.

\echo ==== 9. Расширения (доступно ли нечёткое сравнение на стороне БД) ====
select extname from pg_extension order by 1;
-- Факт: dblink, pgcrypto, plpgsql, uuid-ossp. pg_trgm и fuzzystrmatch НЕ установлены —
-- нечёткое сопоставление названий придётся делать в коде ЕХД.
