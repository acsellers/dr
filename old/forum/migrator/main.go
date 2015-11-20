package main

import (
	"database/sql"
	"fmt"

	"github.com/acsellers/dr/forum"
	"github.com/acsellers/dr/migrate"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	sdb, err := sql.Open("sqlite3", "./forum.db")
	if err != nil {
		fmt.Println("Error during database open", err)
		return
	}

	db := migrate.Database{
		DB:         sdb,
		Schema:     forum.Schema,
		Translator: &forum.AppConfig{},
		DBMS:       migrate.Generic,
	}
	db.Migrate()
	sdb.Close()
}
