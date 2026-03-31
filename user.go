package main

import (
	"fmt"
	"strings"
)

// Role represents a PostgreSQL role with its configuration
type Role struct {
	Name       string
	Password   string
	Type       string // app, ro, dba
	ConnLimit  int
}

func createRoles() error {
	roleList := parseCSV(*roles)
	
	for _, roleSpec := range roleList {
		role, err := parseRoleSpec(roleSpec)
		if err != nil {
			return err
		}
		
		if err := createSingleRole(role); err != nil {
			return err
		}
	}
	return nil
}

func parseRoleSpec(spec string) (*Role, error) {
	parts := strings.Split(spec, ":")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid role spec %s (expected format: name:password:type)", spec)
	}
	
	role := &Role{
		Name:      strings.TrimSpace(parts[0]),
		Password:  strings.TrimSpace(parts[1]),
		Type:      "app",
		ConnLimit: -1, // unlimited
	}
	
	if len(parts) >= 3 {
		role.Type = strings.ToLower(strings.TrimSpace(parts[2]))
	}
	
	// Set connection limits based on type
	switch role.Type {
	case "dba", "ro":
		role.ConnLimit = 10
	case "app":
		role.ConnLimit = -1 // unlimited for app users
	}
	
	return role, nil
}

func createSingleRole(role *Role) error {
	// Check if role exists
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = $1)"
	if err := db.QueryRow(query, role.Name).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check role %s: %w", role.Name, err)
	}

	if exists {
		fmt.Println(yellow("→"), "Role", role.Name, "already exists")
		return nil
	}

	// Create role with LOGIN and PASSWORD
	createQuery := fmt.Sprintf("CREATE ROLE %s LOGIN PASSWORD $1", quoteIdentifier(role.Name))
	if role.ConnLimit >= 0 {
		createQuery += fmt.Sprintf(" CONNECTION LIMIT %d", role.ConnLimit)
	}
	
	if _, err := db.Exec(createQuery, role.Password); err != nil {
		return fmt.Errorf("failed to create role %s: %w", role.Name, err)
	}

	fmt.Println(green("✓"), "Role", role.Name, fmt.Sprintf("(%s)", role.Type), "created")
	return nil
}

func applyGrants() error {
	roleList := parseCSV(*roles)
	schemaList := parseCSV(*schemas)
	grantList := parseCSV(*grants)
	
	for _, roleSpec := range roleList {
		role, err := parseRoleSpec(roleSpec)
		if err != nil {
			return err
		}
		
		for _, schema := range schemaList {
			if err := applyGrantsToRole(role, schema, grantList); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyGrantsToRole(role *Role, schema string, grantList []string) error {
	for _, grantType := range grantList {
		switch strings.ToLower(strings.TrimSpace(grantType)) {
		case "usage":
			query := fmt.Sprintf("GRANT USAGE ON SCHEMA %s TO %s", 
				quoteIdentifier(schema), quoteIdentifier(role.Name))
			if _, err := db.Exec(query); err != nil {
				return fmt.Errorf("failed to grant USAGE on schema %s to %s: %w", schema, role.Name, err)
			}
			fmt.Println(green("✓"), "Granted USAGE on schema", schema, "to", role.Name)
			
		case "select":
			query := fmt.Sprintf("GRANT SELECT ON ALL TABLES IN SCHEMA %s TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name))
			if _, err := db.Exec(query); err != nil {
				return fmt.Errorf("failed to grant SELECT to %s: %w", role.Name, err)
			}
			
			// Also grant SELECT on sequences
			seqQuery := fmt.Sprintf("GRANT SELECT ON ALL SEQUENCES IN SCHEMA %s TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name))
			if _, err := db.Exec(seqQuery); err != nil {
				return fmt.Errorf("failed to grant SELECT on sequences to %s: %w", role.Name, err)
			}
			fmt.Println(green("✓"), "Granted SELECT on schema", schema, "to", role.Name)
			
		case "insert":
			query := fmt.Sprintf("GRANT INSERT ON ALL TABLES IN SCHEMA %s TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name))
			if _, err := db.Exec(query); err != nil {
				return fmt.Errorf("failed to grant INSERT to %s: %w", role.Name, err)
			}
			fmt.Println(green("✓"), "Granted INSERT on schema", schema, "to", role.Name)
			
		case "update":
			query := fmt.Sprintf("GRANT UPDATE ON ALL TABLES IN SCHEMA %s TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name))
			if _, err := db.Exec(query); err != nil {
				return fmt.Errorf("failed to grant UPDATE to %s: %w", role.Name, err)
			}
			
			// Also grant UPDATE on sequences (for nextval)
			seqQuery := fmt.Sprintf("GRANT UPDATE, USAGE ON ALL SEQUENCES IN SCHEMA %s TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name))
			if _, err := db.Exec(seqQuery); err != nil {
				return fmt.Errorf("failed to grant UPDATE on sequences to %s: %w", role.Name, err)
			}
			fmt.Println(green("✓"), "Granted UPDATE on schema", schema, "to", role.Name)
			
		case "delete":
			query := fmt.Sprintf("GRANT DELETE ON ALL TABLES IN SCHEMA %s TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name))
			if _, err := db.Exec(query); err != nil {
				return fmt.Errorf("failed to grant DELETE to %s: %w", role.Name, err)
			}
			fmt.Println(green("✓"), "Granted DELETE on schema", schema, "to", role.Name)
			
		case "execute":
			query := fmt.Sprintf("GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA %s TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name))
			if _, err := db.Exec(query); err != nil {
				return fmt.Errorf("failed to grant EXECUTE to %s: %w", role.Name, err)
			}
			fmt.Println(green("✓"), "Granted EXECUTE on schema", schema, "to", role.Name)
			
		default:
			return fmt.Errorf("invalid grant type: %s", grantType)
		}
	}
	return nil
}

func setSearchPaths() error {
	roleList := parseCSV(*roles)
	
	for _, roleSpec := range roleList {
		role, err := parseRoleSpec(roleSpec)
		if err != nil {
			return err
		}
		
		if err := setRoleSearchPath(role.Name); err != nil {
			return err
		}
	}
	return nil
}

func setRoleSearchPath(roleName string) error {
	// Set search_path for the role
	query := fmt.Sprintf("ALTER ROLE %s SET search_path = %s",
		quoteIdentifier(roleName), *searchPath)
	
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to set search_path for %s: %w", roleName, err)
	}
	
	fmt.Println(green("✓"), "Set search_path for", roleName, "to", *searchPath)
	return nil
}

func applyDefaultPrivileges() error {
	roleList := parseCSV(*roles)
	schemaList := parseCSV(*schemas)
	
	for _, roleSpec := range roleList {
		role, err := parseRoleSpec(roleSpec)
		if err != nil {
			return err
		}
		
		for _, schema := range schemaList {
			if err := applyDefaultPrivilegesForRole(role, schema); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyDefaultPrivilegesForRole(role *Role, schema string) error {
	var queries []string
	
	switch role.Type {
	case "app":
		// App roles get full CRUD on tables and sequences
		queries = []string{
			fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name)),
			fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT SELECT, UPDATE, USAGE ON SEQUENCES TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name)),
		}
		
	case "ro":
		// Read-only roles get SELECT only
		queries = []string{
			fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT SELECT ON TABLES TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name)),
			fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT SELECT ON SEQUENCES TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name)),
		}
		
	case "dba":
		// DBA roles get full privileges
		queries = []string{
			fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT ALL ON TABLES TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name)),
			fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT ALL ON SEQUENCES TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name)),
			fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT ALL ON FUNCTIONS TO %s",
				quoteIdentifier(schema), quoteIdentifier(role.Name)),
		}
	}
	
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to set default privileges for %s in schema %s: %w", role.Name, schema, err)
		}
	}
	
	fmt.Println(green("✓"), "Default privileges set for", role.Name, fmt.Sprintf("(%s)", role.Type), "in schema", schema)
	return nil
}
