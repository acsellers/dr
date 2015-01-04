package migrate

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/acsellers/dr/schema"
)

type GenericDB struct {
	DB                *sql.DB
	Specific          Alterer
	Convert           Translator
	Log               *log.Logger
	PrimaryKeyDef     string
	LengthableColumns map[string]bool
}

func (g *GenericDB) HasIndex(table *schema.Table, index schema.Index) (bool, error) {
	return false, nil
}
func (g *GenericDB) CreateIndex(table *schema.Table, index schema.Index) error {
	return g.Specific.CreateIndex(table, index)
}

func (*GenericDB) getIndexName(*schema.Table, schema.Index) (string, error) {
	return "", fmt.Errorf("Must use RDBMS specific version for this feature")
}

func (g *GenericDB) RemoveIndex(table *schema.Table, index schema.Index) error {
	name, err := g.getIndexName(table, index)
	if err != nil {
		return err
	}
	if name == "" {
		return nil
	}
	_, err = g.DB.Exec("DROP INDEX " + name)
	return err
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
					g.PrimaryKeyDef,
					g.Convert.SQLColumn(table.Name, column.Name),
				),
			)
		default:
			coldef := fmt.Sprintf(
				"%s %s",
				g.Convert.SQLColumn(table.Name, column.Name),
				strings.ToUpper(column.Type),
			)
			if column.Length != 0 && g.LengthableColumns[column.Type] {
				coldef += fmt.Sprintf(
					"(%d)", column.Length,
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
	if err != nil {
		return fmt.Errorf("Error Creating Table\nSQL: %s\nError: %s", sql, err.Error())
	}

	for _, index := range table.Index {
		ok, err := g.HasIndex(table, index)
		if err != nil {
			return fmt.Errorf("Error checking index: %v", err)
		}
		if !ok {
			g.CreateIndex(table, index)
		}
	}
	return nil
}

func (g *GenericDB) UpdateTable(table *schema.Table) error {
	for _, col := range table.Columns {
		exists, err := g.Specific.HasColumn(table, &col)
		if err != nil {
			g.Log.Println("Encountered Error:", err)
			return err
		}
		if !exists {
			g.Log.Println("Adding New Column", col.Name, "to", table.Name)
			err = g.Specific.CreateColumn(table, &col)
			if err != nil {
				g.Log.Println("Error when adding column", err)
				return err
			}
		}
	}

	for _, index := range table.Index {
		ok, err := g.HasIndex(table, index)
		if err != nil {
			return fmt.Errorf("Error checking index: %v", err)
		}
		if !ok {
			g.CreateIndex(table, index)
		}
	}

	return nil
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
	if col.Length != 0 && g.LengthableColumns[col.Type] {
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

// SqliteDB is the standard for the GenericDB, so it has no overridden functions
type SqliteDB struct {
	GenericDB
}

func (s *SqliteDB) CreateTable(table *schema.Table) error {
	s.GenericDB.Specific = s
	s.GenericDB.PrimaryKeyDef = "%s INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT"
	s.GenericDB.LengthableColumns = s.LengthableColumns()
	return s.GenericDB.CreateTable(table)
}

func (s *SqliteDB) UpdateTable(table *schema.Table) error {
	s.GenericDB.Specific = s
	s.GenericDB.PrimaryKeyDef = "%s INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT"
	s.GenericDB.LengthableColumns = s.LengthableColumns()
	return s.GenericDB.UpdateTable(table)
}

func (s *SqliteDB) HasTable(table *schema.Table) (bool, error) {
	var cnt int64
	err := s.DB.QueryRow(
		`SELECT COUNT(name) FROM sqlite_master WHERE type='table' AND name=$1`,
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
			rows.Close()
			return false, err
		}
		if tinfo.Name == s.Convert.SQLColumn(table.Name, col.Name) {
			rows.Close()
			return true, nil
		}
	}
	return false, rows.Err()
}

func (s *SqliteDB) HasIndex(table *schema.Table, index schema.Index) (bool, error) {
	name, err := s.getIndexName(table, index)
	if err != nil {
		return false, err
	}
	if name == "" {
		return false, nil
	}
	return true, nil
}

func (s *SqliteDB) getIndexName(table *schema.Table, index schema.Index) (string, error) {
	rows, err := s.DB.Query(fmt.Sprintf("PRAGMA index_list(%s)", s.Convert.SQLTable(table.Name)))
	if err != nil {
		return "", err
	}
	indexes := []string{}
	var temp1 interface{}
	var indexName string
	var unique bool
	for rows.Next() {
		err = rows.Scan(&temp1, &indexName, &unique)
		if err != nil {
			return "", fmt.Errorf("Enumerate Sqlite indexes: %v", err)
		}
		if index.Unique == unique {
			indexes = append(indexes, indexName)
		}
	}
	rows.Close()

	for _, dbindex := range indexes {
		rows, err := s.DB.Query(fmt.Sprintf("PRAGMA index_info(%s)", dbindex))
		if err != nil {
			return "", err
		}
		columns := []string{}
		var temp1, temp2 interface{}
		var columnName string
		for rows.Next() {
			err = rows.Scan(&temp1, &temp2, &columnName)
			if err != nil {
				return "", fmt.Errorf("Interrogate Sqlite index: %v", err)
			}
			columns = append(columns, columnName)
		}
		rows.Close()
		if len(columns) != len(index.Columns) {
			continue
		}
		for i, column := range columns {
			if column != s.Convert.SQLColumn(table.Name, index.Columns[i]) {
				continue
			}
		}
		return dbindex, nil
	}

	return "", nil
}

func (s *SqliteDB) CreateIndex(table *schema.Table, index schema.Index) error {
	unique := ""
	if index.Unique {
		unique = "UNIQUE"
	}

	indexName := strings.Join(append([]string{"idx", s.Convert.SQLTable(table.Name)}, index.Columns...), "_")

	columns := make([]string, len(index.Columns))
	for i, col := range index.Columns {
		columns[i] = s.Convert.SQLColumn(table.Name, col)
	}

	sql := fmt.Sprintf(
		"CREATE %s INDEX %s ON %s (%s)",
		unique,
		indexName,
		s.Convert.SQLTable(table.Name),
		strings.Join(columns, ", "),
	)
	_, err := s.DB.Exec(sql)

	return err
}

func (*SqliteDB) String() string {
	return "sqlite"
}

func (*SqliteDB) LengthableColumns() map[string]bool {
	return map[string]bool{
		"varchar": true,
		"integer": true,
	}
}

// PostgresDB is pretty much the same as sqlite, except it can modify columns
type PostgresDB struct {
	GenericDB
}

func (*PostgresDB) String() string {
	return "Postgres Master Race"
}

func (p PostgresDB) HasTable(table *schema.Table) (bool, error) {
	var cnt int64
	err := p.DB.QueryRow(
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_type = 'BASE TABLE' AND table_schema = 'public' AND table_name = $1`,
		p.Convert.SQLTable(table.Name),
	).Scan(&cnt)
	if err != nil {
		return false, err
	}
	return cnt == 1, nil
}

func (p *PostgresDB) HasColumn(table *schema.Table, col *schema.Column) (bool, error) {
	var cnt int64
	err := p.DB.QueryRow(
		`SELECT COUNT(*) FROM information_schema.columns WHERE table_name = $1 AND column_name = $2`,
		p.Convert.SQLTable(table.Name),
		p.Convert.SQLColumn(table.Name, col.Name),
	).Scan(&cnt)
	if err != nil {
		return false, err
	}
	return cnt == 1, nil
}

func (p *PostgresDB) CreateTable(table *schema.Table) error {
	p.GenericDB.Specific = p
	p.GenericDB.PrimaryKeyDef = "%s SERIAL PRIMARY KEY"
	p.GenericDB.LengthableColumns = p.LengthableColumns()
	return p.GenericDB.CreateTable(table)
}

func (p *PostgresDB) UpdateTable(table *schema.Table) error {
	p.GenericDB.Specific = p
	p.GenericDB.PrimaryKeyDef = "%s SERIAL PRIMARY KEY"
	p.GenericDB.LengthableColumns = p.LengthableColumns()
	return p.GenericDB.UpdateTable(table)
}

func (p *PostgresDB) HasIndex(table *schema.Table, index schema.Index) (bool, error) {
	name, err := p.getIndexName(table, index)
	return name != "", err
}
func (p *PostgresDB) getIndexName(table *schema.Table, index schema.Index) (string, error) {
	sql := `select i.relname as index_name, array_to_string(array_agg(a.attname), ',') as column_names
from pg_class t, pg_class i, pg_index ix, pg_attribute a
where t.oid = ix.indrelid and i.oid = ix.indexrelid
and a.attrelid = t.oid and a.attnum = ANY(ix.indkey)
and t.relkind = 'r' and t.relname like $1
group by t.relname, i.relname
order by t.relname, i.relname`
	rows, err := p.DB.Query(sql, p.Convert.SQLTable(table.Name))
	if err != nil {
		return "", err
	}
	lookColumns := strings.Join(index.Columns, ",")
	var indexName, columns string
	for rows.Next() {
		err = rows.Scan(&indexName, &columns)
		if err != nil {
			rows.Close()
			return "", err
		}
		if columns == lookColumns {
			rows.Close()
			return indexName, nil
		}
	}
	rows.Close()
	return "", nil
}
func (p *PostgresDB) CreateIndex(table *schema.Table, index schema.Index) error {
	indexName := strings.Join(append([]string{"idx", p.Convert.SQLTable(table.Name)}, index.Columns...), "_")

	columns := make([]string, len(index.Columns))
	for i, col := range index.Columns {
		columns[i] = p.Convert.SQLColumn(table.Name, col)
	}

	sql := fmt.Sprintf(
		"CREATE INDEX %s ON %s (%s)",
		indexName,
		p.Convert.SQLTable(table.Name),
		strings.Join(columns, ", "),
	)
	_, err := p.DB.Exec(sql)

	return err
}
func (p *PostgresDB) LengthableColumns() map[string]bool {
	return map[string]bool{
		"varchar": true,
	}
}

// MysqlDB is ugh
type MysqlDB struct {
	GenericDB
}

func (*MysqlDB) String() string {
	return "myDerpDB"
}

func (*MysqlDB) LengthableColumns() map[string]bool {
	return map[string]bool{
		"varchar": true,
		"integer": true,
	}
}

func (m *MysqlDB) CreateTable(table *schema.Table) error {
	m.GenericDB.Specific = m
	m.GenericDB.PrimaryKeyDef = "%s SERIAL PRIMARY KEY"
	m.GenericDB.LengthableColumns = m.LengthableColumns()
	return m.GenericDB.CreateTable(table)
}

func (m *MysqlDB) UpdateTable(table *schema.Table) error {
	m.GenericDB.Specific = m
	m.GenericDB.PrimaryKeyDef = "%s SERIAL PRIMARY KEY"
	m.GenericDB.LengthableColumns = m.LengthableColumns()
	return m.GenericDB.UpdateTable(table)
}

func (m *MysqlDB) getIndexName(table *schema.Table, index schema.Index) (string, error) {
	sql := `SELECT INDEX_NAME, GROUP_CONCAT(DISTINCT COLUMN_NAME ORDER BY SEQ_IN_INDEX) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_NAME = ? AND INDEX_NAME <> 'PRIMARY' GROUP BY INDEX_NAME`
	rows, err := m.DB.Query(sql, m.Convert.SQLTable(table.Name))
	if err != nil {
		return "", err
	}
	search := strings.Join(index.Columns, ",")
	var indexName, columns string
	for rows.Next() {
		err = rows.Scan(&indexName, &columns)
		if err != nil {
			return "", err
		}
		if columns == search {
			rows.Close()
			return indexName, nil
		}
	}
	rows.Close()
	return "", nil
}

func (m *MysqlDB) CreateIndex(table *schema.Table, index schema.Index) error {
	return nil
}
