package gp

import (
	"fmt"
	"go/ast"
	"reflect"
)

type Table struct {
	name      string
	spec      *ast.TypeSpec
	file      *ast.File
	cols      []Column
	Pkg       *Package
	Relations []Relationship
	Indexes   []Index
}

func (t Table) ColumnByName(colName string) (Column, bool) {
	for _, col := range t.Columns() {
		if col.Name == colName {
			return col, true
		}
	}
	return Column{}, false
}

func (t Table) Name() string {
	return t.name
}
func (t Table) Spec() *ast.TypeSpec {
	return t.spec
}
func (t Table) File() *ast.File {
	return t.file
}

func (t *Table) Columns() []Column {
	if len(t.cols) == 0 {
		for _, field := range t.spec.Type.(*ast.StructType).Fields.List {
			if len(field.Names) == 0 {
				col := Column{
					Name:   "",
					Pkg:    t.Pkg,
					Tbl:    t,
					GoType: fmt.Sprint(field.Type),
				}
				if field.Tag != nil && len(field.Tag.Value) > 0 {
					col.Tag = reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
				}
				if se, ok := field.Type.(*ast.StarExpr); ok {

					col.MustNull = true
					col.GoType = fmt.Sprint(se.X)
				}
				t.cols = append(t.cols, col)
			}
			for _, name := range field.Names {
				var value string
				if field.Tag != nil && len(field.Tag.Value) > 0 {
					// chop off `'s to turn it into a struct tag
					value = field.Tag.Value[1 : len(field.Tag.Value)-1]
				}
				col := Column{
					Name: name.Name,
					Tag:  reflect.StructTag(value),
					Pkg:  t.Pkg,
					Tbl:  t,
				}
				col.GoType = fmt.Sprint(field.Type)
				switch at := field.Type.(type) {
				case *ast.StarExpr:
					col.MustNull = true
					col.GoType = fmt.Sprint(at.X)
				case *ast.ArrayType:
					col.Array = true
					col.GoType = "[]" + fmt.Sprint(at.Elt)
				}
				t.cols = append(t.cols, col)
			}
		}
	}
	return t.cols
}

func (t Table) RelationshipTo(tableName string) (Relationship, bool) {
	for _, relate := range t.Relations {
		if relate.Table == tableName {
			return relate, true
		}
	}
	return Relationship{}, false
}

func (t *Table) PrimaryKeyColumn() Column {
	return t.Columns()[0]
}

func (t Table) HasRelationship(relate string) bool {
	for _, relation := range t.Relations {
		if relation.Type == relate {
			return true
		}
	}
	return false
}
