package gp

import "go/ast"

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
