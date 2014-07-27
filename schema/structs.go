package schema

type Schema struct {
	Tables map[string]*Table
	Views  []View
	DBMS   string
}

type Table struct {
	Name    string
	Columns []Column
	Index   []Index

	// One table record has other records in another table relating to it
	HasMany []ManyRelationship
	ChildOf []ManyRelationship

	// One table record may have a single related table record
	HasOne    []OneRelationship
	BelongsTo []OneRelationship

	// Through's will come later
}

func (t *Table) FindColumn(name string) *Column {
	for _, col := range t.Columns {
		if col.Name == name {
			return &col
		}
	}
	return nil
}

func (t *Table) PrimaryKeyColumn() *Column {
	return &t.Columns[0]
}

type Column struct {
	Name   string
	Type   string
	Length int
}

type Index struct {
	Columns []string
	Type    string
}

type View struct {
	SQL string
}

type ManyRelationship struct {
	Parent      *Table
	Child       *Table
	ChildColumn *Column
}

type OneRelationship struct {
	Parent      *Table
	Child       *Table
	ChildColumn *Column
}
