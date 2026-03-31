# Agent prompt: PostgreSQL resource provisioning CLI (pg-create)

You are an elite Golang Developer. Build a Go CLI to create PostgreSQL resources (users, roles, database, schemas, extensions, grants) in a safe, idempotent way.

## Requirements

- `--database`/`-d`, `--user`/`-u`, `--password`/`-p`, `--schema`/`-sc`, `--roles`/`-r`, `--grants`/`-g`, `--search-path`/`-sp`, `--host`/`-s`
- Idempotent operations: check for existence before create.
- Manage superuser vs role behavior, with controlled grants.
- Provide clear output for existing vs created entities.
- Avoid production unsafe operations without flags or confirmation.

## Desired default process (programmatic equivalent)

1. create database if not exists
2. create role/user(s) if not exists
3. grant connection limits and login
4. create schemas if not exists
5. assign schema ownership
6. create extensions in target schema(s) if not exists
7. grant schema usage function/sequence/table access to roles/users
8. set role search_path
9. set default privileges for table and sequence creation

## Reference SQL (idempotent style)

```sql
-- 1. Database
CREATE DATABASE IF NOT EXISTS myapp_prod;

-- 2. Users/Roles
CREATE ROLE myapp_prod_dba LOGIN PASSWORD 'DBAxxxxx' CONNECTION LIMIT 10;
CREATE ROLE myapp_prod_app LOGIN PASSWORD 'APPxxxxx';

-- 3. Schemas
CREATE SCHEMA IF NOT EXISTS myapp;
CREATE SCHEMA IF NOT EXISTS ext;

-- 4. Schema ownership
ALTER SCHEMA myapp OWNER TO myapp_prod_dba;
ALTER SCHEMA ext OWNER TO myapp_prod_dba;

-- 5. Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" SCHEMA ext;
CREATE EXTENSION IF NOT EXISTS pg_trgm SCHEMA ext;
CREATE EXTENSION IF NOT EXISTS hstore SCHEMA ext;

-- 6. Grants
GRANT USAGE ON SCHEMA ext TO myapp_prod_dba;
GRANT USAGE ON SCHEMA ext TO myapp_prod_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA ext TO myapp_prod_dba;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA ext TO myapp_prod_app;

ALTER USER myapp_prod_dba SET search_path = myapp, ext;
ALTER USER myapp_prod_app SET search_path = myapp, ext;

GRANT USAGE ON SCHEMA myapp TO myapp_prod_app;
GRANT USAGE ON SCHEMA ext TO myapp_prod_app;

-- 7. Default privileges for app role
ALTER DEFAULT PRIVILEGES IN SCHEMA myapp GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO myapp_prod_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA myapp GRANT SELECT, UPDATE, USAGE ON SEQUENCES TO myapp_prod_app;

-- 8. Read-only role (if required)
CREATE ROLE myapp_prod_ro LOGIN PASSWORD 'ROxxxxx' CONNECTION LIMIT 10;
ALTER USER myapp_prod_ro SET search_path = myapp, ext;

GRANT USAGE ON SCHEMA ext TO myapp_prod_ro;
GRANT USAGE ON SCHEMA myapp TO myapp_prod_ro;
GRANT SELECT ON ALL TABLES IN SCHEMA myapp TO myapp_prod_ro;
GRANT SELECT ON ALL SEQUENCES IN SCHEMA myapp TO myapp_prod_ro;
ALTER DEFAULT PRIVILEGES IN SCHEMA myapp GRANT SELECT ON TABLES TO myapp_prod_ro;
ALTER DEFAULT PRIVILEGES IN SCHEMA myapp GRANT SELECT ON SEQUENCES TO myapp_prod_ro;
```

I need you to evalute the methods used. I don't just want to create 2 useres and a database, I want to ensure the CLI is robust, handles errors gracefully, and provides clear feedback on what was created vs what already exists. The CLI should also be flexible enough to allow for different configurations (e.g., different extensions, roles, grants) without hardcoding values.  Schema ownership and search_path management should be handled carefully to avoid security issues. The CLI should also have safeguards against running potentially destructive operations in production without explicit confirmation.

## Go CLI design considerations

- Use cmd/pgcreate/main.go and pkg for any modules.
- If it's a small program then you do not have to add a cli framework.
- Use a color pakge for any logfiles or output to the terminal.
- Use a configuration file (e.g., YAML or JSON) to define the desired state of the PostgreSQL resources, allowing for flexibility and reusability.
- Implement a dry-run mode to show what changes would be made without actually executing them, providing an extra layer of safety.
- Include comprehensive error handling and logging to ensure that any issues are clearly communicated to the user.
- Consider adding a rollback mechanism in case of failures during the provisioning process, to maintain the integrity of the database state.
- Create a Makefile to build the binary and put it in ./bin/pg-create for easy execution.
- Update the READEME.md with usage instructions, examples, and any necessary environment variable configurations.
- When possible, I like to use the ~/.pgpass file or even a json file for storing database credentials securely, and the CLI should support reading from these files for authentication.

## Notes

- This file is a design/agent prompt; actual code lives in `main.go`, `db.go`, `user.go`.
- Ensure your Go CLI mirrors the above SQL flow and error handling.
