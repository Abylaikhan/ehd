-- Локальный sandbox: тестовый источник данных Reporter.
-- Креды фиксированы и используются только в локальном окружении.

CREATE DATABASE IF NOT EXISTS ehd_src;

CREATE TABLE IF NOT EXISTS ehd_src.demo_transactions
(
    id              UInt64,
    full_name       String,
    amount          Decimal(18, 2),
    region_code     LowCardinality(String),
    department_code LowCardinality(String),
    created_at      DateTime
)
ENGINE = MergeTree
ORDER BY id;

INSERT INTO ehd_src.demo_transactions
SELECT
    number + 1,
    concat('Клиент ', toString(number + 1)),
    toDecimal64((rand() % 1000000) / 100, 2),
    ['AST', 'ALA', 'SHY'][(number % 3) + 1],
    ['D01', 'D02'][(number % 2) + 1],
    now() - toIntervalDay(rand() % 365)
FROM numbers(1000);

-- readonly = 2: SELECT разрешён, DDL/DML запрещены, клиентские settings (timeout, query_id) разрешены
CREATE USER IF NOT EXISTS reporter_ro
    IDENTIFIED WITH sha256_password BY 'reporter_ro_local'
    SETTINGS readonly = 2;

GRANT SELECT ON ehd_src.* TO reporter_ro;
