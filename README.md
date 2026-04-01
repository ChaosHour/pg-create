# pg-create

PostgreSQL Resource Provisioning CLI - A safe, idempotent tool for creating and managing PostgreSQL databases, roles, schemas, extensions, and grants.

## Features

- **Idempotent Operations** - Safe to run multiple times
- **Flexible Configuration** - Use CLI flags or YAML/JSON config files
- **Dry Run Mode** - Preview changes before applying
- **Environment Safeguards** - Confirmation prompts for prod/qa
- **Role-based Privileges** - Automatic privilege templates (app, ro, dba)
- **Security-focused SQL handling** - identifier quoting and input validation for dynamic statements
- **Clear Output** - Visual indicators for created vs existing resources

## Installation

### Build from Source

```bash
# Clone the repository
git clone https://github.com/ChaosHour/pg-create.git
cd pg-create

# Build using Makefile
make build

# Binary will be in ./bin/pg-create
```

### Install to GOPATH

```bash
make install
```

## Usage

### Using CLI Flags

```bash
./bin/pg-create \
  -s localhost \
  -u postgres \
  -p your_password \
  -d myapp_prod \
  -sc myapp,ext \
  -r myapp_dba:pass1:dba,myapp_app:pass2:app,myapp_ro:pass3:ro \
  -e uuid-ossp,pg_trgm,hstore \
  -sp "myapp, ext" \
  -g usage,select,insert,update,delete,execute \
  -env standalone
```

### Using Config File (YAML)

```bash
# Create your config file (see config.example.yaml)
cp config.example.yaml config.yaml

# Edit with your settings
vim config.yaml

# Run with config
./bin/pg-create -c config.yaml
```

### Using Config File (JSON)

```bash
# Create your config file (see config.example.json)
cp config.example.json config.json

# Edit with your settings
vim config.json

# Run with config
./bin/pg-create -c config.json
```

### Dry Run Mode

Preview what changes would be made without executing them:

```bash
./bin/pg-create -c config.yaml -dry-run
```

Note: Go's flag parser accepts both `-dry-run` and `--dry-run`.

## Configuration

### CLI Flags

| Flag | Description | Required |
|------|-------------|----------|
| `-s` | PostgreSQL host | Yes (unless `-c` is used) |
| `-u` | Admin user for connection | Yes (unless `-c` is used) |
| `-p` | Admin password (optional when `~/.pgpass` matches) | No |
| `-d` | Database name to create | Yes (unless `-c` is used) |
| `-port` | Port (default: 5432) | No |
| `-sc` | Comma-separated schemas | No |
| `-r` | Roles (`name:password:type`, type is `app|ro|dba`) | No |
| `-e` | Comma-separated extensions | No |
| `-sp` | Search path (comma-separated schemas) | No |
| `-g` | Comma-separated grants | No |
| `-env` | Environment (`standalone|qa|prod`) | No |
| `-c` | Config file (YAML/JSON) | No |
| `-dry-run` | Preview mode (no changes applied) | No |
| `-h` | Show help | No |

### Role Types

- **dba** - Database administrator (full privileges, 10 connections)
- **app** - Application role (CRUD privileges, unlimited connections)
- **ro** - Read-only role (SELECT only, 10 connections)

### Input Validation

Invalid role types and grants fail fast.

```bash
# Invalid role type example
./bin/pg-create -s localhost -u postgres -d myapp -r app_user:secret:readonly
# Output: Invalid configuration: invalid role type "readonly" for role "app_user" (must be app, ro, or dba)

# Invalid grant example
./bin/pg-create -s localhost -u postgres -d myapp -g select,truncate
# Output: Invalid configuration: invalid grant type "truncate"
```

### Grant Types

- `usage` - Schema usage
- `select` - Read tables and sequences
- `insert` - Insert into tables
- `update` - Update tables and sequences
- `delete` - Delete from tables
- `execute` - Execute functions

## Provisioning Flow

The CLI follows this order:

1. **Create Database** - If not exists
2. **Create Roles** - With passwords and connection limits
3. **Create Schemas** - If not exists
4. **Create Extensions** - In specified schema
5. **Apply Grants** - Based on grant types
6. **Set Search Paths** - For each role
7. **Apply Default Privileges** - Based on role type

## Examples

### Simple Database Setup

```bash
./bin/pg-create \
  -s localhost -u postgres -p secret \
  -d testdb -sc public
```

### Full Application Setup

```yaml
# config.yaml
host: localhost
port: 5432
user: postgres
password: admin_pass
database: myapp_prod
environment: prod

schemas:
  - myapp
  - ext

roles:
  - name: myapp_dba
    password: dba_secure_pass
    type: dba
  - name: myapp_app
    password: app_secure_pass
    type: app
  - name: myapp_ro
    password: ro_secure_pass
    type: ro

extensions:
  - uuid-ossp
  - pg_trgm

grants:
  - usage
  - select
  - insert
  - update
  - delete
  - execute

search_path: myapp, ext
```

```bash
./bin/pg-create -c config.yaml
```

## Grant Validation

Use the SQL validation toolkit in `sql/validation` to verify database, schema, table, sequence, function, and default privileges after provisioning.

### 1) Seed validation objects (optional but recommended)

```bash
psql -h localhost -U postgres -d myapp_prod \
  -v schema='myapp' \
  -f sql/validation/seed_validation_objects.sql
```

### 2) Validate grants for one or more roles

```bash
psql -h localhost -U postgres -d myapp_prod \
  -v roles='myapp_app,myapp_ro,myapp_dba' \
  -v schema='myapp' \
  -f sql/validation/validate_grants.sql
```

Notes:
- `roles` defaults to `readonly` when omitted.
- `schema` defaults to empty, which validates all non-system schemas.
- If a schema has no objects yet, object-level result sets can be empty.

## Validation CLI

`pg-validate` is a second CLI in this repo for role/user grant inspection.

### Build

```bash
make build-validate
```

### Usage

```bash
./bin/pg-validate \
  -s localhost \
  -u postgres \
  -db myapp_prod \
  -roles myapp_app,myapp_ro,myapp_dba \
  -schema myapp
```

Key flags:
- `-s`: host
- `-port`: port (default `5432`)
- `-u`: admin user for inspection queries
- `-p`: admin password (optional when `~/.pgpass` matches)
- `-db`: database to inspect
- `-roles`: one role or comma-separated role list
- `-schema`: optional schema filter

## Development

### Project Structure

```
pg-create/
├── cmd/
│   └── pgcreate/
│       └── main.go          # Entry point
│   └── pgvalidate/
│       └── main.go          # Validation CLI entry point
├── pkg/
│   ├── config/
│   │   └── config.go        # Configuration handling
│   └── database/
│       └── provisioner.go   # Database provisioning logic
│   └── validator/
│       └── validator.go      # Validation report logic
├── sql/
│   └── validation/
│       ├── validate_grants.sql
│       ├── seed_validation_objects.sql
│       └── README.md
├── bin/                     # Build output
├── Makefile                 # Build automation
├── config.example.yaml      # Example YAML config
├── config.example.json      # Example JSON config
└── README.md
```

### Build Commands

```bash
make build      # Build binary
make build-validate  # Build validator CLI
make build-all   # Build both CLIs
make clean      # Clean artifacts
make deps       # Install dependencies
make test       # Run tests
make install    # Install to GOPATH
make help       # Show all targets
```

## Security Considerations

- Never commit config files with real passwords
- Use `.pgpass` file for credentials when possible
- Always use `-dry-run` first in production
- The CLI will prompt for confirmation in prod/qa environments
- Dynamic SQL paths use identifier quoting and validated inputs; avoid untrusted config values

## Troubleshooting

### Common Errors

| Error message | Likely cause | Fix |
|---|---|---|
| `Missing required flags: -s (host), -u (user), -d (database)` | Required flags were not provided when not using `-c` | Provide `-s`, `-u`, and `-d`, or use `-c config.yaml` |
| `No password provided: use -p flag or add a matching entry to ~/.pgpass` | No `-p` and no matching `~/.pgpass` entry | Add `-p`, or add `host:port:postgres:user:password` to `~/.pgpass` |
| `Failed to load config file: ...` | Config file path invalid or unreadable | Verify path and file permissions |
| `unsupported config file format: ...` | Config file extension is not `.yaml`, `.yml`, or `.json` | Rename or convert config file to supported format |
| `Invalid configuration: invalid role type "..."` | Role type not one of `app`, `ro`, `dba` | Update role spec or config to one of supported role types |
| `Invalid configuration: invalid grant type "..."` | Unsupported grant value | Use only `usage`, `select`, `insert`, `update`, `delete`, `execute` |
| `failed to connect to database: ...` | Host/port/user/password/network issue | Validate connection info with `psql` and `pg_isready` |
| `invalid search_path "...": must contain at least one schema` | `-sp`/`search_path` is empty or contains only commas/spaces | Set a valid schema list, for example `-sp "myapp,public"` |
| `Operation cancelled by user` | Confirmation prompt was answered `no` in `qa`/`prod` | Re-run and confirm with `yes` |

### Connection Issues

Ensure your PostgreSQL server allows connections:
```bash
# Check PostgreSQL is running
pg_isready -h localhost

# Test connection
psql -h localhost -U postgres -d postgres
```

### Permission Errors

Ensure the admin user has sufficient privileges:
```sql
-- Check your role
SELECT current_user, session_user;

-- Check privileges
\du
```

## License

MIT License - See LICENSE file for details

## Contributing

Contributions welcome! Please open an issue or PR.
