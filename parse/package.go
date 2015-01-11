package parse

import (
	"bytes"
	"fmt"
	"go/ast"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/acsellers/inflections"
	"golang.org/x/tools/imports"
)

type Package struct {
	Tables      []Table
	Mixins      []Mixin
	Subrecords  []Subrecord
	ActiveFiles []ActiveFile
	Funcs       map[string][]Func
	name        *string
}

func (p *Package) Name() string {
	if p.name == nil {
		p.name = &p.ActiveFiles[0].AST.Name.Name
	}
	return *p.name
}

func (p *Package) SetName(n string) {
	p.name = &n
}

func (p Package) TableByName(tableName string) (Table, bool) {
	for _, t := range p.Tables {
		if t.name == tableName {
			return t, true
		}
	}
	return Table{}, false
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

type Index struct {
	Columns []string
}

type Relationship struct {
	Table string
	// One of "ParentHasMany", "ChildHasMany", "HasOne", "BelongsTo"
	Type                  string
	IsArray               bool
	Alias                 string
	Parent                Table
	ParentName, ChildName string
	OperativeColumn       string
}

func (r Relationship) IsHasMany() bool {
	return r.Type == "ParentHasMany"
}

func (r Relationship) IsChildHasMany() bool {
	return r.Type == "ChildHasMany"
}

func (r Relationship) IsHasOne() bool {
	return r.Type == "HasOne"
}

func (r Relationship) IsBelongsTo() bool {
	return r.Type == "BelongsTo"
}

func (r Relationship) ColumnName() string {
	if r.Alias != "" {
		return r.Alias + "ID"
	}
	switch r.Type {
	case "ParentHasMany":
		return r.Parent.name + "ID"
	case "ChildHasMany":
		return r.Table + "ID"
	case "HasOne":
		return r.Parent.name + "ID"
	case "BelongsTo":
		return r.Table + "ID"
	}
	return ""
}

func (r Relationship) Name() string {
	if r.Alias != "" {
		return r.Alias
	}
	return r.Table
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

type Column struct {
	GoType, Name string
	Tag          reflect.StructTag
	Pkg          *Package
	MustNull     bool
	Array        bool
	Tbl          *Table
	IncludeName  string
	// for subrecords
	cols []Column
}

func (c Column) NonZeroCheck() string {
	switch c.GoType {
	case "int", "int32", "int64", "int16":
		return " != 0"
	case "string":
		return ` != ""`
	case "&{time Time}":
		return ".IsZero()"
	case "float32", "float64":
		return " != 0.0"
	case "bool":
		return ""
	default:
		return " != nil"
	}
}

func (c Column) Preset() bool {
	switch c.GoType {
	case "int", "string", "bool", "&{time.Time}", "&{time Time}":
		return c.Tag.Get("length") == ""
	default:
		return false
	}
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
	case "[]byte":
		return true
	default:
		return false
	}
}

func (c Column) Subrecord() *Subrecord {
	if c.SimpleType() {
		return nil
	}
	for _, sr := range c.Pkg.Subrecords {
		if c.GoType == sr.name {
			return &sr
		}
	}
	return nil
}

func (c Column) Subcolumns() chan Column {
	ch := make(chan Column)
	go c.IterateColumns(ch)
	return ch
}

func (c *Column) IterateColumns(ch chan Column) {
	t := c.Subrecord()

	if len(c.cols) == 0 {
		for _, field := range t.spec.Type.(*ast.StructType).Fields.List {
			for _, name := range field.Names {
				var value string
				if field.Tag != nil && len(field.Tag.Value) > 0 {
					// chop off `'s to turn it into a struct tag
					value = field.Tag.Value[1 : len(field.Tag.Value)-1]
				}
				col := Column{
					Name:        name.Name,
					Tag:         reflect.StructTag(value),
					Pkg:         c.Pkg,
					Tbl:         c.Tbl,
					IncludeName: t.Name(),
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
					if sr := col.Subrecord(); sr != nil {
						col.IterateColumns(ch)
					}
				}
				if col.SimpleType() {
					ch <- col
				}
				c.cols = append(c.cols, col)
			}
		}
	}

	close(ch)
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
	case "[]byte":
		return "blob"
	default:
		return "varchar"
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

func (s *Subrecord) AddRetrieved() {

}

func (t Subrecord) Name() string {
	if t.name == "" {
		return t.spec.Name.Name
	}
	return t.name
}
func (t Subrecord) Spec() *ast.TypeSpec {
	return t.spec
}
func (t Subrecord) File() *ast.File {
	return t.file
}

var tmpl *template.Template

func init() {
	rg := regexp.MustCompile(`^[A-Z].*`)
	var err error
	tmpl, err = template.New("dr").
		Funcs(template.FuncMap{
		"plural": inflections.Pluralize,
		"public": func(s string) bool {
			return rg.MatchString(s)
		},
	}).
		New("gen").Parse(genTemplate)
	if err != nil {
		panic(err)
	}

	tmpl, err = tmpl.New("schema").Parse(schemaTemplate)
	if err != nil {
		panic(err)
	}

	tmpl, err = tmpl.New("lib").Parse(libTemplate)
	if err != nil {
		panic(err)
	}

}

func (pkg *Package) OutputTemplates() {
	b := &bytes.Buffer{}
	err := tmpl.ExecuteTemplate(b, "gen", pkg)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(pkg.ActiveFiles[0].AST.Name.Name + "_gen.go")
	if err != nil {
		fmt.Println("Could not write schema file")
	}
	defer f.Close()

	ib, err := imports.Process(pkg.ActiveFiles[0].AST.Name.Name+"_gen.go", b.Bytes(), nil)
	if err != nil {
		fmt.Println("Error in Gen File:", err)
		f.Write(b.Bytes())
		return
	}
	f.Write(ib)

	w, err := os.Create(pkg.ActiveFiles[0].AST.Name.Name + "_schema.go")
	if err != nil {
		panic("Couldn't open schema file for writing: " + err.Error())
	}
	b = &bytes.Buffer{}
	err = tmpl.ExecuteTemplate(b, "schema", pkg)
	if err != nil {
		panic(err)
	}
	ib, err = imports.Process(pkg.ActiveFiles[0].AST.Name.Name+"_schema.go", b.Bytes(), nil)
	if err != nil {
		fmt.Println("Error in Gen File:", err)
		w.Write(b.Bytes())
		return
	}
	w.Write(ib)

	pkg.WriteLibraryFiles()
}

func (pkg *Package) WriteLibraryFiles() {
	if _, err := os.Stat(pkg.Name() + "_lib.go"); err == nil {
		return
	}
	b := &bytes.Buffer{}
	err := tmpl.ExecuteTemplate(b, "lib", pkg)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(pkg.Name() + "_lib.go")
	if err != nil {
		fmt.Println("Could not write schema file")
	}

	ib, err := imports.Process(pkg.Name()+"_lib.go", b.Bytes(), nil)
	if err != nil {
		fmt.Println("Error in Gen File:", err)
		f.Write(b.Bytes())
		f.Close()
		return
	}
	f.Write(ib)
	f.Close()

	filename := "db_config.go"
	if _, err = os.Stat(filename); err == nil {
		// fmt.Println("Library file already written")
		return
	}
	b = &bytes.Buffer{}
	err = tmpl.ExecuteTemplate(b, "config", pkg)
	if err != nil {
		panic(err)
	}

	f, err = os.Create(filename)
	if err != nil {
		fmt.Println("Could not write schema file")
	}

	ib, err = imports.Process(filename, b.Bytes(), nil)
	if err != nil {
		fmt.Println("Error in Gen File:", err)
		f.Write(b.Bytes())
		f.Close()
		return
	}
	f.Write(ib)
	f.Close()

}

func (pkg *Package) WriteStarterFile() {
	if _, err := os.Stat(pkg.Name() + ".gp"); err == nil {
		fmt.Println("Starter file already written")
		return
	}

	f, err := os.Create(pkg.Name() + ".gp")
	if err != nil {
		fmt.Println("Could not write schema file")
	}
	defer f.Close()

	err = tmpl.ExecuteTemplate(f, "starter_file", pkg)
	if err != nil {
		panic(err)
	}
}
