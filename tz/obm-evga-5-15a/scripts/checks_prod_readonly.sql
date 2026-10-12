-- Проверки боевой БД ОБМ ЭВГА (obm_evga@10.58.57.39). ТОЛЬКО ЧТЕНИЕ.
-- Назначение: закрыть открытые вопросы по замечаниям к тестовой системе и понять,
-- какие выводы команды сделаны на обрезанной копии, а какие верны для боевой базы.
--
-- Запуск (сессия принудительно read-only):
--   PGOPTIONS='-c default_transaction_read_only=on' \
--     psql "postgres://<ro_user>:<pass>@10.58.57.39:5432/obm_evga?sslmode=prefer" \
--     -f checks_prod_readonly.sql > checks_out.txt 2>&1
--
-- Результат (checks_out.txt) прислать команде разработки. Персональные данные
-- в выводе не фигурируют: выводятся только счётчики и обезличенные примеры.

\echo '==================== 1. СПРАВОЧНИК АДРЕСАТОВ (вопрос: авто-адресаты, п.3) ===================='
-- Если bin_iin/parent_element_guid заполнены — автоматическое определение адресатов реализуемо.
SELECT count(*)                                                   AS vsego,
       count(*) FILTER (WHERE coalesce(bin_iin,'') <> '')          AS s_bin,
       count(*) FILTER (WHERE coalesce(code,'') <> '')             AS s_code,
       count(*) FILTER (WHERE coalesce(kod,'') <> '')              AS s_kod,
       count(*) FILTER (WHERE parent_element_guid IS NOT NULL)     AS s_roditelem,
       count(*) FILTER (WHERE element_guid IS NOT NULL)            AS s_guid
FROM its_cli
WHERE "in$trash" IS NULL;

\echo '-- 1.1 Чем ГУ витрины сходится со справочником: ищем рабочий ключ связи'
WITH gu AS (
    SELECT DISTINCT gu, gu_bin
    FROM its_tb_5_15a
    WHERE "in$trash" IS NULL AND coalesce(gu,'') <> ''
    LIMIT 500
)
SELECT
    (SELECT count(*) FROM gu JOIN its_cli c ON c.bin_iin = gu.gu_bin AND c."in$trash" IS NULL) AS po_bin,
    (SELECT count(*) FROM gu JOIN its_cli c ON c.code    = gu.gu     AND c."in$trash" IS NULL) AS po_code,
    (SELECT count(*) FROM gu JOIN its_cli c ON c.kod     = gu.gu     AND c."in$trash" IS NULL) AS po_kod,
    (SELECT count(*) FROM gu) AS vsego_gu;

\echo '-- 1.2 Строится ли цепочка «организация → вышестоящая» (пример, без ПДн)'
SELECT left(c.title, 45) AS organizaciya, left(p.title, 45) AS vyshestoyashchaya
FROM its_cli c
JOIN its_cli p ON p.element_guid = c.parent_element_guid AND p."in$trash" IS NULL
WHERE c."in$trash" IS NULL
LIMIT 10;

\echo '==================== 2. НАСКОЛЬКО ОБРЕЗАНА НАША КОПИЯ ===================='
-- В копии эти таблицы пусты/почти пусты; если в боевой они заполнены,
-- часть выводов команды (в т.ч. «блокеры») к боевой базе не относится.
SELECT 'its_req_stat (статусы заявки)' AS tablica, count(*) AS strok FROM its_req_stat
UNION ALL SELECT 'users_import',                     count(*) FROM users_import
UNION ALL SELECT 'users с ФИО (short_name)',         count(*) FROM users WHERE coalesce(short_name,'') <> ''
UNION ALL SELECT 'users всего',                      count(*) FROM users WHERE "in$trash" IS NULL
UNION ALL SELECT 'hr_position (должности)',          count(*) FROM hr_position
UNION ALL SELECT 'its_cli',                          count(*) FROM its_cli WHERE "in$trash" IS NULL
UNION ALL SELECT 'its_departments',                  count(*) FROM its_departments WHERE "in$trash" IS NULL;

\echo '==================== 3. КАСКАД СТАТУСОВ: п.1 против п.5 ===================='
-- Если платежей с РАЗНЫМИ статусами по профилям много — аудиторы уже отрабатывают
-- риски раздельно, и каскад по платежу мешает (аргумент за п.1).
-- Если почти нет — каскад соответствует практике (аргумент за п.5).
SELECT count(*) AS platezhey_s_raznymi_statusami
FROM (
    SELECT ppo_pp
    FROM its_tb_5_15a
    WHERE "in$trash" IS NULL AND coalesce(ppo_pp,'') <> ''
    GROUP BY ppo_pp
    HAVING count(DISTINCT its_risk_status_id) > 1
) x;

\echo '-- 3.1 Для масштаба: сколько всего платежей и сколько из них многопрофильные'
SELECT count(*) AS vsego_platezhey,
       count(*) FILTER (WHERE profiley > 1) AS mnogoprofilnyh
FROM (
    SELECT ppo_pp, count(DISTINCT its_risk_profile_id) AS profiley
    FROM its_tb_5_15a
    WHERE "in$trash" IS NULL AND coalesce(ppo_pp,'') <> ''
    GROUP BY ppo_pp
) y;

\echo '==================== 4. ДЕЙСТВУЮЩАЯ ФОРМА УВЕДОМЛЕНИЯ (п.2) ===================='
\echo '-- 4.1 Бывает ли несколько адресатов и осмыслен ли порядок nn'
SELECT adresatov, count(*) AS uvedomleniy
FROM (
    SELECT its_risk_notice_id, count(*) AS adresatov
    FROM its_risk_notice_recepient
    WHERE "in$trash" IS NULL
    GROUP BY its_risk_notice_id
) z
GROUP BY adresatov
ORDER BY adresatov;

\echo '-- 4.2 Заполняется ли nn (порядок) или он всегда пустой/единица'
SELECT coalesce(nn::text, 'NULL') AS nn, count(*)
FROM its_risk_notice_recepient
WHERE "in$trash" IS NULL
GROUP BY 1 ORDER BY 1;

\echo '-- 4.3 ГДЕ ХРАНИТСЯ «КОММЕНТАРИЙ» к уведомлению: есть ли подходящая колонка'
-- Если такая колонка есть, наше хранение комментария в БД ЕХД нужно переключить на неё.
SELECT column_name, data_type
FROM information_schema.columns
WHERE table_schema = 'public' AND table_name = 'its_risk_notice'
ORDER BY ordinal_position;

\echo '-- 4.4 Как привязано исходящее: у скольких уведомлений заполнен its_out_id'
SELECT count(*) AS vsego_uvedomleniy,
       count(its_out_id) AS s_ishodyashchim
FROM its_risk_notice WHERE "in$trash" IS NULL;

\echo '==================== 5. «ОТПРАВИТЕЛЬ» В СПИСКЕ УВЕДОМЛЕНИЙ (п.4) ===================='
-- Что действующая система считает отправителем: департамент ДВГА или ГУ-отправитель платежа.
SELECT n.id,
       left(coalesce(d.title,''), 40)  AS departament_dvga,
       left(coalesce(sn.sendername,''), 40) AS gu_otpravitel_platezha
FROM its_risk_notice n
LEFT JOIN its_departments d ON d.id = n.its_departments_id
LEFT JOIN LATERAL (
    SELECT min(sendername) AS sendername
    FROM its_risk_notice_5_15a s
    WHERE s.its_risk_notice_id = n.id AND s."in$trash" IS NULL
) sn ON true
WHERE n."in$trash" IS NULL
ORDER BY n.id DESC
LIMIT 5;

\echo '==================== 6. СОВМЕСТИМОСТЬ СХЕМЫ С НАШИМИ ДОРАБОТКАМИ ===================='
-- Колонки, на которые опирается код модуля. Если какой-то нет — доработка там не заведётся.
SELECT t.tbl, t.col,
       CASE WHEN c.column_name IS NULL THEN 'НЕТ ❌' ELSE 'есть' END AS nalichie
FROM (VALUES
        ('its_tb_5_15a',               'risk_status_note'),
        ('its_tb_5_15a',               'ppo_pp'),
        ('its_tb_5_15a',               'la1'),
        ('its_tb_5_15a',               'its_departments_id'),
        ('its_out',                    'exec_due_time'),
        ('its_out',                    'doc_num'),
        ('its_risk_notice_recepient',  'nn'),
        ('its_risk_notice',            'its_out_id'),
        ('its_risk_notice',            'signer_id'),
        ('its_departments',            'code'),
        ('its_departments',            'its_regions_id'),
        ('its_departments',            'notice_num'),
        ('its_departments',            'notice_num_year'),
        ('its_regions',                'region_code'),
        ('users',                      'iin'),
        ('users',                      'its_departments_id'),
        ('users',                      'short_name')
     ) AS t(tbl, col)
LEFT JOIN information_schema.columns c
       ON c.table_schema = 'public' AND c.table_name = t.tbl AND c.column_name = t.col
ORDER BY t.tbl, t.col;

\echo '==================== ГОТОВО ===================='
