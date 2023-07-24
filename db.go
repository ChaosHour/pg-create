package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

// global variables
var (
	db  *sql.DB
	err error
)

func initDB() {
	// read the contents of the .pgpass file
	file, err := os.Open(os.Getenv("HOME") + "/.pgpass")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue // skip comments
		}
		fields := strings.Split(line, ":")
		if len(fields) < 5 {
			continue // skip invalid lines
		}
		host := fields[0]
		port := fields[1]
		dbname := fields[2]
		user := fields[3]
		password := fields[4]

		var err error
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			panic(err)
		}
		err = db.Ping()
		if err != nil {
			panic(err)
		}
		//fmt.Println(green("[+]"), "Connected to database")
		fmt.Println(green("✓"), "Connected to database")

		break // exit the loop after the first valid line is found
	}
}
