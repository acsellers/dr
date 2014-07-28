package main

import (
	"github.com/acsellers/doc/forum"
	"github.com/acsellers/doc/migrate"
)

func main() {
	db := migrate.Database{
		Schema:     forum.Schema,
		Translator: &forum.AppConfig{},
		DBMS:       migrate.Generic,
	}
	db.Migrate()
}
