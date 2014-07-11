package parse

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"regexp"
	"strings"

	"code.google.com/p/go.tools/imports"
)

var (
	subrecordDef = regexp.MustCompile(`type ([a-zA-Z0-9]+) subrecord {`)
	mixinDef     = regexp.MustCompile(`type ([a-zA-Z0-9]+) mixin {`)
	tableDef     = regexp.MustCompile(`type ([a-zA-Z0-9]+) table {`)
)

func (pkg *Package) ParseSrc(src ...*os.File) error {
	fset := token.NewFileSet()

	// process files
	for _, file := range src {
		err := pkg.processFile(fset, file)
		if err != nil {
			return err
		}
	}

	// process mixins
	pkg.exciseMixins()
	for _, table := range pkg.Tables {
		pkg.processForMixins(table)
	}
	for _, subrecord := range pkg.Subrecords {
		pkg.processForMixins(subrecord)
	}

	// Write out processed files
	for _, active := range pkg.ActiveFiles {
		f, err := os.Create(active.DefName())
		if err != nil {
			return err
		}
		b := &bytes.Buffer{}
		format.Node(b, fset, active.AST)
		ib, err := imports.Process(active.DefName(), b.Bytes(), nil)
		if err != nil {
			log.Print(active.DefName())
			log.Fatal(err)
		}
		f.Write(ib)
		f.Close()

	}

	return nil
}

func (pkg *Package) processForMixins(mx Mixinable) {
	if st, ok := mx.Spec().Type.(*ast.StructType); ok {
		fields := []*ast.Field{}
	SRFieldLoop:
		for _, field := range st.Fields.List {
			ft := fmt.Sprint(field.Type)
			for _, mixin := range pkg.Mixins {
				if mixin.Name == ft {
					fields = append(fields, mixin.Fields()...)
					for _, mfunc := range pkg.Funcs[mixin.Name] {
						tident := ast.NewIdent(mx.Name())
						tfunc := &ast.FuncDecl{
							Doc: mfunc.Spec.Doc,
							Recv: &ast.FieldList{
								Opening: mfunc.Spec.Recv.Opening,
								List: []*ast.Field{
									&ast.Field{
										Doc:   mfunc.Spec.Recv.List[0].Doc,
										Names: mfunc.Spec.Recv.List[0].Names,
										Type: &ast.Ident{
											NamePos: mfunc.Spec.Recv.List[0].Type.(*ast.Ident).NamePos,
											Name:    mx.Name(),
											Obj:     tident.Obj,
										},
										Tag:     mfunc.Spec.Recv.List[0].Tag,
										Comment: mfunc.Spec.Recv.List[0].Comment,
									},
								},
								Closing: mfunc.Spec.Recv.Closing,
							},
							Name: mfunc.Spec.Name,
							Type: mfunc.Spec.Type,
							Body: mfunc.Spec.Body,
						}
						mx.File().Decls = append(mx.File().Decls, tfunc)
					}
					continue SRFieldLoop
				}
			}
			fields = append(fields, field)
		}
		st.Fields.List = fields
	}
}

func (pkg *Package) exciseMixins() {
	// remove mixins from file
	for _, mixin := range pkg.Mixins {
		decls := []ast.Decl{}
		for _, decl := range mixin.File.Decls {
			if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
				if td, ok := gd.Specs[0].(*ast.TypeSpec); ok && td == mixin.Spec {
				} else {
					decls = append(decls, decl)
				}
			} else {
				decls = append(decls, decl)
			}
		}
		mixin.File.Decls = decls

		for _, mfunc := range pkg.Funcs[mixin.Name] {
			decls = []ast.Decl{}
			for _, decl := range mfunc.File.Decls {
				if fd, ok := decl.(*ast.FuncDecl); ok {
					if fd != mfunc.Spec {
						decls = append(decls, decl)
					}
				} else {
					decls = append(decls, decl)
				}
			}
			mfunc.File.Decls = decls
		}
	}
}

func (pkg *Package) processFile(fset *token.FileSet, file *os.File) error {
	tables, mixins, subrecords := []string{}, []string{}, []string{}
	output := []string{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()

		switch {
		case subrecordDef.MatchString(text):
			subrecord := subrecordDef.FindStringSubmatch(text)[1]
			subrecords = append(subrecords, subrecord)
			text = strings.Replace(text, " subrecord ", " struct ", 1)
		case mixinDef.MatchString(text):
			mixin := mixinDef.FindStringSubmatch(text)[1]
			mixins = append(mixins, mixin)
			text = strings.Replace(text, " mixin ", " struct ", 1)
		case tableDef.MatchString(text):
			table := tableDef.FindStringSubmatch(text)[1]
			tables = append(tables, table)
			text = strings.Replace(text, " table ", " struct ", 1)
		}

		output = append(output, text)
	}

	fa, err := parser.ParseFile(fset, file.Name(), strings.Join(output, "\n"), parser.ParseComments)
	if err != nil {
		return err
	}

	active := false
DeclLoop:
	for _, decl := range fa.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
			if td, ok := gd.Specs[0].(*ast.TypeSpec); ok {
				name := td.Name.Name
				for _, subrecord := range subrecords {
					if subrecord == name {
						pkg.Subrecords = append(pkg.Subrecords, Subrecord{name, td, fa})
						active = true
						continue DeclLoop
					}
				}
				for _, mixin := range mixins {
					if mixin == name {
						pkg.Mixins = append(pkg.Mixins, Mixin{name, td, fa})
						active = true
						continue DeclLoop
					}
				}
				for _, table := range tables {
					if table == name {
						pkg.Tables = append(pkg.Tables, Table{name, td, fa, []Column{}})
						active = true
						continue DeclLoop
					}
				}
			}
		} else if fd, ok := decl.(*ast.FuncDecl); ok {
			if fd.Recv.NumFields() > 0 {
				name := fmt.Sprint(fd.Recv.List[0].Type)
				pkg.Funcs[name] = append(pkg.Funcs[name], Func{name, fd, fa})
			}
		}
	}
	if active {
		pkg.ActiveFiles = append(pkg.ActiveFiles, ActiveFile{file.Name(), fa})
	}

	return nil
}
