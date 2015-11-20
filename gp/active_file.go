package gp

import (
	"go/ast"
	"strings"
)

type ActiveFile struct {
	SrcName string
	AST     *ast.File
}

func (af ActiveFile) DefName() string {
	return strings.Replace(af.SrcName, ".gp", "_def.go", 1)
}
