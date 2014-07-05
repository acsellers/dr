package parse

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
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
	pkg.WriteSchema()

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
						pkg.Tables = append(pkg.Tables, Table{name, td, fa})
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

const (
	GenerateWarning = `/*
  This code was generated from the Doc ORM Generator and isn't meant to be edited.
	If at all possible, please regenerate this file from your gp files instead of 
	trying to edit it.
*/`
	BasicScopes = `
// Basic conditions
Eq(val interface{}) %[1]sScope
Neq(val interface{}) %[1]sScope
Gt(val interface{}) %[1]sScope
Gte(val interface{}) %[1]sScope
Lt(val interface{}) %[1]sScope
Lte(val interface{}) %[1]sScope

// multi value conditions
Between(lower, upper interface{}) %[1]sScope
In(vals ...interface{}) %[1]sScope
NotIn(vals ...interface{}) %[1]sScope
Where(sql string, vals ...interface{}) %[1]sScope

// ordering conditions
Order(ordering string) %[1]sScope
Desc() %[1]sScope
Asc() %[1]sScope

// Aggregation filtering
Having(sql string, vals ...interface{}) %[1]sScope
// GroupBy(???) %[1]sScope

// Result count filtering
Limit(limit int64) %[1]sScope
Offset(offset int64) %[1]sScope

// Misc. Scope operations
Clear() %[1]sScope
ClearAll() %[1]sScope
Base() %[1]sScope

// Struct instance saving and loading
Find(id interface{}) %[1]s
Retrieve() %[1]s
RetrieveAll() []%[1]s
SaveAll(vals []%[1]s) error

// Subset plucking
PluckString() ([]string, error)
PluckInt() ([]int64, error)
PluckTime() ([]time.Time, error)
PluckStruct(result interface{}) error

// Direct SQL operations
Count() int64
CountBy(sql string) int64
UpdateSQL(sql string, vals ...interface{}) error
Delete() error

// Special operations
ToSQL() (string, []interface{})
As(alias string) %[1]sScope
`
)

func (pkg *Package) WriteSchema() {
	b := &bytes.Buffer{}
	io.WriteString(b, GenerateWarning)
	fmt.Fprintf(b, "\n\npackage %s\n", pkg.ActiveFiles[0].AST.Name.Name)

	pkg.WriteTableScopes(b)
	pkg.WriteConnDefinition(b)
	pkg.WriteTableScopeStructs(b)
	pkg.WriteDocThings(b)

	f, err := os.Create(pkg.ActiveFiles[0].AST.Name.Name + "_gen.go")
	if err != nil {
		fmt.Println("Could not write schema file")
	}
	defer f.Close()

	ib, err := imports.Process(pkg.ActiveFiles[0].AST.Name.Name+"_gen.go", b.Bytes(), nil)
	if err != nil {
		f.Write(b.Bytes())
		log.Fatal(err)
	}

	f.Write(ib)
}

// WriteTableScopes writes out the scope definition for each
// of the tables. Some of the scopes are from columsn and others
// are from the base scopes of doc.
func (pkg *Package) WriteTableScopes(f io.Writer) {
	for _, table := range pkg.Tables {
		fmt.Fprintf(f, "type %sScope interface {\n// column scopes\n", table.Name())
		for _, field := range table.Spec().Type.(*ast.StructType).Fields.List {
			for _, name := range field.Names {
				fmt.Fprintf(f, "\t%s() %sScope\n", name.Name, table.Name())
			}
		}
		fmt.Fprintf(f, BasicScopes, table.Name())
		fmt.Fprintln(f, "}\n\n")
	}
}

// WriteConnDefinition will build a conn struct that the user can use
// to access the scopes.
func (pkg *Package) WriteConnDefinition(f io.Writer) {
	io.WriteString(f, "type Conn struct {\n*sql.DB\n")
	for _, table := range pkg.Tables {
		fmt.Fprintf(f, "\t%[1]s  %[1]sScope\n", table.Name())
	}
	io.WriteString(f, "}\n\n")

	io.WriteString(f, "func Open(dataSourceName string) (*Conn, error) {\nc := &Conn{}\n")
	for _, table := range pkg.Tables {
		fmt.Fprintf(f, "c.%[1]s = new%[1]sScope(c)\n", table.Name())
	}
	io.WriteString(f, "return c, nil\n}\n")
	io.WriteString(f, `func (c *Conn) Close() error {
		return c.DB.Close()
	}

	func (c *Conn) SQLTable(table interface{}) string {
		return strings.ToLower(reflect.ValueOf(table).Type().Name())
	}

	func (c *Conn) SQLColumn(column string) string {
		return strings.ToLower(column)
	}
`)
}

const (
	baseScopeDef = `// basic conditions
func (scope scope%[1]s) Eq(val interface{}) %[1]sScope {
	c := condition{column: scope.currentColumn}
	if val == nil {
		c.cond = "IS NULL"
	} else {
		c.cond = "= ?"
		c.vals = []interface{}{val}
	}

	scope.conditions = append(scope.conditions, c)
	return scope
}

func (scope scope%[1]s) Neq(val interface{}) %[1]sScope {
	c := condition{column: scope.currentColumn}
	if val == nil {
		c.cond = "IS NOT NULL"
	} else {
		c.cond = "<> ?"
		c.vals = []interface{}{val}
	}

	scope.conditions = append(scope.conditions, c)
	return scope
}

func (scope scope%[1]s) Gt(val interface{}) %[1]sScope {
	c := condition{
		column: scope.currentColumn,
		cond: "> ?",
		vals: []interface{}{val},
	}

	scope.conditions = append(scope.conditions, c)
	return scope
}

func (scope scope%[1]s) Gte(val interface{}) %[1]sScope {
	c := condition{
		column: scope.currentColumn,
		cond: ">= ?",
		vals: []interface{}{val},
	}

	scope.conditions = append(scope.conditions, c)
	return scope
}

func (scope scope%[1]s) Lt(val interface{}) %[1]sScope {
	c := condition{
		column: scope.currentColumn,
		cond: "< ?",
		vals: []interface{}{val},
	}

	scope.conditions = append(scope.conditions, c)
	return scope
}

func (scope scope%[1]s) Lte(val interface{}) %[1]sScope {

	c := condition{
		column: scope.currentColumn,
		cond: "<= ?",
		vals: []interface{}{val},
	}

	scope.conditions = append(scope.conditions, c)
	return scope
}

// multi value conditions
func (scope scope%[1]s) Between(lower, upper interface{}) %[1]sScope {
	c := condition{
		column: scope.currentColumn,
		cond: "BETWEEN ? AND ?",
		vals: []interface{}{lower, upper},
	}

	scope.conditions = append(scope.conditions, c)
	return scope
}

func (scope scope%[1]s) In(vals ...interface{}) %[1]sScope {
	vc := make([]string, len(vals))
	c := condition{
		column: scope.currentColumn,
		cond: "IN ("+strings.Join(vc, "?, ")+"?)",
		vals: vals,
	}

	scope.conditions = append(scope.conditions, c)
	return scope
}

func (scope scope%[1]s) NotIn(vals ...interface{}) %[1]sScope {
	vc := make([]string, len(vals))
	c := condition{
		column: scope.currentColumn,
		cond: fmt.Sprintf("NOT IN (%s?)", strings.Join(vc, "?, ")),
		vals: vals,
	}

	scope.conditions = append(scope.conditions, c)
	return scope
}

func (scope scope%[1]s) Where(sql string, vals ...interface{}) %[1]sScope {
	c := condition{
		cond: sql,
		vals: vals,
	}
	scope.conditions = append(scope.conditions, c)
	return scope
}

// ordering conditions
func (scope scope%[1]s) Order(ordering string) %[1]sScope {
	scope.order= append(scope.order, ordering)
	return scope
}

func (scope scope%[1]s) Desc() %[1]sScope {
	scope.order= append(scope.order, scope.currentColumn + " DESC")
	return scope
}

func (scope scope%[1]s) Asc() %[1]sScope {
	scope.order= append(scope.order, scope.currentColumn + " ASC")
	return scope
}

// aggregation filtering
func (scope scope%[1]s) Having(sql string, vals ...interface{}) %[1]sScope {
	scope.having = append(scope.having, sql)
	scope.havevals = append(scope.havevals, vals...)
	return scope
}

/*
	func (scope %[1]sScope) GroupBy(???) %[1]Scope {
		return scope
	}
*/

// Result count filtering
func (scope scope%[1]s) Limit(limit int64) %[1]sScope {
	scope.limit = &limit
	return scope
}

func (scope scope%[1]s) Offset(offset int64) %[1]sScope {
	scope.offset = &offset
	return scope
}

// misc scope operations
func (scope scope%[1]s) Clear() %[1]sScope {
	return scope
}

func (scope scope%[1]s) ClearAll() %[1]sScope {
	scope.conditions = []condition{}
	return scope
}

func (scope scope%[1]s) Base() %[1]sScope {
	return new%[1]sScope(scope.conn)
}

// struct saving and loading
func (scope scope%[1]s) Find(id interface{}) %[1]s {
	return %[1]s{}
}

func (scope scope%[1]s) Retrieve() %[1]s {
	return %[1]s{}
}

func (scope scope%[1]s) RetrieveAll() []%[1]s {
	return []%[1]s{}
}

func (scope scope%[1]s) SaveAll(vals []%[1]s) error {
	return nil
}

// subset plucking
func (scope scope%[1]s)PluckString() ([]string, error) {
	return []string{}, nil
}

func (scope scope%[1]s)PluckInt() ([]int64, error) {
	return []int64{}, nil
}

func (scope scope%[1]s)PluckTime() ([]time.Time, error) {
	return []time.Time{}, nil
}

func (scope scope%[1]s)PluckStruct(result interface{}) error {
	return nil
}

// direct sql
func (scope scope%[1]s) Count() int64 {
	return 0
}

func (scope scope%[1]s) CountBy(sql string) int64 {
	return 0
}

func (scope scope%[1]s) UpdateSQL(sql string, vals ...interface{}) error {
	return nil
}

func (scope scope%[1]s) Delete() error {
	return nil
}

// special
func (scope scope%[1]s) ToSQL() (string, []interface{}) {
	return scope.query()
}

func (scope scope%[1]s) As(alias string) %[1]sScope {
	scope.currentAlias = alias
	return scope
}
`
	columnScopeDef = `
func (scope scope%[1]s) %[2]s() %[1]sScope {
	scope.currentColumn = 
	scope.conn.SQLTable(%[1]s{})+
	"."+
	scope.conn.SQLColumn("%[2]s")
	scope.currentAlias = ""
	return scope
}
`
)

const tableScope = `type scope%[1]s struct {
	conn *Conn
	table string
	columns []string
	order []string
	joins []string
	conditions []condition
	having []string
	havevals []interface{}
	currentColumn, currentAlias string
	limit, offset *int64
}

func new%[1]sScope(c *Conn) *scope%[1]s {
	return &scope%[1]s{
		conn: c,
		table: c.SQLTable(%[1]s{}),
		currentColumn: c.SQLTable(%[1]s{})+"."+c.SQLColumn("%[2]s"),
	}
}

func (s *scope%[1]s) query() (string, []interface{}) {
	// SELECT (columns) FROM (table) (joins) WHERE (conditions) 
	// GROUP BY (grouping) HAVING (havings)
	// ORDER BY (orderings) LIMIT (limit) OFFSET (offset)
	sql := []string{}
	vals := []interface{}{}
	if len(s.columns) == 0 {
		sql = append(sql, "SELECT", s.table+".*")
	} else {
		sql = append(sql, "SELECT", strings.Join(s.columns, ", "))
	}
	// if s.source == nil { // subquery
	// 
	// } else {
	sql = append(sql, "FROM", s.table)
	// }
	sql = append(sql, s.joins...)

	if len(s.conditions) > 0 {
		sql = append(sql, "WHERE")
		for _, condition := range s.conditions {
			sql = append(sql, condition.ToSQL())
			vals = append(vals, condition.vals...)
		}
	}

	// if len(s.groupings) > 0 {
	//   sql = append(sql , "GROUP BY")
	//   for _, grouping := range s.groupings {
	//     sql = append(sql, grouping.ToSQL()
	//   }
	// }

	if len(s.having) > 0 {
		sql = append(sql, "HAVING")
		sql = append(sql, s.having...)
		vals = append(vals, s.havevals...)
	}

	if len(s.order) > 0 {
		sql = append(sql, "ORDER BY")
		sql = append(sql, s.order...)
	}

	if s.limit != nil {
		sql = append(sql, "LIMIT", fmt.Sprint("%v", *s.limit))
	}

	if s.offset != nil {
		sql = append(sql, "OFFSET", fmt.Sprint("%v", *s.offset))
	}

	return strings.Join(sql, " "), vals
}
`

// WriteTableScopeStructs writes out the structs and functions for each
// table that actually implement the scope interfaces.
func (pkg *Package) WriteTableScopeStructs(f io.Writer) {
	for _, table := range pkg.Tables {
		fmt.Fprintf(f,
			tableScope,
			table.Name(),
			table.Spec().Type.(*ast.StructType).Fields.List[0].Names[0].Name,
		)
		fmt.Fprintf(f, baseScopeDef, table.Name())
		for _, field := range table.Spec().Type.(*ast.StructType).Fields.List {
			for _, name := range field.Names {
				fmt.Fprintf(f, columnScopeDef, table.Name(), name.Name)
			}
		}
	}
}

func (pkg *Package) WriteDocThings(f io.Writer) {
	io.WriteString(f, `type condition struct {
		column string
		cond string
		vals []interface{}
	}
	func (c condition) ToSQL() string {
		return c.column+" "+c.cond
	}
`)
}
