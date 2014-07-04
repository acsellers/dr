package parse

import (
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
