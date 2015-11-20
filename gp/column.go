package gp

import (
	"fmt"
	"go/ast"
	"reflect"
	"strconv"
)

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
		return c.Tag.Get("length") == "" && c.Tag.Get("type") == ""
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
