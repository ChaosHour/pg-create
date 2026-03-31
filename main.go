package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	_ "github.com/lib/pq"
)

// Define flags
var (
	host       = flag.String("s", "", "Host")
	userName   = flag.String("u", "", "User (admin user for connection)")
	password   = flag.String("p", "", "Password (admin password for connection)")
	port       = flag.String("port", "5432", "Port")
	grants     = flag.String("g", "", "Comma-separated list of grants (usage,select,insert,update,delete,execute)")
	dbName     = flag.String("d", "", "Database name to create")
	schemas    = flag.String("sc", "", "Comma-separated list of schemas to create")
	searchPath = flag.String("sp", "", "Search path (comma-separated schemas)")
	roles      = flag.String("r", "", "Comma-separated list of roles to create (format: rolename:password:type where type is app|ro|dba)")
	extensions = flag.String("e", "", "Comma-separated list of extensions to create")
	envType    = flag.String("env", "standalone", "Environment type (prod, qa, standalone)")
	help       = flag.Bool("h", false, "Print help")
)

// define colors
var green = color.New(color.FgGreen).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()
var yellow = color.New(color.FgYellow).SprintFunc()
var blue = color.New(color.FgBlue).SprintFunc()

// parse flags
func init() {
	flag.Parse()
}

func main() {
	// check if help flag is set
	if *help {
		printUsage()
		return
	}
	
	// validate required flags
	if *host == "" || *userName == "" || *password == "" || *dbName == "" {
		fmt.Println(red("✗"), "Missing required flags: -s (host), -u (user), -p (password), -d (database)")
		printUsage()
		os.Exit(1)
	}

	// environment safeguard
	if *envType == "prod" || *envType == "qa" {
		if !confirmProduction() {
			fmt.Println(yellow("⚠"), "Operation cancelled by user")
			os.Exit(0)
		}
	}

	// connect to PostgreSQL
	if err := initDB(*host, *port, *userName, *password, "postgres"); err != nil {
		fmt.Println(red("✗"), "Failed to connect to database:", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println(green("✓"), "Connected to PostgreSQL")

	// Execute provisioning in correct order
	if err := provisionResources(); err != nil {
		fmt.Println(red("✗"), "Provisioning failed:", err)
		os.Exit(1)
	}

	fmt.Println(green("✓"), "All resources provisioned successfully")
}

func printUsage() {
	fmt.Println("pg-create: PostgreSQL resource provisioning CLI")
	fmt.Println("\nUsage:")
	flag.PrintDefaults()
	fmt.Println("\nExample:")
	fmt.Println("  pg-create -s localhost -u postgres -p secret -d myapp_prod -sc myapp,ext -r myapp_dba:pass1:dba,myapp_app:pass2:app -e uuid-ossp,pg_trgm -sp myapp,ext")
}

func confirmProduction() bool {
	fmt.Printf(yellow("⚠")+" Running in %s environment. Continue? (yes/no): ", *envType)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "yes" || response == "y"
}

func provisionResources() error {
	// 1. Create database
	if err := createDatabase(); err != nil {
		return fmt.Errorf("database creation failed: %w", err)
	}

	// 2. Create roles/users
	if *roles != "" {
		if err := createRoles(); err != nil {
			return fmt.Errorf("role creation failed: %w", err)
		}
	}

	// 3. Create schemas
	if *schemas != "" {
		if err := createSchemas(); err != nil {
			return fmt.Errorf("schema creation failed: %w", err)
		}
	}

	// 4. Create extensions
	if *extensions != "" {
		if err := createExtensions(); err != nil {
			return fmt.Errorf("extension creation failed: %w", err)
		}
	}

	// 5. Apply grants
	if *grants != "" && *roles != "" && *schemas != "" {
		if err := applyGrants(); err != nil {
			return fmt.Errorf("grant application failed: %w", err)
		}
	}

	// 6. Set search paths
	if *searchPath != "" && *roles != "" {
		if err := setSearchPaths(); err != nil {
			return fmt.Errorf("search path configuration failed: %w", err)
		}
	}

	// 7. Apply default privileges
	if *roles != "" && *schemas != "" {
		if err := applyDefaultPrivileges(); err != nil {
			return fmt.Errorf("default privileges failed: %w", err)
		}
	}

	return nil
}
