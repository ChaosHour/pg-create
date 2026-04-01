package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ChaosHour/pg-create/pkg/config"
	"github.com/ChaosHour/pg-create/pkg/database"
	"github.com/ChaosHour/pg-create/pkg/validator"
)

var (
	host     = flag.String("s", "", "PostgreSQL host")
	port     = flag.String("port", "5432", "PostgreSQL port")
	admin    = flag.String("u", "", "Admin user for inspection queries")
	password = flag.String("p", "", "Admin password (optional when ~/.pgpass matches)")
	dbName   = flag.String("db", "postgres", "Database to connect to for validation")
	roles    = flag.String("roles", "", "Comma-separated target roles/users to validate")
	schema   = flag.String("schema", "", "Optional schema filter")
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

	if *host == "" || *admin == "" || *roles == "" {
		fmt.Println("Missing required flags: -s (host), -u (admin user), -roles (target roles)")
		printUsage()
		os.Exit(1)
	}

	passwd := *password
	if passwd == "" {
		passwd = config.LookupPgPass(*host, *port, *dbName, *admin)
		if passwd == "" {
			fmt.Println("No password provided: use -p flag or add matching entry to ~/.pgpass")
			os.Exit(1)
		}
		fmt.Println("Using credentials from ~/.pgpass")
	}

	db, err := database.Connect(*host, *port, *admin, passwd, *dbName)
	if err != nil {
		fmt.Println("Failed to connect:", err)
		os.Exit(1)
	}
	defer db.Close()

	opts := validator.Options{
		Roles:  parseCSV(*roles),
		Schema: strings.TrimSpace(*schema),
	}

	if err := validator.Run(db, opts); err != nil {
		fmt.Println("Validation failed:", err)
		os.Exit(1)
	}
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
