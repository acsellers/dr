package parse

import (
	"fmt"
	"go/ast"
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
				t.cols = append(t.cols, Column{fmt.Sprint(field.Type), name.Name})
			}
		}
	}
	return t.cols
}

type Column struct {
	Type, Name string
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
