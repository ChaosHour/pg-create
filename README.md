# pg-create
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