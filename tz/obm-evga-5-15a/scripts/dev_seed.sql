-- Сид синтетических тестовых пользователей для dev-реплики (на проде их НЕТ).
-- Нужны для маппинга по ИИН: ehd2-логины auditor1/approver1 имеют ИИН 999999999999/998,
-- которые резолвятся в obm_evga.users → департамент. После DROP DATABASE эти строки исчезают,
-- поэтому пересоздаём их при каждой загрузке реплики.
--
-- Департамент выбираем как САМЫЙ ЧАСТЫЙ среди загруженных уведомлений — чтобы тестовый
-- аудитор гарантированно видел непустой список (видимость по департаменту).
set session_replication_role = replica; -- FK-триггеры off (сэмплы неполны)

delete from users where id in (9100001, 9100002);

with dep as (
  select its_departments_id as id
  from its_risk_notice
  where "in$trash" is null and its_departments_id is not null
  group by its_departments_id
  order by count(*) desc
  limit 1
)
insert into users (id, iin, short_name, login, its_departments_id)
select 9100001, '999999999999', 'Тестов Т.Т.',      'testov',  (select id from dep)
union all
select 9100002, '999999999998', 'Руководитель Р.Р.', 'rukovod', (select id from dep);

-- Информационно: в какой департамент сели тестовые юзеры
select 'dev_seed: тестовые юзеры в департаменте '||coalesce(its_departments_id::text,'—')||
       ' ('||coalesce((select title from its_departments d where d.id = u.its_departments_id), '?')||')'
from users u where u.id = 9100001;
