package gp

import "go/ast"

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

type Mixinable interface {
	Name() string
	Spec() *ast.TypeSpec
	File() *ast.File
}
