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
	"github.com/ChaosHour/pg-create/pkg/pgstress"
	"github.com/fatih/color"
)

var (
	host           = flag.String("host", "", "PostgreSQL host")
	port           = flag.String("port", "5432", "PostgreSQL port")
	userName       = flag.String("user", "", "PostgreSQL user")
	password       = flag.String("password", "", "Password (optional when ~/.pgpass matches)")
	database       = flag.String("db", "postgres", "Database name")
	sslmode        = flag.String("sslmode", "disable", "PostgreSQL sslmode")
	connections    = flag.Int("connections", 100, "Number of concurrent connections to open")
	keepalive      = flag.Duration("keepalive", 15*time.Second, "Keepalive query interval")
	restartCheck   = flag.Duration("restart-check", 30*time.Second, "How often to verify pg_postmaster_start_time()")
	duration       = flag.Duration("duration", 0, "Run duration (0 = until interrupted)")
	verbose        = flag.Bool("verbose", false, "Print detailed worker events")
	connectTimeout = flag.Duration("connect-timeout", 10*time.Second, "Connection timeout for initial connection attempts")
	help           = flag.Bool("h", false, "Print help")
)

func main() {
	flag.Parse()

	if *help {
		printUsage()
		return
	}

	if *host == "" || *userName == "" {
		fmt.Println("Missing required flags: -host and -user")
		printUsage()
		os.Exit(1)
	}

	passwd := *password
	if passwd == "" {
		passwd = config.LookupPgPass(*host, *port, *database, *userName)
		if passwd == "" {
			fmt.Println("No password provided: use -password flag or add a matching entry to ~/.pgpass")
			os.Exit(1)
		}
		fmt.Println(color.YellowString("Using credentials from ~/.pgpass"))
	}

	opts := pgstress.Options{
		Host:           *host,
		Port:           *port,
		User:           *userName,
		Password:       passwd,
		Database:       *database,
		SSLMode:        *sslmode,
		Connections:    *connections,
		KeepAlive:      *keepalive,
		RestartCheck:   *restartCheck,
		ConnectTimeout: *connectTimeout,
		Duration:       *duration,
		Verbose:        *verbose,
	}

	runner, err := pgstress.NewRunner(opts)
	if err != nil {
		fmt.Println("Invalid options:", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *duration > 0 {
		ctx, stop = context.WithTimeout(ctx, *duration)
		defer stop()
	}

	if err := runner.Run(ctx); err != nil {
		fmt.Println("pgstress failed:", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("pg-stress: open many concurrent PostgreSQL connections and keep them active")
	fmt.Println()
	fmt.Println("Usage:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  go run ./cmd/pgstress -host myhost -port 5432 -user postgres -db mydb -connections 200 -keepalive 15s -restart-check 30s")
}
