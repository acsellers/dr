package migrate

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/acsellers/doc/schema"
)

type GenericDB struct {
	DB      *sql.DB
	Convert Translator
}

func (g *GenericDB) CreateTable(table *schema.Table) error {
	// vals := []interface{}{}
	sql := fmt.Sprintf(
		"CREATE TABLE %s(",
		g.Convert.SQLTable(table.Name),
	)

	defs := []string{}
	for i, column := range table.Columns {
		switch {
		case i == 0 && column.Type == "integer":
			defs = append(
				defs,
				fmt.Sprintf(
					"%s INTEGER PRIMARY KEY ASC",
					g.Convert.SQLColumn(table.Name, column.Name),
				),
			)
		default:
			coldef := fmt.Sprintf(
				"%s %s",
				g.Convert.SQLColumn(table.Name, column.Name),
				strings.ToUpper(column.Type),
			)
			if column.Length != 0 {
				coldef += fmt.Sprintf(
					"(%d)",
					column.Length,
				)
			}
			defs = append(defs, coldef)
		}
	}
	for _, child := range table.ChildOf {
		defs = append(
			defs,
			fmt.Sprintf(
				"FOREIGN KEY(%s) REFERENCES %s(%s)",
				g.Convert.SQLColumn(table.Name, child.ChildColumn.Name),
				g.Convert.SQLTable(child.Parent.Name),
				g.Convert.SQLColumn(child.Parent.Name, child.Parent.PrimaryKeyColumn().Name),
			),
		)
	}

	for _, belonging := range table.BelongsTo {
		defs = append(
			defs,
			fmt.Sprintf(
				"FOREIGN KEY(%s) REFERENCES %s(%s)",
				g.Convert.SQLColumn(table.Name, belonging.ChildColumn.Name),
				g.Convert.SQLTable(belonging.Parent.Name),
				g.Convert.SQLColumn(belonging.Parent.Name, belonging.Parent.PrimaryKeyColumn().Name),
			),
		)
	}

	sql += strings.Join(defs, ", ") + ")"

	fmt.Println(sql)
	return nil
	//_, err := g.DB.Exec(sql, vals)
	//return err
}

func (g *GenericDB) AddColumn(table *schema.Table, col *schema.Column) error {
	return nil
}

func (g *GenericDB) RenameColumn(table *schema.Table, col *schema.Column) error {
	return nil
}

func (g *GenericDB) RemoveColumn(table *schema.Table, col *schema.Column) error {
	return nil
}

func (g *GenericDB) ModifyColumn(table *schema.Table, col *schema.Column) error {
	return nil
}

// SqliteDB is the standard for the GenericDB, so it has no overridden functions
type SqliteDB struct {
	GenericDB
}

// PostgresDB is pretty much the same as sqlite, except it can modify columns
type PostgresDB struct {
	GenericDB
}

// MysqlDB is ugh
type MysqlDB struct {
	GenericDB
}
