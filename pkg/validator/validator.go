package validator

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/lib/pq"
)

type Options struct {
	Roles  []string
	Schema string
}

type roleOverview struct {
	RequestedRole string
	RoleExists    bool
	CanLogin      sql.NullBool
	ConnLimit     sql.NullInt64
	SuperUser     sql.NullBool
	Inherit       sql.NullBool
	CreateDB      sql.NullBool
	CreateRole    sql.NullBool
	Replication   sql.NullBool
	BypassRLS     sql.NullBool
}

type membership struct {
	RoleName string
	Members  []string
}

type dbPrivilege struct {
	RoleName   string
	CanConnect bool
	CanCreate  bool
	CanTemp    bool
}

type schemaPrivilege struct {
	RoleName   string
	SchemaName string
	CanUsage   bool
	CanCreate  bool
}

type tableSummary struct {
	RoleName     string
	SchemaName   string
	TableCount   int
	CanSelect    int
	CanInsert    int
	CanUpdate    int
	CanDelete    int
	CanTruncate  int
	CanReference int
	CanTrigger   int
}

type sequenceSummary struct {
	RoleName   string
	SchemaName string
	SeqCount   int
	CanSelect  int
	CanUsage   int
	CanUpdate  int
}

type functionSummary struct {
	RoleName   string
	SchemaName string
	FuncCount  int
	CanExecute int
}

type defaultPrivilege struct {
	Grantor    string
	SchemaName string
	ObjectType string
	Grantee    string
	Privilege  string
	Grantable  bool
}

type ownedObjects struct {
	RoleName       string
	OwnedTables    int
	OwnedSequences int
	OwnedViews     int
	OwnedFunctions int
}

func Run(db *sql.DB, opts Options) error {
	roles := normalizeRoles(opts.Roles)
	if len(roles) == 0 {
		return fmt.Errorf("at least one role must be provided")
	}

	schema := strings.TrimSpace(opts.Schema)

	overview, err := fetchRoleOverview(db, roles)
	if err != nil {
		return err
	}
	memberships, err := fetchMemberships(db, roles)
	if err != nil {
		return err
	}
	dbPrivs, err := fetchDatabasePrivileges(db, roles)
	if err != nil {
		return err
	}
	schemaPrivs, err := fetchSchemaPrivileges(db, roles, schema)
	if err != nil {
		return err
	}
	tables, err := fetchTableSummary(db, roles, schema)
	if err != nil {
		return err
	}
	sequences, err := fetchSequenceSummary(db, roles, schema)
	if err != nil {
		return err
	}
	functions, err := fetchFunctionSummary(db, roles, schema)
	if err != nil {
		return err
	}
	defaults, err := fetchDefaultPrivileges(db, roles, schema)
	if err != nil {
		return err
	}
	owned, err := fetchOwnedObjects(db, roles)
	if err != nil {
		return err
	}

	printHeader(roles, schema)
	printRoleOverview(overview)
	printMemberships(memberships)
	printDatabasePrivileges(dbPrivs)
	printSchemaPrivileges(schemaPrivs)
	printTableSummary(tables)
	printSequenceSummary(sequences)
	printFunctionSummary(functions)
	printDefaultPrivileges(defaults)
	printOwnedObjects(owned)

	return nil
}

func normalizeRoles(roles []string) []string {
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(roles))
	for _, role := range roles {
		trimmed := strings.TrimSpace(role)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	sort.Strings(normalized)
	return normalized
}

func fetchRoleOverview(db *sql.DB, roles []string) ([]roleOverview, error) {
	const q = `
SELECT input.role_name,
       r.rolname IS NOT NULL AS role_exists,
       r.rolcanlogin,
       r.rolconnlimit,
       r.rolsuper,
       r.rolinherit,
       r.rolcreatedb,
       r.rolcreaterole,
       r.rolreplication,
       r.rolbypassrls
FROM unnest($1::text[]) AS input(role_name)
LEFT JOIN pg_roles r ON r.rolname = input.role_name
ORDER BY input.role_name`

	rows, err := db.Query(q, pq.Array(roles))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch role overview: %w", err)
	}
	defer rows.Close()

	var result []roleOverview
	for rows.Next() {
		var r roleOverview
		if err := rows.Scan(&r.RequestedRole, &r.RoleExists, &r.CanLogin, &r.ConnLimit, &r.SuperUser, &r.Inherit, &r.CreateDB, &r.CreateRole, &r.Replication, &r.BypassRLS); err != nil {
			return nil, fmt.Errorf("failed scanning role overview: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func fetchMemberships(db *sql.DB, roles []string) ([]membership, error) {
	const q = `
SELECT input.role_name,
       COALESCE(array_agg(parent.rolname ORDER BY parent.rolname)
                FILTER (WHERE parent.rolname IS NOT NULL), ARRAY[]::text[]) AS member_of
FROM unnest($1::text[]) AS input(role_name)
LEFT JOIN pg_roles child ON child.rolname = input.role_name
LEFT JOIN pg_auth_members m ON m.member = child.oid
LEFT JOIN pg_roles parent ON parent.oid = m.roleid
GROUP BY input.role_name
ORDER BY input.role_name`

	rows, err := db.Query(q, pq.Array(roles))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch role memberships: %w", err)
	}
	defer rows.Close()

	var result []membership
	for rows.Next() {
		var m membership
		if err := rows.Scan(&m.RoleName, pq.Array(&m.Members)); err != nil {
			return nil, fmt.Errorf("failed scanning memberships: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func fetchDatabasePrivileges(db *sql.DB, roles []string) ([]dbPrivilege, error) {
	const q = `
SELECT input.role_name,
       has_database_privilege(input.role_name, current_database(), 'CONNECT') AS can_connect,
       has_database_privilege(input.role_name, current_database(), 'CREATE') AS can_create,
       has_database_privilege(input.role_name, current_database(), 'TEMP') AS can_temp
FROM unnest($1::text[]) AS input(role_name)
ORDER BY input.role_name`

	rows, err := db.Query(q, pq.Array(roles))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch database privileges: %w", err)
	}
	defer rows.Close()

	var result []dbPrivilege
	for rows.Next() {
		var p dbPrivilege
		if err := rows.Scan(&p.RoleName, &p.CanConnect, &p.CanCreate, &p.CanTemp); err != nil {
			return nil, fmt.Errorf("failed scanning database privileges: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func fetchSchemaPrivileges(db *sql.DB, roles []string, schema string) ([]schemaPrivilege, error) {
	const q = `
WITH target_schemas AS (
    SELECT n.nspname
    FROM pg_namespace n
    WHERE n.nspname NOT LIKE 'pg_%'
      AND n.nspname <> 'information_schema'
      AND ($2 = '' OR n.nspname = $2)
)
SELECT input.role_name,
       ts.nspname,
       has_schema_privilege(input.role_name, ts.nspname, 'USAGE') AS can_usage,
       has_schema_privilege(input.role_name, ts.nspname, 'CREATE') AS can_create
FROM unnest($1::text[]) AS input(role_name)
CROSS JOIN target_schemas ts
ORDER BY input.role_name, ts.nspname`

	rows, err := db.Query(q, pq.Array(roles), schema)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schema privileges: %w", err)
	}
	defer rows.Close()

	var result []schemaPrivilege
	for rows.Next() {
		var p schemaPrivilege
		if err := rows.Scan(&p.RoleName, &p.SchemaName, &p.CanUsage, &p.CanCreate); err != nil {
			return nil, fmt.Errorf("failed scanning schema privileges: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func fetchTableSummary(db *sql.DB, roles []string, schema string) ([]tableSummary, error) {
	const q = `
WITH target_schemas AS (
    SELECT n.nspname
    FROM pg_namespace n
    WHERE n.nspname NOT LIKE 'pg_%'
      AND n.nspname <> 'information_schema'
      AND ($2 = '' OR n.nspname = $2)
), target_tables AS (
    SELECT t.table_schema, t.table_name
    FROM information_schema.tables t
    JOIN target_schemas s ON s.nspname = t.table_schema
    WHERE t.table_type = 'BASE TABLE'
)
SELECT input.role_name,
       tt.table_schema,
       count(*)::int AS table_count,
       count(*) FILTER (WHERE has_table_privilege(input.role_name, format('%I.%I', tt.table_schema, tt.table_name), 'SELECT'))::int AS can_select,
       count(*) FILTER (WHERE has_table_privilege(input.role_name, format('%I.%I', tt.table_schema, tt.table_name), 'INSERT'))::int AS can_insert,
       count(*) FILTER (WHERE has_table_privilege(input.role_name, format('%I.%I', tt.table_schema, tt.table_name), 'UPDATE'))::int AS can_update,
       count(*) FILTER (WHERE has_table_privilege(input.role_name, format('%I.%I', tt.table_schema, tt.table_name), 'DELETE'))::int AS can_delete,
       count(*) FILTER (WHERE has_table_privilege(input.role_name, format('%I.%I', tt.table_schema, tt.table_name), 'TRUNCATE'))::int AS can_truncate,
       count(*) FILTER (WHERE has_table_privilege(input.role_name, format('%I.%I', tt.table_schema, tt.table_name), 'REFERENCES'))::int AS can_references,
       count(*) FILTER (WHERE has_table_privilege(input.role_name, format('%I.%I', tt.table_schema, tt.table_name), 'TRIGGER'))::int AS can_trigger
FROM unnest($1::text[]) AS input(role_name)
JOIN target_tables tt ON true
GROUP BY input.role_name, tt.table_schema
ORDER BY input.role_name, tt.table_schema`

	rows, err := db.Query(q, pq.Array(roles), schema)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch table privilege summary: %w", err)
	}
	defer rows.Close()

	var result []tableSummary
	for rows.Next() {
		var p tableSummary
		if err := rows.Scan(&p.RoleName, &p.SchemaName, &p.TableCount, &p.CanSelect, &p.CanInsert, &p.CanUpdate, &p.CanDelete, &p.CanTruncate, &p.CanReference, &p.CanTrigger); err != nil {
			return nil, fmt.Errorf("failed scanning table summary: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func fetchSequenceSummary(db *sql.DB, roles []string, schema string) ([]sequenceSummary, error) {
	const q = `
WITH target_schemas AS (
    SELECT n.nspname
    FROM pg_namespace n
    WHERE n.nspname NOT LIKE 'pg_%'
      AND n.nspname <> 'information_schema'
      AND ($2 = '' OR n.nspname = $2)
), target_sequences AS (
    SELECT n.nspname AS schema_name, c.relname AS sequence_name
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    JOIN target_schemas s ON s.nspname = n.nspname
    WHERE c.relkind = 'S'
)
SELECT input.role_name,
       ts.schema_name,
       count(*)::int AS sequence_count,
       count(*) FILTER (WHERE has_sequence_privilege(input.role_name, format('%I.%I', ts.schema_name, ts.sequence_name), 'SELECT'))::int AS can_select,
       count(*) FILTER (WHERE has_sequence_privilege(input.role_name, format('%I.%I', ts.schema_name, ts.sequence_name), 'USAGE'))::int AS can_usage,
       count(*) FILTER (WHERE has_sequence_privilege(input.role_name, format('%I.%I', ts.schema_name, ts.sequence_name), 'UPDATE'))::int AS can_update
FROM unnest($1::text[]) AS input(role_name)
JOIN target_sequences ts ON true
GROUP BY input.role_name, ts.schema_name
ORDER BY input.role_name, ts.schema_name`

	rows, err := db.Query(q, pq.Array(roles), schema)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sequence privilege summary: %w", err)
	}
	defer rows.Close()

	var result []sequenceSummary
	for rows.Next() {
		var p sequenceSummary
		if err := rows.Scan(&p.RoleName, &p.SchemaName, &p.SeqCount, &p.CanSelect, &p.CanUsage, &p.CanUpdate); err != nil {
			return nil, fmt.Errorf("failed scanning sequence summary: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func fetchFunctionSummary(db *sql.DB, roles []string, schema string) ([]functionSummary, error) {
	const q = `
WITH target_schemas AS (
    SELECT n.nspname
    FROM pg_namespace n
    WHERE n.nspname NOT LIKE 'pg_%'
      AND n.nspname <> 'information_schema'
      AND ($2 = '' OR n.nspname = $2)
), target_functions AS (
    SELECT n.nspname AS schema_name,
           p.proname,
           pg_get_function_identity_arguments(p.oid) AS identity_args
    FROM pg_proc p
    JOIN pg_namespace n ON n.oid = p.pronamespace
    JOIN target_schemas s ON s.nspname = n.nspname
)
SELECT input.role_name,
       tf.schema_name,
       count(*)::int AS function_count,
       count(*) FILTER (WHERE has_function_privilege(input.role_name, format('%I.%I(%s)', tf.schema_name, tf.proname, tf.identity_args), 'EXECUTE'))::int AS can_execute
FROM unnest($1::text[]) AS input(role_name)
JOIN target_functions tf ON true
GROUP BY input.role_name, tf.schema_name
ORDER BY input.role_name, tf.schema_name`

	rows, err := db.Query(q, pq.Array(roles), schema)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch function privilege summary: %w", err)
	}
	defer rows.Close()

	var result []functionSummary
	for rows.Next() {
		var p functionSummary
		if err := rows.Scan(&p.RoleName, &p.SchemaName, &p.FuncCount, &p.CanExecute); err != nil {
			return nil, fmt.Errorf("failed scanning function summary: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func fetchDefaultPrivileges(db *sql.DB, roles []string, schema string) ([]defaultPrivilege, error) {
	const q = `
SELECT pg_get_userbyid(d.defaclrole) AS grantor,
       COALESCE(n.nspname, '(all)') AS schema_name,
       CASE d.defaclobjtype
           WHEN 'r' THEN 'tables'
           WHEN 'S' THEN 'sequences'
           WHEN 'f' THEN 'functions'
           WHEN 'T' THEN 'types'
           WHEN 'n' THEN 'schemas'
           ELSE d.defaclobjtype::text
       END AS object_type,
       pg_get_userbyid(a.grantee) AS grantee,
       a.privilege_type,
       a.is_grantable
FROM pg_default_acl d
LEFT JOIN pg_namespace n ON n.oid = d.defaclnamespace
CROSS JOIN LATERAL aclexplode(COALESCE(d.defaclacl, '{}'::aclitem[])) a
WHERE pg_get_userbyid(a.grantee) = ANY($1)
  AND ($2 = '' OR n.nspname = $2)
ORDER BY grantee, schema_name, object_type, privilege_type`

	rows, err := db.Query(q, pq.Array(roles), schema)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch default privileges: %w", err)
	}
	defer rows.Close()

	var result []defaultPrivilege
	for rows.Next() {
		var p defaultPrivilege
		if err := rows.Scan(&p.Grantor, &p.SchemaName, &p.ObjectType, &p.Grantee, &p.Privilege, &p.Grantable); err != nil {
			return nil, fmt.Errorf("failed scanning default privileges: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func fetchOwnedObjects(db *sql.DB, roles []string) ([]ownedObjects, error) {
	const q = `
SELECT input.role_name,
       (SELECT count(*)::int FROM pg_class c JOIN pg_roles r ON r.oid = c.relowner WHERE r.rolname = input.role_name AND c.relkind = 'r') AS owned_tables,
       (SELECT count(*)::int FROM pg_class c JOIN pg_roles r ON r.oid = c.relowner WHERE r.rolname = input.role_name AND c.relkind = 'S') AS owned_sequences,
       (SELECT count(*)::int FROM pg_class c JOIN pg_roles r ON r.oid = c.relowner WHERE r.rolname = input.role_name AND c.relkind = 'v') AS owned_views,
       (SELECT count(*)::int FROM pg_proc p JOIN pg_roles r ON r.oid = p.proowner WHERE r.rolname = input.role_name) AS owned_functions
FROM unnest($1::text[]) AS input(role_name)
ORDER BY input.role_name`

	rows, err := db.Query(q, pq.Array(roles))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch owned objects: %w", err)
	}
	defer rows.Close()

	var result []ownedObjects
	for rows.Next() {
		var o ownedObjects
		if err := rows.Scan(&o.RoleName, &o.OwnedTables, &o.OwnedSequences, &o.OwnedViews, &o.OwnedFunctions); err != nil {
			return nil, fmt.Errorf("failed scanning owned objects: %w", err)
		}
		result = append(result, o)
	}
	return result, rows.Err()
}

func printHeader(roles []string, schema string) {
	fmt.Println("====================================================")
	fmt.Println("pg-validate: Role and Grant Validation Report")
	fmt.Println("====================================================")
	fmt.Println("Roles:", strings.Join(roles, ", "))
	if schema == "" {
		fmt.Println("Schema filter: <all non-system schemas>")
	} else {
		fmt.Println("Schema filter:", schema)
	}
	fmt.Println()
}

func printRoleOverview(rows []roleOverview) {
	fmt.Println("[1] Role Overview")
	for _, r := range rows {
		if !r.RoleExists {
			fmt.Printf("- %s: NOT FOUND\n", r.RequestedRole)
			continue
		}
		fmt.Printf("- %s: login=%v conn_limit=%d super=%v createdb=%v createrole=%v replication=%v bypassrls=%v\n",
			r.RequestedRole,
			nullBool(r.CanLogin),
			nullInt(r.ConnLimit),
			nullBool(r.SuperUser),
			nullBool(r.CreateDB),
			nullBool(r.CreateRole),
			nullBool(r.Replication),
			nullBool(r.BypassRLS),
		)
	}
	fmt.Println()
}

func printMemberships(rows []membership) {
	fmt.Println("[2] Role Memberships")
	for _, r := range rows {
		if len(r.Members) == 0 {
			fmt.Printf("- %s: member_of=<none>\n", r.RoleName)
			continue
		}
		fmt.Printf("- %s: member_of=%s\n", r.RoleName, strings.Join(r.Members, ","))
	}
	fmt.Println()
}

func printDatabasePrivileges(rows []dbPrivilege) {
	fmt.Println("[3] Database Privileges (current database)")
	for _, r := range rows {
		fmt.Printf("- %s: CONNECT=%v CREATE=%v TEMP=%v\n", r.RoleName, r.CanConnect, r.CanCreate, r.CanTemp)
	}
	fmt.Println()
}

func printSchemaPrivileges(rows []schemaPrivilege) {
	fmt.Println("[4] Schema Privileges")
	if len(rows) == 0 {
		fmt.Println("- No schemas matched the filter")
		fmt.Println()
		return
	}
	for _, r := range rows {
		fmt.Printf("- %s on %s: USAGE=%v CREATE=%v\n", r.RoleName, r.SchemaName, r.CanUsage, r.CanCreate)
	}
	fmt.Println()
}

func printTableSummary(rows []tableSummary) {
	fmt.Println("[5] Table Privilege Summary")
	if len(rows) == 0 {
		fmt.Println("- No tables found in target schemas")
		fmt.Println()
		return
	}
	for _, r := range rows {
		fmt.Printf("- %s on %s: total=%d SELECT=%d INSERT=%d UPDATE=%d DELETE=%d TRUNCATE=%d REFERENCES=%d TRIGGER=%d\n",
			r.RoleName, r.SchemaName, r.TableCount, r.CanSelect, r.CanInsert, r.CanUpdate, r.CanDelete, r.CanTruncate, r.CanReference, r.CanTrigger)
	}
	fmt.Println()
}

func printSequenceSummary(rows []sequenceSummary) {
	fmt.Println("[6] Sequence Privilege Summary")
	if len(rows) == 0 {
		fmt.Println("- No sequences found in target schemas")
		fmt.Println()
		return
	}
	for _, r := range rows {
		fmt.Printf("- %s on %s: total=%d SELECT=%d USAGE=%d UPDATE=%d\n", r.RoleName, r.SchemaName, r.SeqCount, r.CanSelect, r.CanUsage, r.CanUpdate)
	}
	fmt.Println()
}

func printFunctionSummary(rows []functionSummary) {
	fmt.Println("[7] Function Privilege Summary")
	if len(rows) == 0 {
		fmt.Println("- No functions found in target schemas")
		fmt.Println()
		return
	}
	for _, r := range rows {
		fmt.Printf("- %s on %s: total=%d EXECUTE=%d\n", r.RoleName, r.SchemaName, r.FuncCount, r.CanExecute)
	}
	fmt.Println()
}

func printDefaultPrivileges(rows []defaultPrivilege) {
	fmt.Println("[8] Default Privileges")
	if len(rows) == 0 {
		fmt.Println("- No default privileges found for target roles")
		fmt.Println()
		return
	}
	for _, r := range rows {
		fmt.Printf("- grantee=%s schema=%s object=%s privilege=%s grantor=%s grantable=%v\n",
			r.Grantee, r.SchemaName, r.ObjectType, r.Privilege, r.Grantor, r.Grantable)
	}
	fmt.Println()
}

func printOwnedObjects(rows []ownedObjects) {
	fmt.Println("[9] Owned Objects")
	for _, r := range rows {
		fmt.Printf("- %s: tables=%d sequences=%d views=%d functions=%d\n", r.RoleName, r.OwnedTables, r.OwnedSequences, r.OwnedViews, r.OwnedFunctions)
	}
	fmt.Println()
}

func nullBool(v sql.NullBool) bool {
	if !v.Valid {
		return false
	}
	return v.Bool
}

func nullInt(v sql.NullInt64) int64 {
	if !v.Valid {
		return 0
	}
	return v.Int64
}
