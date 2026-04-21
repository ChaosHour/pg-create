package pgwatch

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const DefaultQuery = `select usename, state, count(*) from pg_stat_activity where pid <> pg_backend_pid() group by 1,2 order by 1`

type Options struct {
	Query    string
	Interval time.Duration
}

type Monitor struct {
	db   *sql.DB
	opts Options
}

func New(db *sql.DB, opts Options) *Monitor {
	if opts.Query == "" {
		opts.Query = DefaultQuery
	}
	return &Monitor{db: db, opts: opts}
}

func (m *Monitor) Run(ctx context.Context) error {
	if m.opts.Interval < 0 {
		return fmt.Errorf("interval must be 0 or greater")
	}

	ticker := time.NewTicker(m.opts.Interval)
	if m.opts.Interval == 0 {
		ticker.Stop()
	}
	defer ticker.Stop()

	firstRun := true
	for {
		if !firstRun {
			if m.opts.Interval == 0 {
				break
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
			}
		}

		if m.opts.Interval > 0 {
			clearScreen()
			fmt.Printf("Monitoring connections (%s) - refresh every %d second(s)\n\n", time.Now().Format(time.RFC1123), int(m.opts.Interval.Seconds()))
		}

		if err := m.displayConnectionSummary(ctx); err != nil {
			return err
		}

		if m.opts.Interval == 0 {
			break
		}
		firstRun = false
	}
	return nil
}

func (m *Monitor) displayConnectionSummary(ctx context.Context) error {
	rows, err := m.db.QueryContext(ctx, m.opts.Query)
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	data := make([][]string, 0)
	widths := make([]int, len(cols))
	for i, col := range cols {
		widths[i] = len(col)
	}

	for rows.Next() {
		values := make([]interface{}, len(cols))
		pointers := make([]interface{}, len(cols))
		for i := range values {
			pointers[i] = &values[i]
		}

		if err := rows.Scan(pointers...); err != nil {
			return err
		}

		row := make([]string, len(cols))
		for i, val := range values {
			row[i] = formatValue(val)
			if len(row[i]) > widths[i] {
				widths[i] = len(row[i])
			}
		}
		data = append(data, row)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	printTable(cols, data, widths)
	return nil
}

func formatValue(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case []byte:
		return string(v)
	case string:
		return v
	case int64:
		return fmt.Sprintf("%d", v)
	case int:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func printTable(cols []string, data [][]string, widths []int) {
	printRow(cols, widths)
	printSeparator(widths)
	for _, row := range data {
		printRow(row, widths)
	}
}

func printRow(row []string, widths []int) {
	for i, cell := range row {
		fmt.Printf(" %-*s ", widths[i], cell)
		if i < len(row)-1 {
			fmt.Print("|")
		}
	}
	fmt.Println()
}

func printSeparator(widths []int) {
	for i, width := range widths {
		fmt.Print(strings.Repeat("-", width+2))
		if i < len(widths)-1 {
			fmt.Print("+")
		}
	}
	fmt.Println()
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}
