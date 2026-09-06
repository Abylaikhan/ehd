-- EVGA-DB-002 (ТЗ §14.2). Идемпотентно.
-- Вывод из оборота неиспользуемых статусов витрины (их title уже «-»).
update public.its_risk_status
   set "in$trash" = 1
 where id in (3, 5, 9, 10)
   and "in$trash" is distinct from 1;

-- title_kk действующих статусов заполняется отдельно значениями заказчика (ТЗ §14.2).
