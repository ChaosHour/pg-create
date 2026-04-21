# pgstress

`pgstress` is a small CLI for opening many concurrent PostgreSQL connections and keeping them alive.
It is useful for testing how a live PostgreSQL instance behaves under sustained connection pressure, including observing whether the database server restarts when Cloud SQL instance storage is increased.

## Build

From the repository root:

```bash
go build ./cmd/pgstress
```

## Run

```bash
./cmd/pgstress -host <host> -port <port> -user <user> -db <database> -connections 200
```

## Flags

- `-host` : PostgreSQL host (required)
- `-port` : PostgreSQL port (default: `5432`)
- `-user` : PostgreSQL user (required)
- `-password` : Password (optional when `~/.pgpass` matches)
- `-db` : Database name (default: `postgres`)
- `-sslmode` : SSL mode (default: `disable`)
- `-connections` : Number of concurrent connections to open (required)
- `-keepalive` : Interval between `SELECT 1` keepalive queries (default: `15s`)
- `-restart-check` : Interval for checking `pg_postmaster_start_time()` (default: `30s`)
- `-duration` : Run duration; use `0` to run until interrupted (default: `0`)
- `-connect-timeout` : Timeout for initial connection attempts (default: `10s`)
- `-verbose` : Print worker-level events
- `-h` : Print help

## Example

```bash
./cmd/pgstress -host mycloudsql.example.com -port 5432 -user postgres -db postgres -connections 250 -keepalive 15s -restart-check 30s
```

## Notes

- If `-password` is omitted, `pgstress` will attempt to read credentials from `~/.pgpass`.
- Restart detection is performed by polling `pg_postmaster_start_time()` on each active connection.
- Use `CTRL+C` to stop the load test and print a final summary.
