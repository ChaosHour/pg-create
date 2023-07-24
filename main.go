package main

import (
	"flag"

	"github.com/fatih/color"
	_ "github.com/lib/pq"
)

// Define flags
var (
	source   = flag.String("s", "", "Host")
	userName = flag.String("u", "", "User")
	password = flag.String("p", "", "Password")
	//port       = flag.String("port", "", "Port")
	grants     = flag.String("g", "", "Comma-separated list of grants to create")
	dbName     = flag.String("d", "", "Database name")
	schemaName = flag.String("sc", "", "Schema name")
	searchPath = flag.String("sp", "", "Search path")
	roleName   = flag.String("r", "", "Comma-separated list of roles to create")
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
		flag.PrintDefaults()
		return
	}
	// if no flags are set, print help
	if *source == "" && *userName == "" && *password == "" && *grants == "" && *dbName == "" && *schemaName == "" && *searchPath == "" && *roleName == "" {
		flag.PrintDefaults()
		return
	}

	// if just the -h flag is set, print help
	if *help {
		flag.PrintDefaults()
		return
	}

	initDB()
	createRole()
	createUser()
	checkSchemaExists()
	createGrants()
	createSchema()
	grantSchema()
	grantDatabase()

}
