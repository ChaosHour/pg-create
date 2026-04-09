\set ON_ERROR_STOP on

-- Grant validation script for pg-create outputs.
-- Usage example:
--   psql -h localhost -U postgres -d myapp_prod \
--     -v roles='myapp_app,myapp_ro' \
--     -v schema='myapp' \
--     -f sql/validation/validate_grants.sql
--
-- Optional variables:
--   roles   : comma-separated role names to validate (default: readonly)
--   schema  : restrict validation to one schema; empty means all non-system schemas (default: empty)

\if :{?roles}
\else
\set roles readonly
\endif

\if :{?schema}
\else
\set schema ''
\endif

\echo
\echo ===== Validation Input =====
SELECT current_database() AS connected_database,
       current_user AS executing_user,
       :'roles' AS target_roles,
       NULLIF(:'schema', '') AS schema_filter;

CREATE TEMP TABLE _target_roles AS
SELECT trim(value) AS rolname
FROM regexp_split_to_table(:'roles', ',') AS value
WHERE trim(value) <> '';

\echo
\echo ===== 1) Role existence and attributes =====
SELECT tr.rolname AS requested_role,
       r.rolname IS NOT NULL AS role_exists,
       r.rolcanlogin,
       r.rolconnlimit,
       r.rolsuper,
       r.rolinherit,
       r.rolcreatedb,
       r.rolcreaterole,
       r.rolreplication,
       r.rolbypassrls
FROM _target_roles tr
LEFT JOIN pg_roles r ON r.rolname = tr.rolname
ORDER BY tr.rolname;

\echo
\echo ===== 2) Database CONNECT privilege =====
SELECT tr.rolname,
       has_database_privilege(tr.rolname, current_database(), 'CONNECT') AS can_connect
FROM _target_roles tr
ORDER BY tr.rolname;

\echo
\echo ===== 3) Schema privileges (USAGE/CREATE) =====
WITH target_schemas AS (
    SELECT n.nspname
    FROM pg_namespace n
    WHERE n.nspname NOT LIKE 'pg_%'
      AND n.nspname <> 'information_schema'
      AND (NULLIF(:'schema', '') IS NULL OR n.nspname = NULLIF(:'schema', ''))
)
SELECT tr.rolname,
       ts.nspname AS schema_name,
       has_schema_privilege(tr.rolname, ts.nspname, 'USAGE') AS has_usage,
       has_schema_privilege(tr.rolname, ts.nspname, 'CREATE') AS has_create
FROM _target_roles tr
CROSS JOIN target_schemas ts
ORDER BY tr.rolname, ts.nspname;

\echo
\echo ===== 4) Table privileges =====
WITH target_schemas AS (
    SELECT n.nspname
    FROM pg_namespace n
    WHERE n.nspname NOT LIKE 'pg_%'
      AND n.nspname <> 'information_schema'
      AND (NULLIF(:'schema', '') IS NULL OR n.nspname = NULLIF(:'schema', ''))
), target_tables AS (
    SELECT t.table_schema, t.table_name
    FROM information_schema.tables t
    JOIN target_schemas s ON s.nspname = t.table_schema
    WHERE t.table_type = 'BASE TABLE'
)
SELECT tr.rolname,
       tt.table_schema,
       tt.table_name,
       has_table_privilege(tr.rolname, format('%I.%I', tt.table_schema, tt.table_name), 'SELECT') AS can_select,
       has_table_privilege(tr.rolname, format('%I.%I', tt.table_schema, tt.table_name), 'INSERT') AS can_insert,
       has_table_privilege(tr.rolname, format('%I.%I', tt.table_schema, tt.table_name), 'UPDATE') AS can_update,
       has_table_privilege(tr.rolname, format('%I.%I', tt.table_schema, tt.table_name), 'DELETE') AS can_delete,
       has_table_privilege(tr.rolname, format('%I.%I', tt.table_schema, tt.table_name), 'TRUNCATE') AS can_truncate,
       has_table_privilege(tr.rolname, format('%I.%I', tt.table_schema, tt.table_name), 'REFERENCES') AS can_references,
       has_table_privilege(tr.rolname, format('%I.%I', tt.table_schema, tt.table_name), 'TRIGGER') AS can_trigger
FROM _target_roles tr
CROSS JOIN target_tables tt
ORDER BY tr.rolname, tt.table_schema, tt.table_name;

\echo
\echo ===== 5) Sequence privileges =====
WITH target_schemas AS (
    SELECT n.nspname
    FROM pg_namespace n
    WHERE n.nspname NOT LIKE 'pg_%'
      AND n.nspname <> 'information_schema'
      AND (NULLIF(:'schema', '') IS NULL OR n.nspname = NULLIF(:'schema', ''))
)
SELECT tr.rolname,
       n.nspname AS sequence_schema,
       c.relname AS sequence_name,
       has_sequence_privilege(tr.rolname, format('%I.%I', n.nspname, c.relname), 'SELECT') AS can_select,
       has_sequence_privilege(tr.rolname, format('%I.%I', n.nspname, c.relname), 'USAGE') AS can_usage,
       has_sequence_privilege(tr.rolname, format('%I.%I', n.nspname, c.relname), 'UPDATE') AS can_update
FROM _target_roles tr
JOIN target_schemas ts ON true
JOIN pg_namespace n ON n.nspname = ts.nspname
JOIN pg_class c ON c.relnamespace = n.oid
WHERE c.relkind = 'S'
ORDER BY tr.rolname, n.nspname, c.relname;

\echo
\echo ===== 6) Function EXECUTE privileges =====
WITH target_schemas AS (
    SELECT n.nspname
    FROM pg_namespace n
    WHERE n.nspname NOT LIKE 'pg_%'
      AND n.nspname <> 'information_schema'
      AND (NULLIF(:'schema', '') IS NULL OR n.nspname = NULLIF(:'schema', ''))
), target_functions AS (
    SELECT n.nspname,
           p.proname,
           p.oid AS function_oid
    FROM pg_proc p
    JOIN pg_namespace n ON n.oid = p.pronamespace
    JOIN target_schemas s ON s.nspname = n.nspname
)
SELECT tr.rolname,
       tf.nspname AS function_schema,
       tf.proname AS function_name,
       has_function_privilege(tr.rolname, tf.function_oid::regprocedure, 'EXECUTE') AS can_execute
FROM _target_roles tr
CROSS JOIN target_functions tf
ORDER BY tr.rolname, tf.nspname, tf.proname;

\echo
\echo ===== 7) Default privileges for target roles =====
SELECT
    pg_get_userbyid(d.defaclrole) AS grantor,
    COALESCE(n.nspname, '(all)') AS schema,
    CASE d.defaclobjtype
        WHEN 'r' THEN 'tables'
        WHEN 'S' THEN 'sequences'
        WHEN 'f' THEN 'functions'
        WHEN 'T' THEN 'types'
        WHEN 'n' THEN 'schemas'
        ELSE d.defaclobjtype::text
    END AS object_type,
    pg_get_userbyid(a.grantee) AS grantee,
    a.privilege_type,
    a.is_grantable
FROM pg_default_acl d
LEFT JOIN pg_namespace n ON n.oid = d.defaclnamespace
CROSS JOIN LATERAL aclexplode(COALESCE(d.defaclacl, '{}'::aclitem[])) a
JOIN _target_roles tr ON tr.rolname = pg_get_userbyid(a.grantee)
WHERE (NULLIF(:'schema', '') IS NULL OR n.nspname = NULLIF(:'schema', ''))
ORDER BY grantee, schema, object_type, privilege_type;

\echo
\echo ===== 8) Summary =====
WITH target_schemas AS (
    SELECT n.nspname
    FROM pg_namespace n
    WHERE n.nspname NOT LIKE 'pg_%'
      AND n.nspname <> 'information_schema'
      AND (NULLIF(:'schema', '') IS NULL OR n.nspname = NULLIF(:'schema', ''))
), table_counts AS (
    SELECT t.table_schema, count(*)::int AS table_count
    FROM information_schema.tables t
    JOIN target_schemas s ON s.nspname = t.table_schema
    WHERE t.table_type = 'BASE TABLE'
    GROUP BY t.table_schema
), sequence_counts AS (
    SELECT n.nspname AS sequence_schema, count(*)::int AS sequence_count
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    JOIN target_schemas s ON s.nspname = n.nspname
    WHERE c.relkind = 'S'
    GROUP BY n.nspname
), function_counts AS (
    SELECT n.nspname AS function_schema, count(*)::int AS function_count
    FROM pg_proc p
    JOIN pg_namespace n ON n.oid = p.pronamespace
    JOIN target_schemas s ON s.nspname = n.nspname
    GROUP BY n.nspname
)
SELECT s.nspname AS schema_name,
       COALESCE(tc.table_count, 0) AS tables_found,
       COALESCE(sc.sequence_count, 0) AS sequences_found,
       COALESCE(fc.function_count, 0) AS functions_found,
       CASE
           WHEN COALESCE(tc.table_count, 0) = 0
             AND COALESCE(sc.sequence_count, 0) = 0
             AND COALESCE(fc.function_count, 0) = 0
           THEN 'No objects found in schema (object-level checks may be empty)'
           ELSE 'Objects found; review role privilege rows above'
       END AS interpretation
FROM target_schemas s
LEFT JOIN table_counts tc ON tc.table_schema = s.nspname
LEFT JOIN sequence_counts sc ON sc.sequence_schema = s.nspname
LEFT JOIN function_counts fc ON fc.function_schema = s.nspname
ORDER BY s.nspname;
