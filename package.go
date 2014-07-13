package parse

import (
	"fmt"
	"go/ast"
	"reflect"
	"strconv"
	"strings"
)

type Package struct {
	Tables      []Table
	Mixins      []Mixin
	Subrecords  []Subrecord
	ActiveFiles []ActiveFile
	Funcs       map[string][]Func
}

func (p Package) Name() string {
	return p.ActiveFiles[0].AST.Name.Name
}

type ActiveFile struct {
	SrcName string
	AST     *ast.File
}

func (af ActiveFile) DefName() string {
	return strings.Replace(af.SrcName, ".gp", "_def.go", 1)
}

type Mixin struct {
	Name string
	Spec *ast.TypeSpec
	File *ast.File
}

func (m Mixin) Fields() []*ast.Field {
	if st, ok := m.Spec.Type.(*ast.StructType); ok {
		return st.Fields.List
	}
	return []*ast.Field{}
}

type Func struct {
	Name string
	Spec *ast.FuncDecl
	File *ast.File
}

type Mixinable interface {
	Name() string
	Spec() *ast.TypeSpec
	File() *ast.File
}

type Table struct {
	name string
	spec *ast.TypeSpec
	file *ast.File
	cols []Column
	Pkg  *Package
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
					col.GoType = fmt.Sprint(at.Elt)
				default:
					if !col.SimpleType() {
						fmt.Println(fmt.Sprint(field.Type))
					}
				}
				t.cols = append(t.cols, col)
			}
		}
	}
	return t.cols
}

func (t *Table) PrimaryKeyColumn() Column {
	return t.Columns()[0]
}

type Column struct {
	GoType, Name string
	Tag          reflect.StructTag
	Pkg          *Package
	MustNull     bool
	Array        bool
	ParentCol    *Column
	ChildHasMany bool
	Tbl          *Table
}

func (c Column) SimpleType() bool {
	switch c.GoType {
	case "int", "int32", "int64", "int16":
		return true
	case "string":
		return true
	case "&{time Time}":
		return true
	case "float32", "float64":
		return true
	case "bool":
		return true
	default:
		return false
	}
}

func (c Column) Type() string {
	switch c.GoType {
	case "int":
		return "integer"
	case "string":
		if c.Tag.Get("type") == "text" {
			return "text"
		}
		return "varchar"
	case "&{time Time}":
		return "timestamp"
	case "float32":
		return "real"
	case "float64":
		return "double precision"
	case "bool":
		return "boolean"
	default:
		return "varchar"
	}
	return ""
}

func (c Column) IsHasMany() bool {
	if !c.SimpleType() {
		return c.Array && c.Tag.Get("through") == ""
	}
	return false
}

func (c Column) IsChildHasMany() bool {
	return c.ParentCol != nil && c.ChildHasMany
}

func (c Column) IsHasOne() bool {
	if !c.SimpleType() {
		return !c.Array && c.Tag.Get("through") == ""
	}
	return false
}

func (c Column) IsBelongsTo() bool {
	return c.ParentCol != nil && !c.ChildHasMany
}
func (c Column) IsHasManyThrough() bool {
	if !c.SimpleType() {
		return c.Array && c.Tag.Get("through") != ""
	}
	return false
}

func (c Column) ChildColumn() string {
	if c.Tag.Get("column") != "" {
		colname := c.Tag.Get("column")
		for _, tbl := range c.Pkg.Tables {
			if tbl.Name() == c.GoType {
				for i, col := range tbl.cols {
					if col.Name == colname {
						col.ParentCol = &c
						if c.Array {
							col.ChildHasMany = true
						}
						tbl.cols[i] = col
						return colname
					}

				}
			}
		}
	}
	for _, tbl := range c.Pkg.Tables {
		if tbl.Name() == c.GoType {
			for i, col := range tbl.cols {
				if strings.HasPrefix(col.Name, c.Tbl.Name()) && col.GoType != c.Tbl.Name() {
					col.ParentCol = &c
					if c.Array {
						col.ChildHasMany = true
					}
					tbl.cols[i] = col

					return col.Name
				}
			}
		}
	}
	return ""
}

func (c Column) Length() int {
	if s := c.Tag.Get("length"); s != "" {
		l, err := strconv.ParseInt(s, 10, 32)
		if err == nil {
			return int(l)
		}
	}
	switch c.GoType {
	case "int":
		return 10
	case "string":
		if c.Tag.Get("type") == "text" {
			return 0
		}
		return 255
	default:
		return 0
	}
}

type Subrecord struct {
	name string
	spec *ast.TypeSpec
	file *ast.File
}

func (t Subrecord) Name() string {
	return t.name
}
func (t Subrecord) Spec() *ast.TypeSpec {
	return t.spec
}
func (t Subrecord) File() *ast.File {
	return t.file
}
