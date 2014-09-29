// migrate is the simple migrator, it has one option, which is whether it should remove
// other tables and fields that aren't mentioned in the schema.
package migrate

import (
	"database/sql"
	"io/ioutil"
	"log"

	"github.com/acsellers/dr/schema"
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

type Alterer interface {
	HasTable(*schema.Table) (bool, error)
	CreateTable(*schema.Table) error
	RemoveTable(*schema.Table) error
	UpdateTable(*schema.Table) error
	RenameTable(*schema.Table, string) error

	HasColumn(*schema.Table, *schema.Column) (bool, error)
	CreateColumn(*schema.Table, *schema.Column) error
	ModifyColumn(*schema.Table, *schema.Column) error
	RenameColumn(*schema.Table, *schema.Column) error
	RemoveColumn(*schema.Table, *schema.Column) error
}

type Database struct {
	Alterer
	DB     *sql.DB
	Schema schema.Schema
	Translator
	NewTables      []*schema.Table
	ModifiedTables []*schema.Table
	DBMS           System
	Log            *log.Logger
}

func (d *Database) UpToDate() (bool, error) {
TableIter:
	for _, table := range d.Schema.Tables {
		d.Log.Println("Checking For Table:", table.Name)
		exists, err := d.HasTable(table)
		if err != nil {
			d.Log.Println("Error Checking for Table:", table.Name)
			d.Log.Println("Error Was :", err)
			return false, err
		}
		if !exists {
			d.Log.Println("Non-Existant Table:", table.Name)
			d.NewTables = append(d.NewTables, table)
			continue
		}

		for _, col := range table.Columns {

			exists, err = d.HasColumn(table, &col)
			if err != nil {
				d.Log.Println("Error Checking Column", col.Name, "for Table", table.Name)
				d.Log.Println("Error Was:", err)
				return false, err
			}
			if !exists {
				d.Log.Println("Non-Existant Column", col.Name, "for Table", table.Name)
				d.Log.Println("Setting Table", table.Name, "to have field(s) added")
				d.ModifiedTables = append(d.ModifiedTables, table)
				continue TableIter
			}
		}
		d.Log.Println("Table", table.Name, "is up to date")
	}
	return false, nil
}

func (d *Database) Migrate() error {
	if d.Log == nil {
		d.Log = log.New(ioutil.Discard, "", 0)
	}

	d.Log.Println("Starting Migration Code")
	if d.Alterer == nil {
		d.SetAlterer()
	}
	d.Log.Println("Alterer Type is Set to", d.Alterer)

	d.Log.Println("Beginning Check for Tables that need Migrations")
	current, err := d.UpToDate()
	if err != nil || current {
		return err
	}

	d.Log.Printf("Creating New Tables (%d)\n", len(d.NewTables))
	for _, table := range d.NewTables {
		err = d.CreateTable(table)
		if err != nil {
			return err
		}
	}

	d.Log.Printf("Modifying Existing Tables (%d)\n", len(d.ModifiedTables))
	for _, table := range d.ModifiedTables {
		d.UpdateTable(table)
	}

	d.Log.Println("Completed migration")
	return nil
}

func (d *Database) PareFields() error {
	return nil
}

func (d *Database) SetAlterer() {
	switch d.DBMS {
	case Sqlite:
		d.Alterer = &SqliteDB{GenericDB{DB: d.DB, Convert: d.Translator, Log: d.Log}}
	case Postgres:
		d.Alterer = &PostgresDB{GenericDB{DB: d.DB, Convert: d.Translator, Log: d.Log}}
	case MySQL:
		d.Alterer = &MysqlDB{GenericDB{DB: d.DB, Convert: d.Translator, Log: d.Log}}
	}
}
