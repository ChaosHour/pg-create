package database

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/ChaosHour/pg-create/pkg/config"
	"github.com/fatih/color"
	_ "github.com/lib/pq"
)

var (
	green  = color.New(color.FgGreen).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	blue   = color.New(color.FgBlue).SprintFunc()
)

type Provisioner struct {
	db  *sql.DB
	cfg *config.Config
}

func Connect(host, port, user, pass, database string) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, pass, database)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func NewProvisioner(db *sql.DB, cfg *config.Config) *Provisioner {
	return &Provisioner{db: db, cfg: cfg}
}

func (p *Provisioner) Provision() error {
	// Phase 1: global operations that must run against the maintenance DB (pg_database, pg_roles are cluster-wide).
	if err := p.createDatabase(); err != nil {
		return fmt.Errorf("database creation failed: %w", err)
	}

	if len(p.cfg.Roles) > 0 {
		if err := p.createRoles(); err != nil {
			return fmt.Errorf("role creation failed: %w", err)
		}
		if err := p.grantConnect(); err != nil {
			return fmt.Errorf("grant connect failed: %w", err)
		}
	}

	// Phase 2: switch to the target database so that schema checks, grants, and
	// default privileges all operate in the correct database context.
	targetDB, err := Connect(p.cfg.Host, p.cfg.Port, p.cfg.User, p.cfg.Password, p.cfg.Database)
	if err != nil {
		if p.cfg.DryRun {
			fmt.Println(blue("i"), "[DRY RUN] Target database not yet available; remaining steps shown for reference only")
		} else {
			return fmt.Errorf("failed to connect to database %s: %w", p.cfg.Database, err)
		}
	} else {
		defer targetDB.Close()
		p.db = targetDB
		fmt.Println(green("✓"), "Switched connection to database", p.cfg.Database)
	}

	if len(p.cfg.Schemas) > 0 {
		if err := p.createSchemas(); err != nil {
			return fmt.Errorf("schema creation failed: %w", err)
		}
	}

	if len(p.cfg.Extensions) > 0 {
		if err := p.createExtensions(); err != nil {
			return fmt.Errorf("extension creation failed: %w", err)
		}
	}

	// Always apply grants and privileges when roles + schemas are present,
	// regardless of whether the database or schema already existed.
	if len(p.cfg.Roles) > 0 && len(p.cfg.Schemas) > 0 {
		if err := p.applyGrants(); err != nil {
			return fmt.Errorf("grant application failed: %w", err)
		}
	}

	if p.cfg.SearchPath != "" && len(p.cfg.Roles) > 0 {
		if err := p.setSearchPaths(); err != nil {
			return fmt.Errorf("search path configuration failed: %w", err)
		}
	}

	if len(p.cfg.Roles) > 0 && len(p.cfg.Schemas) > 0 {
		if err := p.applyDefaultPrivileges(); err != nil {
			return fmt.Errorf("default privileges failed: %w", err)
		}
	}

	return nil
}

func (p *Provisioner) createDatabase() error {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)"
	if err := p.db.QueryRow(query, p.cfg.Database).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if exists {
		fmt.Println(yellow("→"), "Database", p.cfg.Database, "already exists")
		return nil
	}

	if p.cfg.DryRun {
		fmt.Println(blue("ℹ"), "[DRY RUN] Would create database", p.cfg.Database)
		return nil
	}

	createQuery := fmt.Sprintf("CREATE DATABASE %s", quoteIdentifier(p.cfg.Database))
	if _, err := p.db.Exec(createQuery); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	fmt.Println(green("✓"), "Database", p.cfg.Database, "created")
	return nil
}

func (p *Provisioner) createSchemas() error {
	for _, schema := range p.cfg.Schemas {
		if err := p.createSingleSchema(schema); err != nil {
			return err
		}
	}
	return nil
}

func (p *Provisioner) createSingleSchema(schema string) error {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)"
	if err := p.db.QueryRow(query, schema).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check schema %s: %w", schema, err)
	}

	if exists {
		fmt.Println(yellow("→"), "Schema", schema, "already exists")
		return nil
	}

	if p.cfg.DryRun {
		fmt.Println(blue("ℹ"), "[DRY RUN] Would create schema", schema)
		return nil
	}

	createQuery := fmt.Sprintf("CREATE SCHEMA %s", quoteIdentifier(schema))
	if _, err := p.db.Exec(createQuery); err != nil {
		return fmt.Errorf("failed to create schema %s: %w", schema, err)
	}

	fmt.Println(green("✓"), "Schema", schema, "created")
	return nil
}

func (p *Provisioner) createExtensions() error {
	targetSchema := ""
	if len(p.cfg.Schemas) > 0 {
		targetSchema = p.cfg.Schemas[0]
	}

	for _, ext := range p.cfg.Extensions {
		if err := p.createSingleExtension(ext, targetSchema); err != nil {
			return err
		}
	}
	return nil
}

func (p *Provisioner) createSingleExtension(extension, schema string) error {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = $1)"
	if err := p.db.QueryRow(query, extension).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check extension %s: %w", extension, err)
	}

	if exists {
		fmt.Println(yellow("→"), "Extension", extension, "already exists")
		return nil
	}

	if p.cfg.DryRun {
		fmt.Println(blue("ℹ"), "[DRY RUN] Would create extension", extension, "in schema", schema)
		return nil
	}

	var createQuery string
	if schema != "" {
		createQuery = fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s SCHEMA %s",
			quoteIdentifier(extension), quoteIdentifier(schema))
	} else {
		createQuery = fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", quoteIdentifier(extension))
	}

	if _, err := p.db.Exec(createQuery); err != nil {
		return fmt.Errorf("failed to create extension %s: %w", extension, err)
	}

	fmt.Println(green("✓"), "Extension", extension, "created")
	return nil
}

func (p *Provisioner) createRoles() error {
	for _, role := range p.cfg.Roles {
		if err := p.createSingleRole(role); err != nil {
			return err
		}
	}
	return nil
}

func (p *Provisioner) createSingleRole(role config.Role) error {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = $1)"
	if err := p.db.QueryRow(query, role.Name).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check role %s: %w", role.Name, err)
	}

	if exists {
		fmt.Println(yellow("→"), "Role", role.Name, "already exists")
		return nil
	}

	if p.cfg.DryRun {
		fmt.Println(blue("ℹ"), "[DRY RUN] Would create role", role.Name, fmt.Sprintf("(%s)", role.Type))
		return nil
	}

	// DDL statements don't support parameterized queries; escape single quotes manually
	escapedPassword := strings.ReplaceAll(role.Password, "'", "''")
	createQuery := fmt.Sprintf("CREATE ROLE %s LOGIN PASSWORD '%s'", quoteIdentifier(role.Name), escapedPassword)
	if role.ConnLimit >= 0 {
		createQuery += fmt.Sprintf(" CONNECTION LIMIT %d", role.ConnLimit)
	}

	if _, err := p.db.Exec(createQuery); err != nil {
		return fmt.Errorf("failed to create role %s: %w", role.Name, err)
	}

	fmt.Println(green("✓"), "Role", role.Name, fmt.Sprintf("(%s)", role.Type), "created")
	return nil
}

func (p *Provisioner) grantConnect() error {
	for _, role := range p.cfg.Roles {
		if p.cfg.DryRun {
			fmt.Println(blue("i"), "[DRY RUN] Would grant CONNECT on database", p.cfg.Database, "to", role.Name)
			continue
		}
		query := fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s",
			quoteIdentifier(p.cfg.Database), quoteIdentifier(role.Name))
		if _, err := p.db.Exec(query); err != nil {
			return fmt.Errorf("failed to grant CONNECT to %s: %w", role.Name, err)
		}
		fmt.Println(green("✓"), "Granted CONNECT on database", p.cfg.Database, "to", role.Name)
	}
	return nil
}

func (p *Provisioner) applyGrants() error {
	for _, role := range p.cfg.Roles {
		for _, schema := range p.cfg.Schemas {
			if err := p.applyGrantsToRole(role, schema); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Provisioner) applyGrantsToRole(role config.Role, schema string) error {
	for _, grantType := range p.cfg.Grants {
		if p.cfg.DryRun {
			fmt.Println(blue("ℹ"), "[DRY RUN] Would grant", grantType, "on schema", schema, "to", role.Name)
			continue
		}

		switch strings.ToLower(strings.TrimSpace(grantType)) {
		case "usage":
			query := fmt.Sprintf("GRANT USAGE ON SCHEMA %s TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name))
			if _, err := p.db.Exec(query); err != nil {
				return fmt.Errorf("failed to grant USAGE: %w", err)
			}
			fmt.Println(green("✓"), "Granted USAGE on schema", schema, "to", role.Name)

		case "select":
			query := fmt.Sprintf("GRANT SELECT ON ALL TABLES IN SCHEMA %s TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name))
			if _, err := p.db.Exec(query); err != nil {
				return fmt.Errorf("failed to grant SELECT: %w", err)
			}

			seqQuery := fmt.Sprintf("GRANT SELECT ON ALL SEQUENCES IN SCHEMA %s TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name))
			if _, err := p.db.Exec(seqQuery); err != nil {
				return fmt.Errorf("failed to grant SELECT on sequences: %w", err)
			}
			fmt.Println(green("✓"), "Granted SELECT on schema", schema, "to", role.Name)

		case "insert":
			query := fmt.Sprintf("GRANT INSERT ON ALL TABLES IN SCHEMA %s TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name))
			if _, err := p.db.Exec(query); err != nil {
				return fmt.Errorf("failed to grant INSERT: %w", err)
			}
			fmt.Println(green("✓"), "Granted INSERT on schema", schema, "to", role.Name)

		case "update":
			query := fmt.Sprintf("GRANT UPDATE ON ALL TABLES IN SCHEMA %s TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name))
			if _, err := p.db.Exec(query); err != nil {
				return fmt.Errorf("failed to grant UPDATE: %w", err)
			}

			seqQuery := fmt.Sprintf("GRANT UPDATE, USAGE ON ALL SEQUENCES IN SCHEMA %s TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name))
			if _, err := p.db.Exec(seqQuery); err != nil {
				return fmt.Errorf("failed to grant UPDATE on sequences: %w", err)
			}
			fmt.Println(green("✓"), "Granted UPDATE on schema", schema, "to", role.Name)

		case "delete":
			query := fmt.Sprintf("GRANT DELETE ON ALL TABLES IN SCHEMA %s TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name))
			if _, err := p.db.Exec(query); err != nil {
				return fmt.Errorf("failed to grant DELETE: %w", err)
			}
			fmt.Println(green("✓"), "Granted DELETE on schema", schema, "to", role.Name)

		case "execute":
			query := fmt.Sprintf("GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA %s TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name))
			if _, err := p.db.Exec(query); err != nil {
				return fmt.Errorf("failed to grant EXECUTE: %w", err)
			}
			fmt.Println(green("✓"), "Granted EXECUTE on schema", schema, "to", role.Name)

		default:
			return fmt.Errorf("invalid grant type: %s", grantType)
		}
	}
	return nil
}

func (p *Provisioner) setSearchPaths() error {
	for _, role := range p.cfg.Roles {
		if err := p.setRoleSearchPath(role.Name); err != nil {
			return err
		}
	}
	return nil
}

func (p *Provisioner) setRoleSearchPath(roleName string) error {
	if p.cfg.DryRun {
		fmt.Println(blue("ℹ"), "[DRY RUN] Would set search_path for", roleName, "to", p.cfg.SearchPath)
		return nil
	}

	query := fmt.Sprintf("ALTER ROLE %s SET search_path = %s",
		quoteIdentifier(roleName), p.cfg.SearchPath)

	if _, err := p.db.Exec(query); err != nil {
		return fmt.Errorf("failed to set search_path for %s: %w", roleName, err)
	}

	fmt.Println(green("✓"), "Set search_path for", roleName, "to", p.cfg.SearchPath)
	return nil
}

func (p *Provisioner) applyDefaultPrivileges() error {
	for _, role := range p.cfg.Roles {
		for _, schema := range p.cfg.Schemas {
			if err := p.applyDefaultPrivilegesForRole(role, schema); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Provisioner) applyDefaultPrivilegesForRole(role config.Role, schema string) error {
	var queries []string

	switch role.Type {
	case "app":
		queries = []string{
			fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name)),
			fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT SELECT, UPDATE, USAGE ON SEQUENCES TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name)),
		}

	case "ro":
		queries = []string{
			fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT SELECT ON TABLES TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name)),
			fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT SELECT ON SEQUENCES TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name)),
		}

	case "dba":
		queries = []string{
			fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT ALL ON TABLES TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name)),
			fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT ALL ON SEQUENCES TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name)),
			fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT ALL ON FUNCTIONS TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name)),
		}
	}

	if p.cfg.DryRun {
		fmt.Println(blue("ℹ"), "[DRY RUN] Would set default privileges for", role.Name, fmt.Sprintf("(%s)", role.Type), "in schema", schema)
		return nil
	}

	for _, query := range queries {
		if _, err := p.db.Exec(query); err != nil {
			return fmt.Errorf("failed to set default privileges: %w", err)
		}
	}

	fmt.Println(green("✓"), "Default privileges set for", role.Name, fmt.Sprintf("(%s)", role.Type), "in schema", schema)
	return nil
}

func quoteIdentifier(name string) string {
	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(name, `"`, `""`))
}
