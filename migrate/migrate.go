// migrate is the simple migrator, it has one option, which is whether it should remove
// other tables and fields that aren't mentioned in the schema.
package migrate

import (
	"database/sql"
	"fmt"

	"github.com/acsellers/doc/schema"
)

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
	return false, nil
}

func (d *Database) Migrate() error {
	current, err := d.UpToDate()
	if err != nil || current {
		return err
	}

	for _, table := range d.NewTables {
		err = d.CreateTable(table)
		if err != nil {
			return err
		}
	}

	for _, table := range d.ModifiedTables {
		d.ModifyTable(table)
	}

	return nil
}

func (d *Database) PareFields() error {
	return nil
}

func (d *Database) CreateTable(table *schema.Table) error {
	switch d.DBMS {
	case Generic, Sqlite:
		return (&GenericDB{d.DB, d.Translator}).CreateTable(table)
	}
	return fmt.Errorf("Could not locate database for %v", d.DBMS)
}

func (d *Database) ModifyTable(table *schema.Table) {

}
