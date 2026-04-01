# SQL Grant Validation Toolkit

These scripts help validate grants created by pg-create at database, schema, table, sequence, and function levels.

## Files

- `validate_grants.sql`: comprehensive readout for role existence, CONNECT, schema privileges, table/sequence/function privileges, and default privileges.
- `seed_validation_objects.sql`: creates a predictable table + function for privilege checks.

## Quick Start

1. Provision with pg-create.
2. Seed validation objects (optional but recommended if schema may be empty):

```bash
psql -h localhost -U postgres -d myapp_prod \
  -v schema='myapp' \
  -f sql/validation/seed_validation_objects.sql
```

3. Validate grants for one or more roles:

```bash
psql -h localhost -U postgres -d myapp_prod \
  -v roles='myapp_app,myapp_ro,myapp_dba' \
  -v schema='myapp' \
  -f sql/validation/validate_grants.sql
```

## Notes

- `roles` defaults to `readonly` when omitted.
- `schema` defaults to empty, which means all non-system schemas.
- If there are no objects in a schema, object-level sections will return empty result sets; this is expected.
