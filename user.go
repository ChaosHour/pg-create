package main

import (
	_ "database/sql"
	"fmt"
	"strings"
)

// use a case statement to determine which functions to call for the role grants. Like what is used in the createGrants func - KL
func createRole() {
	// check if role already exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM pg_roles WHERE rolname = $1", *roleName).Scan(&count)
	if err != nil {
		panic(err)
	}
	if count > 0 {
		fmt.Println(yellow("[*]"), "Role", *roleName, "already exists")
	} else {
		// create role
		_, err = db.Exec("CREATE ROLE " + *roleName + " LOGIN PASSWORD '" + *password + "'")
		if err != nil {
			panic(err)
		}
		fmt.Println(green("[+]"), "Role", *roleName, "created")

		// add grants to role
		_, err = db.Exec("GRANT USAGE ON SCHEMA " + *schemaName + " TO " + *roleName)
		if err != nil {
			panic(err)
		}
		_, err = db.Exec("GRANT SELECT ON ALL SEQUENCES IN SCHEMA " + *schemaName + " TO " + *roleName)
		if err != nil {
			panic(err)
		}
		_, err = db.Exec("GRANT INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA " + *schemaName + " TO " + *roleName)
		if err != nil {
			panic(err)
		}
		//fmt.Println(green("[+]"), "Role", *roleName, "granted privileges for schema", *schemaName)

		// grant privileges to user for schema
		// ALTER DEFAULT PRIVILEGES FOR ROLE example_user IN SCHEMA public GRANT SELECT ON TABLES TO example_user;
		_, err = db.Exec("ALTER DEFAULT PRIVILEGES FOR ROLE " + *roleName + " IN SCHEMA " + *schemaName + " GRANT SELECT ON TABLES TO " + *roleName)
		if err != nil {
			panic(err)
		}
		//fmt.Println(green("[+]"), "Role", *roleName, "granted privileges for schema", *schemaName)
	}

	// add user to role
	_, err = db.Exec("GRANT " + *roleName + " TO " + *userName)
	if err != nil {
		panic(err)
	}
	fmt.Println(green("[+]"), "User", *userName, "added to role", *roleName)
}

func createUser() {
	// check if user already exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM pg_user WHERE usename = $1", *userName).Scan(&count)
	if err != nil {
		panic(err)
	}
	if count > 0 {
		fmt.Println(yellow("[*]"), "User", *userName, "already exists")
		return
	}

	// create user
	_, err = db.Exec("CREATE USER " + *userName + " WITH PASSWORD '" + *password + "'")
	if err != nil {
		panic(err)
	}
	fmt.Println(green("[+]"), "User", *userName, "created")
}

func createGrants() {
	// check if role has grants
	var count int
	err = db.QueryRow("select count(*) from information_schema.role_table_grants where grantee = $1 and table_schema = $2", *roleName, *schemaName).Scan(&count)
	if err != nil {
		panic(err)
	}
	if count > 0 {
		fmt.Println(yellow("[*]"), "Role", *roleName, "already has privileges for schema", *schemaName)
		return
	}

	// split grant types into array
	grantTypes := strings.Split(*grants, ",")

	// apply grants based on grant types
	for _, grantType := range grantTypes {
		switch grantType {
		case "usage":
			_, err := db.Exec("GRANT USAGE ON SCHEMA " + *schemaName + " TO " + *roleName)
			if err != nil {
				panic(err)
			}
			fmt.Println(green("[+]"), "Role", *roleName, "granted USAGE privilege for schema", *schemaName)
		case "select":
			_, err := db.Exec("GRANT SELECT ON ALL TABLES IN SCHEMA " + *schemaName + " TO " + *roleName)
			if err != nil {
				panic(err)
			}
			fmt.Println(green("[+]"), "Role", *roleName, "granted SELECT privilege for all tables in schema", *schemaName)
		case "insert":
			_, err := db.Exec("GRANT INSERT ON ALL TABLES IN SCHEMA " + *schemaName + " TO " + *roleName)
			if err != nil {
				panic(err)
			}
			fmt.Println(green("[+]"), "Role", *roleName, "granted INSERT privilege for all tables in schema", *schemaName)
		case "update":
			_, err := db.Exec("GRANT UPDATE ON ALL TABLES IN SCHEMA " + *schemaName + " TO " + *roleName)
			if err != nil {
				panic(err)
			}
			fmt.Println(green("[+]"), "Role", *roleName, "granted UPDATE privilege for all tables in schema", *schemaName)
		case "delete":
			_, err := db.Exec("GRANT DELETE ON ALL TABLES IN SCHEMA " + *schemaName + " TO " + *roleName)
			if err != nil {
				panic(err)
			}
			fmt.Println(green("[+]"), "Role", *roleName, "granted DELETE privilege for all tables in schema", *schemaName)
		default:
			fmt.Println(red("[-]"), "Invalid grant type:", grantType)
		}
	}
}

// funtion to create database if not exists
func createDatabase() {
	// check if database already exists
	var database string
	err = db.QueryRow("select datname from pg_database where datname = $1", *dbName).Scan(&database)
	if err != nil {
		fmt.Println(yellow("[*]"), "Database", *dbName, "does not exist")
		_, err = db.Exec("create database " + *dbName + " with owner " + *userName)
		if err != nil {
			panic(err)
		}
		fmt.Println(green("[+]"), "Database", *dbName, "created")
	} else {
		fmt.Println(yellow("[*]"), "Database", *dbName, "already exists")
	}
}

func grantDatabase() {
	// create database if it does not exist
	createDatabase()

	// grant database to user
	if !checkUserHasDatabase() {
		_, err = db.Exec("grant connect on database " + *dbName + " to " + *userName)
		if err != nil {
			panic(err)
		}
		fmt.Println(green("[+]"), "Database", *dbName, "granted to user", *userName)
	}
}

func checkUserHasDatabase() bool {
	var user string
	err = db.QueryRow("select datname from pg_database where datname = $1", *dbName).Scan(&user)
	if err != nil {
		return false
	}
	fmt.Println(yellow("[*]"), "User", *userName, "already has database", *dbName)
	return true
}

// function to check if schema exists and if not create it
func checkSchemaExists() bool {
	var schema string
	err = db.QueryRow("select schema_name from information_schema.schemata where schema_name = $1", *schemaName).Scan(&schema)
	if err != nil {
		// create schema
		_, err = db.Exec("create schema " + *schemaName)
		if err != nil {
			panic(err)
		}
		fmt.Println(green("[+]"), "Schema", *schemaName, "created")
		return false
	}
	fmt.Println(yellow("[*]"), "Schema", *schemaName, "already exists")

	// grant privileges to user for schema
	grantSchema()

	return true
}

// function to grant privileges to user for schema
func grantSchema() {
	// check if user has access to schema
	var count int
	err = db.QueryRow("select count(*) from information_schema.role_table_grants where grantee = $1 and table_schema = $2", *userName, *schemaName).Scan(&count)
	if err != nil {
		panic(err)
	}
	if count == 0 {
		// grant privileges to user for schema
		_, err = db.Exec("grant select on all tables in schema " + *schemaName + " to " + *userName)
		if err != nil {
			panic(err)
		}
		// grant privileges to role for schema
		_, err := db.Exec("GRANT USAGE ON SCHEMA " + *schemaName + " TO " + *roleName)
		if err != nil {
			panic(err)
		}
		_, err = db.Exec("GRANT SELECT ON ALL SEQUENCES IN SCHEMA " + *schemaName + " TO " + *roleName)
		if err != nil {
			panic(err)
		}
		_, err = db.Exec("GRANT INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA " + *schemaName + " TO " + *roleName)
		if err != nil {
			panic(err)
		}
		//fmt.Println(green("[+]"), "Role", *roleName, "granted privileges for schema", *schemaName)
		//fmt.Println(green("[+]"), "User", *userName, "granted privileges for schema", *schemaName)
	} else {
		fmt.Println(yellow("[*]"), "User", *userName, "already has privileges for schema", *schemaName)
	}
}

func createSchema() bool {
	// check if schema exists
	var count int
	err = db.QueryRow("select count(*) from information_schema.schemata where schema_name = $1", *schemaName).Scan(&count)
	if err != nil {
		panic(err)
	}
	if count == 0 {
		// create schema
		_, err = db.Exec("CREATE SCHEMA " + *schemaName + " AUTHORIZATION " + *userName)
		if err != nil {
			panic(err)
		}
		fmt.Println(green("[+]"), "Schema", *schemaName, "created")

		// set owner of schema to user
		_, err = db.Exec("ALTER SCHEMA " + *schemaName + " OWNER TO " + *userName)
		if err != nil {
			panic(err)
		}
		fmt.Println(green("[+]"), "Schema", *schemaName, "owner set to", *userName)

		// grant privileges to user for schema
		grantSchema()

		return true
	}

	// set owner of schema to user if not already set
	var owner string
	err = db.QueryRow("SELECT schema_owner FROM information_schema.schemata WHERE schema_name = $1", *schemaName).Scan(&owner)
	if err != nil {
		panic(err)
	}
	if owner != *userName {
		_, err = db.Exec("ALTER SCHEMA " + *schemaName + " OWNER TO " + *userName)
		if err != nil {
			panic(err)
		}
		fmt.Println(green("[+]"), "Schema", *schemaName, "owner set to", *userName)
	}

	//fmt.Println(yellow("[*]"), "Schema", *schemaName, "already exists")

	// grant privileges to user for schema
	grantSchema()

	return false
}
