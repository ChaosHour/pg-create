# pg-create

PostgreSQL Resource Provisioning CLI - A safe, idempotent tool for creating and managing PostgreSQL databases, roles, schemas, extensions, and grants.

## Features

- **Idempotent Operations** - Safe to run multiple times
- **Flexible Configuration** - Use CLI flags or YAML/JSON config files
- **Dry Run Mode** - Preview changes before applying
- **Environment Safeguards** - Confirmation prompts for prod/qa
- **Role-based Privileges** - Automatic privilege templates (app, ro, dba)
- **Security** - Parameterized queries, no SQL injection
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
./bin/pg-create -c config.yaml --dry-run
```

## Configuration

### CLI Flags

| Flag | Short | Description | Required |
|------|-------|-------------|----------|
| `--host` | `-s` | PostgreSQL host | Yes |
| `--user` | `-u` | Admin user for connection | Yes |
| `--password` | `-p` | Admin password | Yes |
| `--database` | `-d` | Database name to create | Yes |
| `--port` | | Port (default: 5432) | No |
| `--schema` | `-sc` | Comma-separated schemas | No |
| `--roles` | `-r` | Roles (format: name:pass:type) | No |
| `--extensions` | `-e` | Comma-separated extensions | No |
| `--search-path` | `-sp` | Search path for roles | No |
| `--grants` | `-g` | Comma-separated grants | No |
| `--env` | | Environment (standalone/qa/prod) | No |
| `--config` | `-c` | Config file (YAML/JSON) | No |
| `--dry-run` | | Preview mode | No |
| `--help` | `-h` | Show help | No |

### Role Types

- **dba** - Database administrator (full privileges, 10 connections)
- **app** - Application role (CRUD privileges, unlimited connections)
- **ro** - Read-only role (SELECT only, 10 connections)

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

## Development

### Project Structure

```
pg-create/
├── cmd/
│   └── pgcreate/
│       └── main.go          # Entry point
├── pkg/
│   ├── config/
│   │   └── config.go        # Configuration handling
│   └── database/
│       └── provisioner.go   # Database provisioning logic
├── bin/                     # Build output
├── Makefile                 # Build automation
├── config.example.yaml      # Example YAML config
├── config.example.json      # Example JSON config
└── README.md
```

### Build Commands

```bash
make build      # Build binary
make clean      # Clean artifacts
make deps       # Install dependencies
make test       # Run tests
make install    # Install to GOPATH
make help       # Show all targets
```

## Security Considerations

- Never commit config files with real passwords
- Use `.pgpass` file for credentials when possible
- Always use `--dry-run` first in production
- The CLI will prompt for confirmation in prod/qa environments
- All SQL queries use parameterized statements

## Troubleshooting

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

---

**pg-create**

Create PostgreSQL Users, Grants and Roles 

## !!! WARNING !!!

This is only used currently for testing. 
Do not use in PROD or any environment that you care about. More testing and validation needs to happen before this is ready for PROD.

## Usage

```GO
pg-create -h
Usage of pg-create:
  -d string
        Database name
  -g string
        Comma-separated list of grants to create
  -h    Print help
  -p string
        Password
  -r string
        Comma-separated list of roles to create
  -s string
        Host
  -sc string
        Schema name
  -sp string
        Search path
  -u string
        User
```

Dependencies:

- [docker](https://www.docker.com/)
- `docker run -d --name postq -d -p 5432:5432/tcp -e POSTGRES_PASSWORD=s3cr3t postgres:latest`
- `brew install libpq`

To create a password:

- `pwgen -s -c -n 23 1`

## Examples

The user johny5_ro will be created with the password izEqeKcKMrk45YmeQsgwS1z and
will have the following grants: usage,select on the data schema and will be a member of the data_ro role.

I am running this multiple times to test the idempotency of the script in this example.

```GO
pg-create -s 10.8.0.10 -u johny5_ro -p izEqeKcKMrk45YmeQsgwS1z -g usage,select  -d data -sc data_schema -r data_ro
✓ Connected to database
[*] Role data_ro already exists
[+] User johny5_ro added to role data_ro
[*] User johny5_ro already exists
[*] Schema data_schema already exists
[+] Role data_ro granted USAGE privilege for schema data_schema
[+] Role data_ro granted SELECT privilege for all tables in schema data_schema
[*] Database data already exists
[*] User johny5_ro already has database data
```

## Validations

```bash
pg-create on  main [!?] via 🐹 v1.20.6 
❯ psql -U johny5_ro -h 10.8.0.10  data
Password for user johny5_ro: 
psql (15.3)
Type "help" for help.

data=>


data=> \l
                                                 List of databases
   Name    |   Owner    | Encoding |  Collate   |   Ctype    | ICU Locale | Locale Provider |   Access privileges   
-----------+------------+----------+------------+------------+------------+-----------------+-----------------------
 books     | klarsen_ro | UTF8     | en_US.utf8 | en_US.utf8 |            | libc            | 
 chaos     | klarsen_ro | UTF8     | en_US.utf8 | en_US.utf8 |            | libc            | 
 data      | johny5_ro  | UTF8     | en_US.utf8 | en_US.utf8 |            | libc            | 
 movies    | johny5_wr  | UTF8     | en_US.utf8 | en_US.utf8 |            | libc            | 
 postgres  | postgres   | UTF8     | en_US.utf8 | en_US.utf8 |            | libc            | 
 template0 | postgres   | UTF8     | en_US.utf8 | en_US.utf8 |            | libc            | =c/postgres          +
           |            |          |            |            |            |                 | postgres=CTc/postgres
 template1 | postgres   | UTF8     | en_US.utf8 | en_US.utf8 |            | libc            | =c/postgres          +
           |            |          |            |            |            |                 | postgres=CTc/postgres
 test      | klarsen_ro | UTF8     | en_US.utf8 | en_US.utf8 |            | libc            | 
(8 rows)

data=> \du+
                                           List of roles
 Role name  |                         Attributes                         | Member of | Description 
------------+------------------------------------------------------------+-----------+-------------
 blarsen_ro |                                                            | {}        | 
 blarsen_wr |                                                            | {}        | 
 chaos_wr   |                                                            | {rw_user} | 
 data_ro    |                                                            | {}        | 
 johny5_ro  |                                                            | {data_ro} | 
 johny5_wr  |                                                            | {}        | 
 jojo_ro    |                                                            | {}        | 
 klarsen_ro |                                                            | {}        | 
 login      | Cannot login                                               | {}        | 
 movies_ro  |                                                            | {}        | 
 movies_wr  |                                                            | {}        | 
 postgres   | Superuser, Create role, Create DB, Replication, Bypass RLS | {}        | 
 rw_user    |                                                            | {}        | 


data=> \x
Expanded display is on.
data=> SELECT * FROM pg_roles WHERE rolname = 'johny5_ro';
-[ RECORD 1 ]--+----------
rolname        | johny5_ro
rolsuper       | f
rolinherit     | t
rolcreaterole  | f
rolcreatedb    | f
rolcanlogin    | t
rolreplication | f
rolconnlimit   | -1
rolpassword    | ********
rolvaliduntil  | 
rolbypassrls   | f
rolconfig      | 
oid            | 24593

data=> \x
Expanded display is off.
data=> \q

```
