package migrate

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/acsellers/doc/schema"
)

type GenericDB struct {
	DB      *sql.DB
	Convert Translator
	Log     *log.Logger
}

func (g *GenericDB) HasTable(table *schema.Table) (bool, error) {
	return false, fmt.Errorf("Need a RDBMS-specific for schema check functionality")
}

func (g *GenericDB) CreateTable(table *schema.Table) error {
	vals := []interface{}{}
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

	_, err := g.DB.Exec(sql, vals...)
	return err
}

func (g *GenericDB) RemoveTable(table *schema.Table) error {
	_, err := g.DB.Exec(fmt.Sprint("DROP TABLE ", g.Convert.SQLTable(table.Name)))
	return err
}

func (g *GenericDB) RenameTable(table *schema.Table, oldName string) error {
	_, err := g.DB.Exec(
		fmt.Sprintf("ALTER TABLE %s RENAME TO %s", oldName, g.Convert.SQLTable(table.Name)),
	)
	return err
}

func (g *GenericDB) HasColumn(table *schema.Table, col *schema.Column) (bool, error) {
	return false, fmt.Errorf("Need a RDBMS-specific for schema check functionality")
}

func (g *GenericDB) CreateColumn(table *schema.Table, col *schema.Column) error {
	coldef := fmt.Sprintf(
		"%s %s",
		g.Convert.SQLColumn(table.Name, col.Name),
		strings.ToUpper(col.Type),
	)
	if col.Length != 0 {
		coldef += fmt.Sprintf(
			"(%d)",
			col.Length,
		)
	}
	sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", g.Convert.SQLTable(table.Name), coldef)
	_, err := g.DB.Exec(sql)

	return err
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

func (g *GenericDB) UpdateTable(table *schema.Table) error {
	return nil
}

// SqliteDB is the standard for the GenericDB, so it has no overridden functions
type SqliteDB struct {
	GenericDB
}

func (s *SqliteDB) HasTable(table *schema.Table) (bool, error) {
	var cnt int64
	err := s.DB.QueryRow(
		`SELECT COUNT(name) FROM sqlite_master WHERE type='table' AND name=?`,
		s.Convert.SQLTable(table.Name),
	).Scan(&cnt)
	if err != nil {
		return false, err
	}
	return cnt == 1, nil
}

func (s *SqliteDB) HasColumn(table *schema.Table, col *schema.Column) (bool, error) {
	rows, err := s.DB.Query(fmt.Sprintf("PRAGMA table_info(%s)", s.Convert.SQLTable(table.Name)))
	if err != nil {
		return false, err
	}
	var tinfo struct {
		CID        int
		Name       string
		Type       string
		NotNull    bool
		Default    interface{}
		PrimaryKey bool
	}
	for rows.Next() {
		err = rows.Scan(
			&tinfo.CID,
			&tinfo.Name,
			&tinfo.Type,
			&tinfo.NotNull,
			&tinfo.Default,
			&tinfo.PrimaryKey,
		)
		if err != nil {
			return false, err
		}
		if tinfo.Name == s.Convert.SQLColumn(table.Name, col.Name) {
			return true, nil
		}
	}
	return false, rows.Err()
}

func (s *SqliteDB) UpdateTable(table *schema.Table) error {
	for _, col := range table.Columns {
		exists, err := s.HasColumn(table, &col)
		if err != nil {
			return err
		}
		if !exists {
			s.Log.Println("Adding New Column", col.Name, "to", table.Name)
			err = s.CreateColumn(table, &col)
			if err != nil {
				s.Log.Println("Error when adding column", err)
				return err
			}
		}
	}

	return nil
}

func (*SqliteDB) String() string {
	return "sqlite"
}

// PostgresDB is pretty much the same as sqlite, except it can modify columns
type PostgresDB struct {
	GenericDB
}

func (*PostgresDB) String() string {
	return "Postgres Master Race"
}

// MysqlDB is ugh
type MysqlDB struct {
	GenericDB
}

func (*MysqlDB) String() string {
	return "myDerpDB"
}
