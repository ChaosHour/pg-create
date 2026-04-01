\set ON_ERROR_STOP on

-- Creates representative objects so grants can be validated consistently.
-- Usage example:
--   psql -h localhost -U postgres -d myapp_prod -v schema='myapp' -f sql/validation/seed_validation_objects.sql
--
-- Optional variables:
--   schema : target schema for test objects (default: public)

\if :{?schema}
\else
\set schema public
\endif

CREATE TEMP TABLE IF NOT EXISTS pg_temp.seed_report (
    step text,
    detail text
);

SELECT format('CREATE SCHEMA IF NOT EXISTS %I', :'schema') AS ddl \gexec

SELECT format(
$$
CREATE TABLE IF NOT EXISTS %1$I.grant_validation_table (
    id BIGSERIAL PRIMARY KEY,
    payload TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
)
$$,
:'schema') AS ddl \gexec

SELECT format(
$$
CREATE OR REPLACE FUNCTION %1$I.grant_validation_fn(input_text TEXT)
RETURNS TEXT
LANGUAGE sql
AS $fn$
    SELECT upper(input_text)
$fn$
$$,
:'schema') AS ddl \gexec

INSERT INTO pg_temp.seed_report(step, detail)
SELECT 'seed', format('Objects ensured in schema %I', :'schema')
FROM (SELECT 1) t;

SELECT * FROM pg_temp.seed_report;
