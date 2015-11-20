package gp

import "go/ast"

type Func struct {
	Name string
	Spec *ast.FuncDecl
	File *ast.File
}
