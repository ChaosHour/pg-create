package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Host        string   `yaml:"host" json:"host"`
	Port        string   `yaml:"port" json:"port"`
	User        string   `yaml:"user" json:"user"`
	Password    string   `yaml:"password" json:"password"`
	Database    string   `yaml:"database" json:"database"`
	Schemas     []string `yaml:"schemas" json:"schemas"`
	Roles       []Role   `yaml:"roles" json:"roles"`
	Extensions  []string `yaml:"extensions" json:"extensions"`
	Grants      []string `yaml:"grants" json:"grants"`
	SearchPath  string   `yaml:"search_path" json:"search_path"`
	Environment string   `yaml:"environment" json:"environment"`
	DryRun      bool     `yaml:"dry_run" json:"dry_run"`
}

type Role struct {
	Name      string `yaml:"name" json:"name"`
	Password  string `yaml:"password" json:"password"`
	Type      string `yaml:"type" json:"type"`
	ConnLimit int    `yaml:"conn_limit" json:"conn_limit"`
}

var validRoleTypes = map[string]struct{}{
	"app": {},
	"ro":  {},
	"dba": {},
}

var validGrantTypes = map[string]struct{}{
	"usage":   {},
	"select":  {},
	"insert":  {},
	"update":  {},
	"delete":  {},
	"execute": {},
}

var validEnvironments = map[string]struct{}{
	"standalone": {},
	"qa":         {},
	"prod":       {},
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &Config{}
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s (use .yaml, .yml, or .json)", ext)
	}

	// Set defaults
	if cfg.Port == "" {
		cfg.Port = "5432"
	}
	if cfg.Environment == "" {
		cfg.Environment = "standalone"
	} else {
		cfg.Environment = strings.ToLower(strings.TrimSpace(cfg.Environment))
	}

	// Set role defaults
	for i := range cfg.Roles {
		cfg.Roles[i].Name = strings.TrimSpace(cfg.Roles[i].Name)
		cfg.Roles[i].Password = strings.TrimSpace(cfg.Roles[i].Password)
		if cfg.Roles[i].Type == "" {
			cfg.Roles[i].Type = "app"
		} else {
			cfg.Roles[i].Type = strings.ToLower(strings.TrimSpace(cfg.Roles[i].Type))
		}
		if cfg.Roles[i].ConnLimit == 0 {
			switch cfg.Roles[i].Type {
			case "dba", "ro":
				cfg.Roles[i].ConnLimit = 10
			case "app":
				cfg.Roles[i].ConnLimit = -1
			}
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func FromFlags(host, port, user, password, database, schemas, roles, extensions, grants, searchPath, environment string) (*Config, error) {
	cfg := &Config{
		Host:        host,
		Port:        port,
		User:        user,
		Password:    password,
		Database:    database,
		Schemas:     parseCSV(schemas),
		Extensions:  parseCSV(extensions),
		Grants:      parseCSV(grants),
		SearchPath:  searchPath,
		Environment: strings.ToLower(strings.TrimSpace(environment)),
	}

	// Parse roles
	roleSpecs := parseCSV(roles)
	cfg.Roles = make([]Role, 0, len(roleSpecs))
	for _, spec := range roleSpecs {
		role, err := parseRoleSpec(spec)
		if err != nil {
			return nil, err
		}
		cfg.Roles = append(cfg.Roles, *role)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func parseRoleSpec(spec string) (*Role, error) {
	parts := strings.Split(spec, ":")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid role spec: %s", spec)
	}

	role := &Role{
		Name:      strings.TrimSpace(parts[0]),
		Password:  strings.TrimSpace(parts[1]),
		Type:      "app",
		ConnLimit: -1,
	}

	if len(parts) >= 3 {
		role.Type = strings.ToLower(strings.TrimSpace(parts[2]))
	}

	if _, ok := validRoleTypes[role.Type]; !ok {
		return nil, fmt.Errorf("invalid role type %q for role %q (must be app, ro, or dba)", role.Type, role.Name)
	}

	switch role.Type {
	case "dba", "ro":
		role.ConnLimit = 10
	case "app":
		role.ConnLimit = -1
	}

	return role, nil
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.Host) == "" {
		return fmt.Errorf("host is required")
	}
	if strings.TrimSpace(c.User) == "" {
		return fmt.Errorf("user is required")
	}
	if strings.TrimSpace(c.Database) == "" {
		return fmt.Errorf("database is required")
	}

	if c.Port == "" {
		c.Port = "5432"
	}

	if c.Environment == "" {
		c.Environment = "standalone"
	}
	if _, ok := validEnvironments[c.Environment]; !ok {
		return fmt.Errorf("invalid environment %q (must be standalone, qa, or prod)", c.Environment)
	}

	for i := range c.Roles {
		role := &c.Roles[i]
		if role.Name == "" {
			return fmt.Errorf("role name cannot be empty")
		}
		if role.Password == "" {
			return fmt.Errorf("password cannot be empty for role %q", role.Name)
		}
		if role.Type == "" {
			role.Type = "app"
		}
		if _, ok := validRoleTypes[role.Type]; !ok {
			return fmt.Errorf("invalid role type %q for role %q (must be app, ro, or dba)", role.Type, role.Name)
		}
	}

	for _, grant := range c.Grants {
		normalized := strings.ToLower(strings.TrimSpace(grant))
		if normalized == "" {
			continue
		}
		if _, ok := validGrantTypes[normalized]; !ok {
			return fmt.Errorf("invalid grant type %q", grant)
		}
	}

	return nil
}

// LookupPgPass reads ~/.pgpass and returns the password matching the given connection parameters.
// File format per line: hostname:port:database:username:password
func LookupPgPass(host, port, database, user string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	data, err := os.ReadFile(filepath.Join(home, ".pgpass"))
	if err != nil {
		return ""
	}

	match := func(pattern, value string) bool {
		return pattern == "*" || pattern == value
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Split into exactly 5 parts; password may contain colons
		parts := strings.SplitN(line, ":", 5)
		if len(parts) != 5 {
			continue
		}
		if match(parts[0], host) && match(parts[1], port) && match(parts[2], database) && match(parts[3], user) {
			return parts[4]
		}
	}
	return ""
}

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
