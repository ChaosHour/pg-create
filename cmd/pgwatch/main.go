package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ChaosHour/pg-create/pkg/config"
	"github.com/ChaosHour/pg-create/pkg/database"
	"github.com/ChaosHour/pg-create/pkg/pgwatch"
)

var (
	host     = flag.String("s", "", "PostgreSQL host")
	port     = flag.String("port", "5432", "PostgreSQL port")
	userName = flag.String("u", "", "Admin user for inspection queries")
	password = flag.String("p", "", "Password for the admin user (optional when ~/.pgpass matches)")
	dbName   = flag.String("db", "postgres", "Database to connect to")
	interval = flag.Int("interval", 0, "Refresh interval in seconds; 0 = run once")
	query    = flag.String("query", pgwatch.DefaultQuery, "Query to execute against pg_stat_activity")
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

	if *host == "" || *userName == "" {
		fmt.Println("Missing required flags: -s (host), -u (user)")
		printUsage()
		os.Exit(1)
	}

	passwd := *password
	if passwd == "" {
		passwd = config.LookupPgPass(*host, *port, *dbName, *userName)
		if passwd == "" {
			fmt.Println("No password provided: use -p flag or add matching entry to ~/.pgpass")
			os.Exit(1)
		}
		fmt.Println("Using credentials from ~/.pgpass")
	}

	db, err := database.Connect(*host, *port, *userName, passwd, *dbName)
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	monitor := pgwatch.New(db, pgwatch.Options{
		Query:    *query,
		Interval: time.Duration(*interval) * time.Second,
	})

	if err := monitor.Run(ctx); err != nil {
		if err == context.Canceled || err == context.DeadlineExceeded {
			fmt.Println("Exiting")
			return
		}
		fmt.Println("Monitor failed:", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("pg-watch: PostgreSQL activity monitor")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  pg-watch -s host -u user [-p password] [-db postgres] [-interval 1] [-query '...']")
	fmt.Println()
	fmt.Println("Flags:")
	flag.PrintDefaults()
}
