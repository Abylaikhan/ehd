-- Проверка боевой obm_evga: справочники для автоадресатов (замечание 3) и аудит схемы.
-- Выполнено 13.10.2026, результаты — в voprosy-analitiku-7-itogovyy.md.
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

-- ====================================================================
-- Запросы 10-14 добавлены 13.10.2026: ответы на вопросы аналитика из данных.
-- Результаты — в voprosy-analitiku-7-itogovyy.md.
-- ====================================================================

\echo ==== 10. ВОПРОС «код ДВГА»: заполненность кода департамента ====
select count(*) as vsego,
       count(*) filter (where coalesce(trim(code),'') <> '') as s_code,
       count(*) filter (where coalesce(trim(bin),'')  <> '') as s_bin
  from its_departments where "in$trash" is null;
select id, left(title,45) as title, coalesce(code,'(пусто)') as code
  from its_departments where "in$trash" is null order by id;
-- Факт: 22 департамента, код у 18. Формат — 5-значный бюджетный (30301, 35801).
-- Без кода: Абайская, Жетісу, Павлодарская, Улытау. Региональный код НЕ является
-- кодом ДВГА — подстановка отменена спекой 023.

\echo ==== 11. ВОПРОС «выбор существующего исходящего»: переиспользуется ли its_out ====
select uvedomleniy_na_odno_ishodyashchee, count(*) as skolko_ishodyashchih
  from (select o.id, count(n.id) as uvedomleniy_na_odno_ishodyashchee
          from its_out o
          join its_risk_notice n on n.its_out_id = o.id and n."in$trash" is null
         where o."in$trash" is null group by o.id) z
 group by 1 order by 1;
-- Факт: 26942 исходящих, ВСЕ 1:1 с уведомлением. Переиспользования нет ни разу —
-- функция «выбрать существующее письмо» не нужна.

\echo ==== 12. ВОПРОС «повторяющийся риск»: ключ ГУ+получатель+профиль по периодам ====
with k as (
  select gu, iin, its_risk_profile_id,
         count(distinct (god, mes)) as periodov,
         count(distinct its_risk_status_id) as raznyh_statusov,
         count(*) filter (where its_risk_status_id is not null and its_risk_status_id <> 1) as resheno
    from its_tb_5_15a
   where "in$trash" is null and coalesce(gu,'') <> '' and coalesce(iin,'') <> ''
     and its_risk_profile_id is not null
   group by 1,2,3
)
select count(*) as klyuchey_vsego,
       count(*) filter (where periodov > 1) as v_neskolkih_periodah,
       count(*) filter (where periodov > 1 and resheno >= 2) as est_2_resheniya,
       count(*) filter (where periodov > 1 and resheno >= 2 and raznyh_statusov = 1) as status_odinakovyy,
       count(*) filter (where periodov > 1 and resheno >= 2 and raznyh_statusov > 1)  as status_raznyy
  from k;
-- Факт: 1363625 / 649874 / 160375 / 56611 (35%) / 103764 (65%).
-- В 65% случаев аудиторы ставили РАЗНЫЕ статусы одному и тому же риску в разных
-- периодах → автоматический перенос статуса противоречит практике (замечание 5).

\echo ==== 13. ВОПРОС «адресаты»: адресат упомянут в названии самого ГУ? ====
-- Сравнение по префиксам слов (5 симв.), чтобы падежи не мешали.
with chosen as (
  select max(t.sendername) as gu_name, c.title as cli_title, count(*) as raz
    from its_risk_notice n
    join its_risk_notice_5_15a sn on sn.its_risk_notice_id = n.id and sn."in$trash" is null
    join its_tb_5_15a t on t.id = sn.tb_5_15a_id
    join its_risk_notice_recepient r on r.its_risk_notice_id = n.id and r."in$trash" is null
    join its_cli c on c.id = r.its_cli_id
   where n."in$trash" is null and coalesce(t.sendername,'') <> '' and coalesce(c.title,'') <> ''
   group by t.gu_bin, c.title
), w as (
  select raz,
    (select array_agg(distinct left(x,5)) from unnest(regexp_split_to_array(
        regexp_replace(lower(translate(gu_name,'«»"''','    ')),'[^a-zа-яәғқңөұүһі0-9]+',' ','g'),' ')) x
      where length(x)>=5 and x not in ('государственное','учреждение','коммунальное','коммуналдык','мемлекеттик','мекемеси')) as gu_w,
    (select array_agg(distinct left(x,5)) from unnest(regexp_split_to_array(
        regexp_replace(lower(translate(regexp_replace(cli_title,'\s+-\s+\d{4,}\s*$','','g'),'«»"''','    ')),'[^a-zа-яәғқңөұүһі0-9]+',' ','g'),' ')) x
      where length(x)>=5 and x not in ('государственное','учреждение','коммунальное','коммуналдык','мемлекеттик','мекемеси')) as cli_w
   from chosen
), f as (
  select raz, coalesce(array_length(cli_w,1),0) as cli_n,
         (select count(*) from unnest(cli_w) x where x = any(gu_w)) as sovpalo
    from w where cli_w is not null and gu_w is not null
)
select sum(raz) as sluchaev,
       round(100.0*sum(raz) filter (where sovpalo = cli_n)/sum(raz),1) as proc_polnostyu,
       round(100.0*sum(raz) filter (where cli_n>0 and sovpalo::numeric/cli_n >= 0.7)/sum(raz),1) as proc_v_osnovnom,
       round(100.0*sum(raz) filter (where sovpalo = 0)/sum(raz),1) as proc_ne_svyazan
  from f;
-- Факт: 1799437 случаев. Адресат полностью упомянут в названии ГУ — 55.2%,
-- в основном (>=70% слов) — 63.7%, никак не связан — 21.1%.
-- Это ЛУЧШЕ справочника подчинённости (40.6%). Вместе оба правила дают 66.7%.
-- Вывод: подсказка «2 случая из 3», полный автомат неприемлем.
