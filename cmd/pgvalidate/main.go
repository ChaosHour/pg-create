package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ChaosHour/pg-create/pkg/config"
	"github.com/ChaosHour/pg-create/pkg/database"
	"github.com/ChaosHour/pg-create/pkg/validator"
	"github.com/fatih/color"
)

var (
	host     = flag.String("s", "", "PostgreSQL host")
	port     = flag.String("port", "5432", "PostgreSQL port")
	admin    = flag.String("u", "", "Admin user for inspection queries")
	password = flag.String("p", "", "Admin password (optional when ~/.pgpass matches)")
	dbName   = flag.String("db", "", "Database to connect to for validation. If omitted, all non-template databases are scanned.")
	roles    = flag.String("roles", "", "Comma-separated target roles/users to validate")
	schema   = flag.String("schema", "", "Optional schema filter; empty = all non-system schemas")
	help     = flag.Bool("h", false, "Print help")
)

func init() {
	flag.Parse()
}

func main() {
	if *help {
		printUsage()
		return
	}

	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	if *host == "" || *admin == "" || *roles == "" {
		fmt.Println(red("Missing required flags: -s (host), -u (admin user), -roles (target roles)"))
		printUsage()
		os.Exit(1)
	}

	passwd := *password
	if passwd == "" {
		// Use -db as lookup key, but if unset then default to postgres for pgpass lookup
		lookupDB := *dbName
		if lookupDB == "" {
			lookupDB = "postgres"
		}
		passwd = config.LookupPgPass(*host, *port, lookupDB, *admin)
		if passwd == "" {
			fmt.Println(red("No password provided: use -p flag or add matching entry to ~/.pgpass"))
			os.Exit(1)
		}
		fmt.Println(yellow("Using credentials from ~/.pgpass"))
	}

	baseDB := *dbName
	if baseDB == "" {
		baseDB = "postgres"
	}

	bootstrap, err := database.Connect(*host, *port, *admin, passwd, baseDB)
	if err != nil {
		fmt.Println("Failed to connect to base database:", err)
		os.Exit(1)
	}
	defer bootstrap.Close()

	// Default: validate the postgres maintenance database unless -db is given.
	if *dbName == "" {
		*dbName = "postgres"
	}
	targetDatabases := []string{*dbName}

	opts := validator.Options{
		Roles:  parseCSV(*roles),
		Schema: strings.TrimSpace(*schema),
	}

	blue := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	housekeepingErr := false
	for _, dbn := range targetDatabases {
		fmt.Printf("\n%s\n", blue("==========================================="))
		fmt.Printf("%s %s\n", green("Validating database:"), blue(dbn))
		fmt.Printf("%s\n", blue("==========================================="))

		db, err := database.Connect(*host, *port, *admin, passwd, dbn)
		if err != nil {
			fmt.Printf("%s %s: %v\n", red("Skipping"), dbn, err)
			housekeepingErr = true
			continue
		}

		if err := validator.Run(db, opts); err != nil {
			fmt.Printf("Validation failed for %s: %v\n", dbn, err)
			housekeepingErr = true
		}
		db.Close()
	}

	if housekeepingErr {
		os.Exit(1)
	}
}

func listDatabases(dbConn *sql.DB) ([]string, error) {
	const q = `
SELECT datname
FROM pg_database
WHERE datistemplate = false
  AND datallowconn
ORDER BY datname`
	rows, err := dbConn.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dbs []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		dbs = append(dbs, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return dbs, nil
}

func parseCSV(input string) []string {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func printUsage() {
	fmt.Println("pg-validate: PostgreSQL role/grant validation CLI")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Single role")
	fmt.Println("  pg-validate -s localhost -u postgres -db myapp_prod -roles myapp_ro")
	fmt.Println()
	fmt.Println("  # Multiple roles with schema filter")
	fmt.Println("  pg-validate -s localhost -u postgres -db myapp_prod -roles myapp_app,myapp_ro,myapp_dba -schema myapp")
	fmt.Println()
	fmt.Println("Flags:")
	flag.PrintDefaults()
}
