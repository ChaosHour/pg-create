package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ChaosHour/pg-create/pkg/config"
	"github.com/ChaosHour/pg-create/pkg/database"
	"github.com/fatih/color"
)

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
	configFile = flag.String("c", "", "Config file (YAML/JSON) for resource definitions")
	dryRun     = flag.Bool("dry-run", false, "Show what would be done without executing")
	help       = flag.Bool("h", false, "Print help")
)

var (
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	blue   = color.New(color.FgBlue).SprintFunc()
)

func init() {
	flag.Parse()
}

func main() {
	if *help {
		printUsage()
		return
	}

	var cfg *config.Config
	var err error

	// Load from config file if provided
	if *configFile != "" {
		cfg, err = config.LoadConfig(*configFile)
		if err != nil {
			fmt.Println(red("✗"), "Failed to load config file:", err)
			os.Exit(1)
		}
	} else {
		// Build config from flags
		if *host == "" || *userName == "" || *password == "" || *dbName == "" {
			fmt.Println(red("✗"), "Missing required flags: -s (host), -u (user), -p (password), -d (database)")
			printUsage()
			os.Exit(1)
		}
		
		cfg = config.FromFlags(*host, *port, *userName, *password, *dbName, 
			*schemas, *roles, *extensions, *grants, *searchPath, *envType)
	}

	if *dryRun {
		fmt.Println(blue("ℹ"), "DRY RUN MODE - No changes will be made")
		cfg.DryRun = true
	}

	// Environment safeguard
	if cfg.Environment == "prod" || cfg.Environment == "qa" {
		if !confirmProduction(cfg.Environment) {
			fmt.Println(yellow("⚠"), "Operation cancelled by user")
			os.Exit(0)
		}
	}

	// Connect to PostgreSQL
	db, err := database.Connect(cfg.Host, cfg.Port, cfg.User, cfg.Password, "postgres")
	if err != nil {
		fmt.Println(red("✗"), "Failed to connect to database:", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println(green("✓"), "Connected to PostgreSQL")

	// Execute provisioning
	provisioner := database.NewProvisioner(db, cfg)
	if err := provisioner.Provision(); err != nil {
		fmt.Println(red("✗"), "Provisioning failed:", err)
		os.Exit(1)
	}

	if *dryRun {
		fmt.Println(blue("ℹ"), "DRY RUN completed - No actual changes were made")
	} else {
		fmt.Println(green("✓"), "All resources provisioned successfully")
	}
}

func printUsage() {
	fmt.Println("pg-create: PostgreSQL resource provisioning CLI")
	fmt.Println("\nUsage:")
	flag.PrintDefaults()
	fmt.Println("\nExamples:")
	fmt.Println("  # Using flags:")
	fmt.Println("  pg-create -s localhost -u postgres -p secret -d myapp_prod \\")
	fmt.Println("    -sc myapp,ext -r myapp_dba:pass1:dba,myapp_app:pass2:app \\")
	fmt.Println("    -e uuid-ossp,pg_trgm -sp myapp,ext -g usage,select,insert,update,delete")
	fmt.Println("\n  # Using config file:")
	fmt.Println("  pg-create -c config.yaml")
	fmt.Println("\n  # Dry run:")
	fmt.Println("  pg-create -c config.yaml --dry-run")
}

func confirmProduction(env string) bool {
	fmt.Printf(yellow("⚠")+" Running in %s environment. Continue? (yes/no): ", env)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "yes" || response == "y"
}
