# pg-watch

`pg-watch` is a small PostgreSQL activity monitor CLI.

It connects to a PostgreSQL database and repeatedly runs a query against `pg_stat_activity`.
The default query shows connection counts grouped by user and session state.

## Build

From the repository root:

```sh
make build-pgwatch
```

This produces:

```sh
./bin/pg-watch
```

## Usage

```sh
./bin/pg-watch -s <host> -u <user> [-p <password>] [-db <database>] [-interval <seconds>] [-query '<sql>']
```

### Example

```sh
./bin/pg-watch -s your-postgres-host.example.com -u admin -db postgres -interval 1
```

## Flags

- `-s` : PostgreSQL host (required)
- `-u` : Admin user (required)
- `-p` : Password for the admin user (optional; uses `~/.pgpass` if omitted)
- `-db` : Database name, default is `postgres`
- `-interval` : Refresh interval in seconds; `0` runs once
- `-query` : Query to execute; defaults to the built-in `pg_stat_activity` summary query
- `-h` : Print help

## Notes

- A password is only required when `~/.pgpass` does not contain a matching entry.
- If `-interval` is set to `1`, the output refreshes every second.
