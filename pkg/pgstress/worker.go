package pgstress

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

func (r *Runner) runWorker(ctx context.Context, id int, wg *sync.WaitGroup) {
	defer wg.Done()
	atomic.AddInt64(&r.metrics.WorkersReady, 1)

	var db *sql.DB
	var conn *sql.Conn
	var lastStartTime time.Time
	var lastRestartCheck time.Time

	retryDelay := 1 * time.Second

	for {
		if ctx.Err() != nil {
			cleanup(db, conn)
			return
		}

		if conn == nil {
			if db != nil {
				_ = db.Close()
				db = nil
			}

			if r.options.Verbose {
				fmt.Printf("worker %d: dialing\n", id)
			}

			var err error
			db, err = sql.Open("postgres", r.dsn)
			if err != nil {
				atomic.AddInt64(&r.metrics.Errors, 1)
				r.logWorker(id, "open failed: %v", err)
				select {
				case <-ctx.Done():
					cleanup(db, conn)
					return
				case <-time.After(retryDelay):
					retryDelay = minDuration(retryDelay*2, 10*time.Second)
				}
				continue
			}

			db.SetMaxOpenConns(1)
			db.SetMaxIdleConns(1)
			db.SetConnMaxIdleTime(0)
			db.SetConnMaxLifetime(0)

			connCtx, cancel := context.WithTimeout(ctx, r.options.ConnectTimeout)
			conn, err = db.Conn(connCtx)
			cancel()
			if err != nil {
				atomic.AddInt64(&r.metrics.Errors, 1)
				r.logWorker(id, "connection failed: %v", err)
				retryDelay = minDuration(retryDelay*2, 10*time.Second)
				select {
				case <-ctx.Done():
					cleanup(db, conn)
					return
				case <-time.After(retryDelay):
					continue
				}
			}

			if err := r.pingConnection(ctx, conn); err != nil {
				atomic.AddInt64(&r.metrics.Errors, 1)
				r.logWorker(id, "ping failed: %v", err)
				_ = conn.Close()
				conn = nil
				retryDelay = minDuration(retryDelay*2, 10*time.Second)
				continue
			}

			atomic.AddInt64(&r.metrics.Connected, 1)
			atomic.AddInt64(&r.metrics.Active, 1)
			retryDelay = 1 * time.Second
			lastRestartCheck = time.Now()

			startTime, err := r.fetchPostmasterStartTime(ctx, conn)
			if err != nil {
				atomic.AddInt64(&r.metrics.Errors, 1)
				r.logWorker(id, "start time read failed: %v", err)
				_ = conn.Close()
				conn = nil
				continue
			}
			lastStartTime = startTime
			r.logWorker(id, "connected with start_time=%s", lastStartTime.Format(time.RFC3339Nano))
		}

		select {
		case <-ctx.Done():
			cleanup(db, conn)
			return
		case <-time.After(r.options.KeepAlive):
			if conn == nil {
				continue
			}

			if err := r.keepAlive(ctx, conn); err != nil {
				atomic.AddInt64(&r.metrics.Errors, 1)
				r.logWorker(id, "keepalive failed: %v", err)
				_ = conn.Close()
				conn = nil
				atomic.AddInt64(&r.metrics.Active, -1)
				continue
			}

			if time.Since(lastRestartCheck) >= r.options.RestartCheck {
				startTime, err := r.fetchPostmasterStartTime(ctx, conn)
				if err != nil {
					atomic.AddInt64(&r.metrics.Errors, 1)
					r.logWorker(id, "restart-check failed: %v", err)
					_ = conn.Close()
					conn = nil
					atomic.AddInt64(&r.metrics.Active, -1)
					continue
				}
				lastRestartCheck = time.Now()
				if !startTime.Equal(lastStartTime) {
					atomic.AddInt64(&r.metrics.Restarts, 1)
					r.logWorker(id, "RESTART detected: old=%s new=%s", lastStartTime.Format(time.RFC3339Nano), startTime.Format(time.RFC3339Nano))
					lastStartTime = startTime
				}
			}
		}
	}
}

func cleanup(db *sql.DB, conn *sql.Conn) {
	if conn != nil {
		_ = conn.Close()
	}
	if db != nil {
		_ = db.Close()
	}
}

func (r *Runner) pingConnection(ctx context.Context, conn *sql.Conn) error {
	_, err := conn.ExecContext(ctx, "SELECT 1")
	return err
}

func (r *Runner) keepAlive(ctx context.Context, conn *sql.Conn) error {
	_, err := conn.ExecContext(ctx, "SELECT 1")
	return err
}

func (r *Runner) fetchPostmasterStartTime(ctx context.Context, conn *sql.Conn) (time.Time, error) {
	rows, err := conn.QueryContext(ctx, "SELECT pg_postmaster_start_time()")
	if err != nil {
		return time.Time{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return time.Time{}, err
		}
		return time.Time{}, sql.ErrNoRows
	}

	var startTime time.Time
	if err := rows.Scan(&startTime); err != nil {
		return time.Time{}, err
	}
	return startTime, nil
}

func (r *Runner) logWorker(id int, format string, args ...interface{}) {
	if !r.options.Verbose {
		return
	}
	fmt.Printf("worker %03d: %s\n", id, fmt.Sprintf(format, args...))
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
