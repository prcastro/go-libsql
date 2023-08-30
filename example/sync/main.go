package main

import (
	"database/sql"
	"fmt"
	"github.com/libsql/go-libsql"
	"os"
	"strings"
)

func run() (err error) {
	primaryUrl := os.Getenv("TURSO_URL")
	if primaryUrl == "" {
		return fmt.Errorf("TURSO_URL environment variable not set")
	}
	dir, err := os.MkdirTemp("", "libsql-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	connector, err := libsql.NewEmbeddedReplicaConnector(dir+"/test.db", primaryUrl)
	if err != nil {
		return err
	}
	defer func() {
		if closeError := connector.Close(); closeError != nil {
			fmt.Println("Error closing connector", closeError)
			if err == nil {
				err = closeError
			}
		}
	}()

	db := sql.OpenDB(connector)
	defer func() {
		if closeError := db.Close(); closeError != nil {
			fmt.Println("Error closing database", closeError)
			if err == nil {
				err = closeError
			}
		}
	}()

	for {
		fmt.Println("What would you like to do?")
		fmt.Println("1. Sync with primary")
		fmt.Println("2. Select from test table")
		fmt.Println("3. Exit")
		var choice int
		_, err := fmt.Scanln(&choice)
		if err != nil {
			return err
		}
		switch choice {
		case 1:
			if err := connector.Sync(); err != nil {
				return err
			}
		case 2:
			err = func() (err error) {
				rows, err := db.Query("SELECT * FROM test")
				if err != nil {
					if strings.Contains(err.Error(), "`no such table: test`") {
						fmt.Println("Table test not found. Please run `CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)` on primary first and then sync.")
						return nil
					}
					return err
				}
				defer func() {
					if closeError := rows.Close(); closeError != nil {
						fmt.Println("Error closing rows", closeError)
						if err == nil {
							err = closeError
						}
					}
				}()
				count := 0
				for rows.Next() {
					var id int
					var name string
					err = rows.Scan(&id, &name)
					if err != nil {
						return err
					}
					fmt.Println(id, name)
					count++
				}
				if rows.Err() != nil {
					return rows.Err()
				}
				if count == 0 {
					fmt.Println("Empty table. Please run `INSERT INTO test (id, name) VALUES (random(), lower(hex(randomblob(16))))` on primary and then sync.")
				}
				return nil
			}()
			if err != nil {
				return err
			}
		case 3:
			return nil
		}
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
