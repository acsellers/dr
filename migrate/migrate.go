// migrate is the simple migrator, it has one option, which is whether it should remove
// other tables and fields that aren't mentioned in the schema.
package migrate

import (
	"database/sql"
	"flag"
	"fmt"

	"github.com/acsellers/doc/schema"
)

var PareFields = flag.Bool(
	"pare",
	false,
	"Remove tables and fields not mentioned in the schema",
)

func init() {
	flag.Parse()
}

type System int

const (
	Generic System = iota
	MySQL
	Sqlite
	Postgres
)

type Translator interface {
	SQLTable(string) string
	SQLColumn(string, string) string
}

type Database struct {
	DB     *sql.DB
	Schema schema.Schema
	Translator
	NewTables      []*schema.Table
	ModifiedTables []*schema.Table
	DBMS           System
}

func (d *Database) UpToDate() (bool, error) {
	for _, table := range d.Schema.Tables {
		d.NewTables = append(d.NewTables, table)
	}
	fmt.Println("Checked")
	return false, nil
}

func (d *Database) Migrate() error {
	current, err := d.UpToDate()
	if err != nil || current {
		return err
	}

	for _, table := range d.NewTables {
		fmt.Println("Create table", table.Name)
		d.CreateTable(table)
	}

	for _, table := range d.ModifiedTables {
		d.ModifyTable(table)
	}

	if *PareFields {
		return d.PareFields()
	}

	return nil
}

func (d *Database) PareFields() error {
	return nil
}

func (d *Database) CreateTable(table *schema.Table) {
	switch d.DBMS {
	case Generic, Sqlite:
		(&GenericDB{d.DB, d.Translator}).CreateTable(table)
	}
}

func (d *Database) ModifyTable(table *schema.Table) {

}
