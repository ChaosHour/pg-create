package pgstress

import (
	"context"
	"fmt"
	_ "github.com/lib/pq"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const summaryInterval = 15 * time.Second

// Options defines the configuration for the pgstress runner.
type Options struct {
	Host           string
	Port           string
	User           string
	Password       string
	Database       string
	SSLMode        string
	Connections    int
	KeepAlive      time.Duration
	RestartCheck   time.Duration
	ConnectTimeout time.Duration
	Duration       time.Duration
	Verbose        bool
}

type Metrics struct {
	Connected    int64
	Active       int64
	Errors       int64
	Restarts     int64
	WorkersReady int64
}

type Runner struct {
	options Options
	dsn     string
	metrics Metrics
}

func NewRunner(opts Options) (*Runner, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	dsn, err := buildDSN(opts.Host, opts.Port, opts.User, opts.Password, opts.Database, opts.SSLMode)
	if err != nil {
		return nil, err
	}

	return &Runner{options: opts, dsn: dsn}, nil
}

func (o Options) Validate() error {
	if strings.TrimSpace(o.Host) == "" {
		return fmt.Errorf("host is required")
	}
	if strings.TrimSpace(o.User) == "" {
		return fmt.Errorf("user is required")
	}
	if strings.TrimSpace(o.Database) == "" {
		return fmt.Errorf("database is required")
	}
	if o.Connections < 1 {
		return fmt.Errorf("connections must be greater than 0")
	}
	if o.KeepAlive <= 0 {
		return fmt.Errorf("keepalive must be greater than 0")
	}
	if o.RestartCheck <= 0 {
		return fmt.Errorf("restart-check must be greater than 0")
	}
	if o.ConnectTimeout <= 0 {
		return fmt.Errorf("connect-timeout must be greater than 0")
	}
	if o.SSLMode == "" {
		o.SSLMode = "disable"
	}
	return nil
}

func (r *Runner) Run(ctx context.Context) error {
	fmt.Printf("Starting pg-stress with %d concurrent connections to %s:%s/%s\n", r.options.Connections, r.options.Host, r.options.Port, r.options.Database)
	fmt.Printf("keepalive=%s restart-check=%s sslmode=%s\n", r.options.KeepAlive, r.options.RestartCheck, r.options.SSLMode)

	var wg sync.WaitGroup
	wg.Add(r.options.Connections)

	for i := 0; i < r.options.Connections; i++ {
		go r.runWorker(ctx, i, &wg)
	}

	ticker := time.NewTicker(summaryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nStopping...")
			wg.Wait()
			r.printSummary()
			return nil
		case <-ticker.C:
			r.printSummary()
		}
	}
}

func (r *Runner) Summary() Metrics {
	return Metrics{
		Connected:    atomic.LoadInt64(&r.metrics.Connected),
		Active:       atomic.LoadInt64(&r.metrics.Active),
		Errors:       atomic.LoadInt64(&r.metrics.Errors),
		Restarts:     atomic.LoadInt64(&r.metrics.Restarts),
		WorkersReady: atomic.LoadInt64(&r.metrics.WorkersReady),
	}
}

func (r *Runner) printSummary() {
	metrics := r.Summary()
	fmt.Printf("summary: connected=%d active=%d errors=%d restarts=%d ready=%d\n",
		metrics.Connected, metrics.Active, metrics.Errors, metrics.Restarts, metrics.WorkersReady)
}
