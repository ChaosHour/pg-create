\c ecommerce

-- 1. Role exists, can login, connection limit
SELECT rolname, rolcanlogin, rolconnlimit
FROM pg_roles
WHERE rolname = 'ecommerce_ro';

-- 2. CONNECT privilege on the database
SELECT has_database_privilege('ecommerce_ro', 'ecommerce', 'CONNECT') AS can_connect;

-- 3. USAGE privilege on schema store
SELECT has_schema_privilege('ecommerce_ro', 'store', 'USAGE') AS has_usage;

-- 4. SELECT granted on all current tables in store
SELECT table_name, privilege_type
FROM information_schema.role_table_grants
WHERE grantee = 'ecommerce_ro'
  AND table_schema = 'store'
ORDER BY table_name;

-- 5. SELECT granted on all current sequences in store
SELECT relname AS sequence_name,
       has_sequence_privilege('ecommerce_ro', n.nspname || '.' || c.relname, 'SELECT') AS can_select
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'S'
  AND n.nspname = 'store'
ORDER BY relname;

-- 6. Default privileges (what ecommerce_ro gets on FUTURE tables/sequences)
SELECT
    pg_get_userbyid(d.defaclrole) AS grantor,
    n.nspname                     AS schema,
    CASE d.defaclobjtype
        WHEN 'r' THEN 'tables'
        WHEN 'S' THEN 'sequences'
        WHEN 'f' THEN 'functions'
    END                           AS object_type,
    d.defaclacl                   AS acl
FROM pg_default_acl d
JOIN pg_namespace n ON n.oid = d.defaclnamespace
WHERE n.nspname = 'store'
ORDER BY object_type;

-- 7. All-in-one summary: which tables ecommerce_ro can SELECT
SELECT t.table_name,
       has_table_privilege('ecommerce_ro', 'store.' || t.table_name, 'SELECT') AS can_select
FROM information_schema.tables t
WHERE t.table_schema = 'store'
ORDER BY t.table_name;


SELECT * FROM pg_roles WHERE rolname = 'readonly';
SELECT * FROM information_schema.role_table_grants WHERE grantee='readonly';
