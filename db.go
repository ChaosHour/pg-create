package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

// global variables
var (
	db *sql.DB
)

func initDB(host, port, user, pass, database string) error {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, pass, database)

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}

	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

func createDatabase() error {
	// Check if database exists
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)"
	if err := db.QueryRow(query, *dbName).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if exists {
		fmt.Println(yellow("→"), "Database", *dbName, "already exists")
		return nil
	}

	// Create database (cannot use parameterized query for DDL)
	createQuery := fmt.Sprintf("CREATE DATABASE %s", quoteIdentifier(*dbName))
	if _, err := db.Exec(createQuery); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	fmt.Println(green("✓"), "Database", *dbName, "created")
	return nil
}

func createSchemas() error {
	schemaList := parseCSV(*schemas)

	for _, schema := range schemaList {
		if err := createSingleSchema(schema); err != nil {
			return err
		}
	}
	return nil
}

func createSingleSchema(schema string) error {
	// Check if schema exists
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)"
	if err := db.QueryRow(query, schema).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check schema %s: %w", schema, err)
	}

	if exists {
		fmt.Println(yellow("→"), "Schema", schema, "already exists")
		return nil
	}

	// Create schema
	createQuery := fmt.Sprintf("CREATE SCHEMA %s", quoteIdentifier(schema))
	if _, err := db.Exec(createQuery); err != nil {
		return fmt.Errorf("failed to create schema %s: %w", schema, err)
	}

	fmt.Println(green("✓"), "Schema", schema, "created")
	return nil
}

func createExtensions() error {
	extList := parseCSV(*extensions)
	schemaList := parseCSV(*schemas)

	// Default to first schema or no schema specified
	targetSchema := ""
	if len(schemaList) > 0 {
		targetSchema = schemaList[0]
	}

	for _, ext := range extList {
		if err := createSingleExtension(ext, targetSchema); err != nil {
			return err
		}
	}
	return nil
}

func createSingleExtension(extension, schema string) error {
	// Check if extension exists
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = $1)"
	if err := db.QueryRow(query, extension).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check extension %s: %w", extension, err)
	}

	if exists {
		fmt.Println(yellow("→"), "Extension", extension, "already exists")
		return nil
	}

	// Create extension
	var createQuery string
	if schema != "" {
		createQuery = fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s SCHEMA %s",
			quoteIdentifier(extension), quoteIdentifier(schema))
	} else {
		createQuery = fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", quoteIdentifier(extension))
	}

	if _, err := db.Exec(createQuery); err != nil {
		return fmt.Errorf("failed to create extension %s: %w", extension, err)
	}

	fmt.Println(green("✓"), "Extension", extension, "created")
	return nil
}

// Helper to safely quote PostgreSQL identifiers
func quoteIdentifier(name string) string {
	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(name, `"`, `""`))
}

// Helper to parse comma-separated values
func parseCSV(input string) []string {
	if input == "" {
		return []string{}
	}
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
