package parse

var libTemplate = `package {{ .Name }}

//go:generate dr build

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
	"github.com/acsellers/dr/schema"
)

type Scope interface {
	condSQL() (string, []interface{})
	QuerySQL() (string, []interface{})
	UpdateSQL() (string, []interface{})
	DeleteSQL() (string, []interface{})
	scopeName() string
	Conn() *Conn
	SetConn(*Conn) Scope
	joinOn(string, Scope) (string, bool)
	joinable() string
	joinTable() string
	conds() []condition
}

func (c *Conn) Exec(query string, args ...interface{}) (sql.Result, error) {
	if c.Log != nil {
		c.Log.Printf("%s %v", query, args)
	}
	return c.DB.Exec(c.FormatQuery(query), args...)
}

func (c *Conn) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if c.Log != nil {
		c.Log.Printf("%s %v", query, args)
	}
	return c.DB.Query(c.FormatQuery(query), args...)
}

func (c *Conn) QueryRow(query string, args ...interface{}) *sql.Row {
	if c.Log != nil {
		c.Log.Printf("%s %v", query, args)
	}
	return c.DB.QueryRow(c.FormatQuery(query), args...)
}

func (c *Conn) FormatQuery(query string) string {
	if !c.reformat {
		return query
	}

	parts := strings.Split(query, "?")
	var newQuery []string
	for i, part := range parts[:len(parts)-1] {
		newQuery = append(newQuery, fmt.Sprintf("%s$%d", part, i+1))
	}
	newQuery = append(newQuery, parts[len(parts)-1])

	return strings.Join(newQuery, "")
}

func (c *Conn) Close() error {
	return c.DB.Close()
}

type internalScope struct {
	conn                        *Conn
	table                       string
	columns                     []string
	order                       []string
	joins                       []string
	joinedScopes                []Scope
	includes                    []string
	conditions                  []condition
	having                      []string
	havevals                    []interface{}
	groupBy                     []string
	currentColumn, currentAlias string
	isDistinct                  bool
	limit, offset               *int64
	updates                     map[string]interface{}
}

func (scope internalScope) Conn() *Conn {
	return scope.conn
}

func (scope internalScope) conds() []condition {
	return scope.conditions
}

func (s *internalScope) query() (string, []interface{}) {
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
		cs, cv := s.conditionSQL()
		sql = append(sql, "WHERE", cs)
		vals = append(vals, cv...)
	}

	// if len(s.groupings) > 0 {
	//   sql = append(sql , "GROUP BY")
	//   for _, grouping := range s.groupings {
	//     sql = append(sql, grouping.QuerySQL()
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
		sql = append(sql, "LIMIT", fmt.Sprintf("%v", *s.limit))
	}

	if s.offset != nil {
		sql = append(sql, "OFFSET", fmt.Sprintf("%v", *s.offset))
	}

	return strings.Join(sql, " "), vals
}

func (scope internalScope) conditionSQL() (string, []interface{}) {
	var vals []interface{}
	conds := []string{}
	for _, condition := range scope.conditions {
		conds = append(conds, condition.ToSQL())
		vals = append(vals, condition.vals...)
	}
	return strings.Join(conds, " AND "), vals
}

func (scope internalScope) Eq(val interface{}) internalScope {
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

func (scope internalScope) Neq(val interface{}) internalScope {
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

func (scope internalScope) Gt(val interface{}) internalScope {
	c := condition{
		column: scope.currentColumn,
		cond:   "> ?",
		vals:   []interface{}{val},
	}

	scope.conditions = append(scope.conditions, c)
	return scope
}

func (scope internalScope) Gte(val interface{}) internalScope {
	c := condition{
		column: scope.currentColumn,
		cond:   ">= ?",
		vals:   []interface{}{val},
	}

	scope.conditions = append(scope.conditions, c)
	return scope
}

func (scope internalScope) Lt(val interface{}) internalScope {
	c := condition{
		column: scope.currentColumn,
		cond:   "< ?",
		vals:   []interface{}{val},
	}

	scope.conditions = append(scope.conditions, c)
	return scope
}

func (scope internalScope) Lte(val interface{}) internalScope {

	c := condition{
		column: scope.currentColumn,
		cond:   "<= ?",
		vals:   []interface{}{val},
	}

	scope.conditions = append(scope.conditions, c)
	return scope
}

// multi value conditions
func (scope internalScope) Between(lower, upper interface{}) internalScope {
	c := condition{
		column: scope.currentColumn,
		cond:   "BETWEEN ? AND ?",
		vals:   []interface{}{lower, upper},
	}

	scope.conditions = append(scope.conditions, c)
	return scope
}

func (scope internalScope) In(vals ...interface{}) internalScope {
	if len(vals) == 0 {
		if reflect.TypeOf(vals[0]).Kind() == reflect.Slice {
			rv := reflect.ValueOf(vals[0])
			vals = make([]interface{}, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				vals[i] = rv.Index(i).Interface()
			}
		}
	}

	vc := make([]string, len(vals))
	c := condition{
		column: scope.currentColumn,
		cond:   "IN (" + strings.Join(vc, "?, ") + "?)",
		vals:   vals,
	}

	scope.conditions = append(scope.conditions, c)
	return scope
}

func (scope internalScope) NotIn(vals ...interface{}) internalScope {
	vc := make([]string, len(vals))
	c := condition{
		column: scope.currentColumn,
		cond:   "NOT IN (" + strings.Join(vc, "?, ") + "?)",
		vals:   vals,
	}

	scope.conditions = append(scope.conditions, c)
	return scope
}

func (scope internalScope) Like(str string) internalScope {
	c := condition{
		column: scope.currentColumn,
		cond:   "LIKE ?",
		vals:   []interface{}{str},
	}

	scope.conditions = append(scope.conditions, c)
	return scope
}

func (scope internalScope) Where(sql string, vals ...interface{}) internalScope {
	c := condition{
		cond: sql,
		vals: vals,
	}
	scope.conditions = append(scope.conditions, c)
	return scope
}


func (scope internalScope)	outerJoin(name string, things ...Scope) internalScope {
	for _, thing := range things {
		thing = thing.SetConn(scope.conn)
		if joinString, ok := scope.joinOn(name, thing); ok {
			scope.joins = append(scope.joins, fmt.Sprintf(
				"LEFT JOIN %s ON %s",
				thing.joinable(),
				joinString, 
			))
			scope.joinedScopes = append(scope.joinedScopes, thing)
			scope.conditions = append(scope.conditions, thing.conds()...)
			continue
		} else {
			for _, joinscope := range scope.joinedScopes {
				if joinString, ok := joinscope.joinOn(name, thing); ok {
					scope.joins = append(scope.joins, fmt.Sprintf(
						"LEFT JOIN %s ON %s",
						thing.joinable(),
						joinString, 
					))
					scope.joinedScopes = append(scope.joinedScopes, thing)
					scope.conditions = append(scope.conditions, thing.conds()...)
					continue		
				}
			}
		}
		// error
	}
	return scope
}

func (scope internalScope)	innerJoin(name string, things ...Scope) internalScope {
	for _, thing := range things {
		thing = thing.SetConn(scope.conn)
		if joinString, ok := scope.joinOn(name, thing); ok {
			scope.joins = append(scope.joins, fmt.Sprintf(
				"INNER JOIN %s ON %s",
				thing.joinable(),
				joinString, 
			))
			scope.conditions = append(scope.conditions, thing.conds()...)
			scope.joinedScopes = append(scope.joinedScopes, thing)
			continue
		} else {
			for _, joinscope := range scope.joinedScopes {
				if joinString, ok := joinscope.joinOn(name, thing); ok {
					scope.joins = append(scope.joins, fmt.Sprintf(
						"INNER JOIN %s ON %s",
						thing.joinable(),
						joinString, 
					))
					scope.joinedScopes = append(scope.joinedScopes, thing)
					scope.conditions = append(scope.conditions, thing.conds()...)
					continue		
				}
			}
		}
		// error
	}

	return scope
}

func (scope internalScope) joinOn(name string, joinee Scope) (string, bool) {
	ts := Schema.Tables[name]
	for _, hm := range ts.HasMany {
		if (hm.Parent.Name == name && hm.Child.Name == joinee.scopeName()) || hm.Parent.Name == name && hm.Child.Name == joinee.scopeName() {
			pkc := hm.Parent.PrimaryKeyColumn()
			return fmt.Sprintf(
				"%s.%s = %s.%s",
				scope.conn.SQLTable(hm.Parent.Name),
				scope.conn.SQLColumn(hm.Parent.Name, pkc.Name),
				scope.conn.SQLTable(hm.Child.Name),
				scope.conn.SQLColumn(hm.Child.Name, hm.ChildColumn.Name),
			), true
		}
	}
	for _, hm := range ts.ChildOf {
		if (hm.Parent.Name == name && hm.Child.Name == joinee.scopeName()) || hm.Parent.Name == name && hm.Child.Name == joinee.scopeName() {
			pkc := hm.Parent.PrimaryKeyColumn()
			return fmt.Sprintf(
				"%s.%s = %s.%s",
				scope.conn.SQLTable(hm.Parent.Name),
				scope.conn.SQLColumn(hm.Parent.Name, pkc.Name),
				scope.conn.SQLTable(hm.Child.Name),
				scope.conn.SQLColumn(hm.Child.Name, hm.ChildColumn.Name),
			), true
		}
	}
	for _, hm := range ts.HasOne {
		if (hm.Parent.Name == name && hm.Child.Name == joinee.scopeName()) || hm.Parent.Name == name && hm.Child.Name == joinee.scopeName() {
			pkc := hm.Parent.PrimaryKeyColumn()
			return fmt.Sprintf(
				"%s.%s = %s.%s",
				scope.conn.SQLTable(hm.Parent.Name),
				scope.conn.SQLColumn(hm.Parent.Name, pkc.Name),
				scope.conn.SQLTable(hm.Child.Name),
				scope.conn.SQLColumn(hm.Child.Name, hm.ChildColumn.Name),
			), true
		}
	}
	for _, hm := range ts.BelongsTo {
		if (hm.Parent.Name == name && hm.Child.Name == joinee.scopeName()) || hm.Parent.Name == name && hm.Child.Name == joinee.scopeName() {
			pkc := hm.Parent.PrimaryKeyColumn()
			return fmt.Sprintf(
				"%s.%s = %s.%s",
				scope.conn.SQLTable(hm.Parent.Name),
				scope.conn.SQLColumn(hm.Parent.Name, pkc.Name),
				scope.conn.SQLTable(hm.Child.Name),
				scope.conn.SQLColumn(hm.Child.Name, hm.ChildColumn.Name),
			), true
		}
	}
	return "", false
}

func (scope internalScope) PluckString() ([]string, error) {
	if scope.isDistinct{
		scope.currentColumn = "DISTINCT " + scope.currentColumn
	}
	scope.columns = []string{scope.currentColumn}
	ss, vv := scope.query()
	rows, err := scope.conn.Query(ss, vv...)
	if err != nil {
		return []string{}, err
	}
	vals := []string{}
	defer rows.Close()
	for rows.Next() {
		var temp string
		err = rows.Scan(&temp)
		if err != nil {
			return []string{}, err
		}
		vals = append(vals, temp)
	}

	return vals, nil
}

func (scope internalScope) PluckInt() ([]int64, error) {
	if scope.isDistinct{
		scope.currentColumn = "DISTINCT " + scope.currentColumn
	}

	scope.columns = []string{scope.currentColumn}
	ss, vv := scope.query()
	rows, err := scope.conn.Query(ss, vv...)
	if err != nil {
		return []int64{}, err
	}
	vals := []int64{}
	defer rows.Close()
	for rows.Next() {
		var temp int64
		err = rows.Scan(&temp)
		if err != nil {
			return []int64{}, err
		}
		vals = append(vals, temp)
	}

	return vals, nil
}

func (scope internalScope) PluckTime() ([]time.Time, error) {
	if scope.isDistinct{
		scope.currentColumn = "DISTINCT " + scope.currentColumn
	}

	scope.columns = []string{scope.currentColumn}
	ss, vv := scope.query()
	rows, err := scope.conn.Query(ss, vv...)
	if err != nil {
		return []time.Time{}, err
	}
	vals := []time.Time{}
	defer rows.Close()
	for rows.Next() {
		var temp time.Time
		err = rows.Scan(&temp)
		if err != nil {
			return []time.Time{}, err
		}
		vals = append(vals, temp)
	}

	return vals, nil
}

func (scope internalScope) pluckStruct(name string, result interface{}) error {
	destSlice := reflect.ValueOf(result).Elem()
	tempSlice := reflect.Zero(destSlice.Type())
	elem := destSlice.Type().Elem()
	vn := reflect.New(elem)
	rfltr := reflector{vn}
	p := &planner{[]*reflectScanner{}}

	for i := 0; i < elem.NumField(); i++ {
		f := elem.Field(i)
		if f.Tag.Get("column") != "" {
			scope.columns = append(scope.columns, f.Tag.Get("column"))
		} else {
			scope.columns = append(
				scope.columns, 
				fmt.Sprintf("%s.%s",
					scope.conn.SQLTable(name),
					scope.conn.SQLColumn(name, f.Name),
				),
			)
		}
		p.scanners = append(p.scanners, &reflectScanner{index: i, parent: rfltr, column: f})
	}

	ss, sv := scope.query()
	rows, err := scope.conn.Query(ss, sv...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(p.Items()...)
		if err != nil {
			return err
		}
		p.Finalize(vn.Interface())
		tempSlice = reflect.Append(tempSlice, vn.Elem())
		rfltr.item = reflect.New(elem)
	}
	destSlice.Set(tempSlice)

	return nil
}

type drStringArray []string

func (sa drStringArray) Includes(s string) bool{
	for _, si := range sa {
		if si == s {
			return true
		}
	}
	return false
}

type condition struct {
	column string
	cond   string
	vals   []interface{}
}

func (c condition) ToSQL() string {
	if c.column == "" {
		return c.cond
	}
	return c.column + " " + c.cond
}

func questions(n int) string {
	chars := make([]byte, n*2-1)
	for i, _ := range chars {
		if i % 2 == 0 {
			chars[i] = '?'
		} else {
			chars[i] = ','
	 	}
	}
	return string(chars)
}

type planner struct {
	scanners []*reflectScanner
}

func (p *planner) Items() []interface{} {
	output := make([]interface{}, len(p.scanners))
	for i, _ := range output {
		output[i] = p.scanners[i].iface()
	}

	return output
}

type mixed interface {
	SetNull(string)
}

func (p *planner) Finalize(val interface{}) {
	for _, s := range p.scanners {
		if s.column.Type.Kind() == reflect.Ptr {
			s.finalize()
		}
	}
}

type reflectScanner struct {
	parent reflector
	column reflect.StructField
	index int
	b      sql.NullBool
	f      sql.NullFloat64
	i      sql.NullInt64
	s      sql.NullString
	isnull bool
}

type reflector struct {
	item reflect.Value
}

func (rf *reflectScanner) iface() interface{} {
	if rf.column.Type.Kind() == reflect.Ptr {
		switch rf.column.Type.Elem().Kind() {
		case reflect.String:
			return &rf.s
		case reflect.Bool:
			return &rf.b
		case reflect.Float32, reflect.Float64:
			return &rf.f
		default:
			return &rf.i
		}
	} else {
		return rf.parent.item.Elem().Field(rf.index).Addr().Interface()
	}
}

func (rf *reflectScanner) finalize() bool {
	switch rf.column.Type.Kind() {
	case reflect.String:
		if rf.s.Valid {
			rf.parent.item.Elem().Field(rf.index).Set(reflect.ValueOf(&rf.s.String))
		} else {
			return true
		}
	case reflect.Bool:
		if rf.b.Valid {
			rf.parent.item.Elem().Field(rf.index).Set(reflect.ValueOf(&rf.b.Bool))
		} else {
			return true
		}
	case reflect.Float64:
		if rf.f.Valid {
			rf.parent.item.Elem().Field(rf.index).Set(reflect.ValueOf(&rf.f.Float64))
		} else {
			return true
		}
	case reflect.Float32:
		if rf.f.Valid {
			f := float32(rf.f.Float64)
			rf.parent.item.Elem().Field(rf.index).Set(reflect.ValueOf(f))
		} else {
			return true
		}
	case reflect.Int64:
		if rf.i.Valid {
			rf.parent.item.Elem().Field(rf.index).Set(reflect.ValueOf(&rf.i.Int64))
		} else {
			return true
		}
	case reflect.Int:
		if rf.i.Valid {
			i := int(rf.i.Int64)
			rf.parent.item.Elem().Field(rf.index).Set(reflect.ValueOf(&i))
		} else {
			return true
		}
	default:
		if rf.i.Valid {
			rf.parent.item.Elem().Field(rf.index).SetInt(rf.i.Int64)
		} else {
			return true
		}
	}
	return false
}

func DefaultInt(col string) schema.Column {
	return schema.Column{Name: col, Type: "integer", Length: 10}
}

func DefaultString(col string) schema.Column {
	return schema.Column{Name: col, Type: "varchar", Length: 255}
}

func DefaultBool(col string) schema.Column {
	return schema.Column{Name: col, Type: "bool"}
}

func DefaultTime(col string) schema.Column {
	return schema.Column{Name: col, Type: "timestamp"}
}

func createRecord(c *Conn, cols []string, vals []interface{}, name, pkname string) (int, error) {
	sql := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		c.SQLTable(name),
		strings.Join(cols, ", "),
		questions(len(cols)),
	)
	if c.returning {
		sql += " RETURNING " + c.SQLColumn(name, pkname)
		var pk int
		row := c.QueryRow(sql, vals...)
		err := row.Scan(&pk)
		return pk, err
	} else {
		result, err := c.Exec(sql, vals...)
		if err != nil {
			return 0, err
		}
		
		id, err := result.LastInsertId()
		return int(id), err
	}
}

func updateRecord(c *Conn, cols []string, vals []interface{}, name, pkname string) error {
	sql := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s=?",
		c.SQLTable(name),
		strings.Join(cols, " = ?, ") + " = ?",
		c.SQLColumn(name, pkname),
	)
	_, err := c.Exec(sql, vals...)
	return err
}

func deleteRecord(c *Conn, val interface{}, name, pkname string) error {
	sql := fmt.Sprintf(
		"DELETE FROM %s WHERE %s = ?",
		c.SQLTable(name),
		c.SQLColumn(name, pkname),
	)
	_, err := c.Exec(sql, val)
	return err

}

{{ define "config" }}
package {{ .Name }}

import (
		"strings"

		"github.com/acsellers/inflections"
)



type SQLConfig interface {
	SQLTable(string) string
	SQLColumn(string, string) string
}

type NameMap map[string]string

type AppConfig struct {
	SpecialTables NameMap
	SpecialColumns map[string]NameMap

	Normal SQLConfig	
}

func NewAppConfig(driverName string) AppConfig {
	return AppConfig{
		SpecialTables: NameMap{},
		SpecialColumns: map[string]NameMap{},
		Normal: nil,
	}
}

func (c AppConfig) SQLTable(table string) string {
	if name, ok := c.SpecialTables[table]; ok {
		return name
	}
	if c.Normal == nil {
		return table
	}
	return c.Normal.SQLTable(table)
}

func (c AppConfig) SQLColumn(table, column string) string {
	if cols, ok := c.SpecialColumns[table]; ok {
		if name, ok := cols[column]; ok {
			return name
		}
	}
	if c.Normal == nil {
		return column
	}
	return c.Normal.SQLColumn(table, column)
}

type LowerConfig struct{}

func (LowerConfig) SQLTable(table string) string {
	return strings.ToLower(table)
}

func (LowerConfig) SQLColumn(table, column string) string {
	return strings.ToLower(column)
}

type PrefixConfig struct {
	TablePrefix string
	ColumnPrefix string
}

func (pc PrefixConfig) SQLTable(table string) string {
	return pc.TablePrefix + table
}

func (pc PrefixConfig) SQLColumn(table, column string) string {
	return pc.ColumnPrefix + column
}

type RailsConfig struct {}

func (RailsConfig) SQLTable(table string) string {
	return strings.ToLower(inflections.Pluralize(inflections.Underscore(table)))
}

func (RailsConfig) SQLColumn(table, column string) string {
	return strings.ToLower(inflections.Underscore(column))
}
{{ end }}

{{ define "starter_file" }}
package {{ .Name }}

// Add your tables in here, or another'
// file with the .gp extension.
//
// Example Tables:
// type User table {

//         ID int
// 				 Name string
//         Email string
//         BCryptPassword
// }
//
// type BCryptPassword mixin {
//         CryptPassword string
// }
// bcrypt password functions

{{ end }}
{{ define "int_mapper" }}
	if v == nil {
		// do nothing, use zero value
	} else if s, ok := v.(int64); ok {
		{{ if .MustNull }}
			var temp int
			temp = int(s)
			(*m.Mapper.Current).{{ .Name }} = &temp
		{{ else }}
			(*m.Mapper.Current).{{ .Name }} = int(s)
		{{ end }}
	} else if b, ok := v.([]byte); ok {
		i, err := strconv.Atoi(string(b))
		{{ if .MustNull }}
			(*m.Mapper.Current).{{ .Name }} = &i
		{{ else }}
			(*m.Mapper.Current).{{ .Name }} = i
		{{ end }}
		return err
	}
	return nil
{{ end }}
{{ define "string_mapper" }}
	if v == nil {
		// do nothing, use zero value
	} else if s, ok := v.(string); ok {
		{{ if .MustNull }}
			(*m.Mapper.Current).{{ .Name }} = &s
		{{ else }}
			(*m.Mapper.Current).{{ .Name }} = s
		{{ end }}
	} else if s, ok := v.([]byte); ok {
		{{ if .MustNull }}
			(*m.Mapper.Current).{{ .Name }} = &string(s)
		{{ else }}
			(*m.Mapper.Current).{{ .Name }} = string(s)
		{{ end }}
	}

	return nil
{{ end }}
{{ define "time_mapper" }}
	if v == nil {
		// do nothing, use zero value
	} else if s, ok := v.(time.Time); ok {
		{{ if .MustNull }}
			(*m.Mapper.Current).{{ .Name }} = &s
		{{ else }}
			(*m.Mapper.Current).{{ .Name }} = s
		{{ end }}
	}

	return nil
{{ end }}
{{ define "bool_mapper" }}
	if v == nil {
		// it is false or null, the zero values
	} else if b, ok := v.(bool); ok {
		{{ if .MustNull }}
			(*m.Mapper.Current).{{ .Name }} = &b
		{{ else }}
			(*m.Mapper.Current).{{ .Name }} = b
		{{ end }}
	} else if i, ok := v.(int64); ok {
		{{ if .MustNull }}
			var tb bool
			if i == 0 {
				(*m.Mapper.Current).{{ .Name }} = &tb
			} else {
				tb = true
				(*m.Mapper.Current).{{ .Name }} = &tb
			}
		{{ else }}
			if i != 0 {
				(*m.Mapper.Current).{{ .Name }} = true
			}
		{{ end }}
	}

	return nil
{{ end }}
{{ define "float64_mapper" }}
	if v == nil {
		// it is false or null, the zero values
	} else if f, ok := v.(float64); ok {
		{{ if .MustNull }}
			(*m.Mapper.Current).{{ .Name }} = &f
		{{ else }}
			(*m.Mapper.Current).{{ .Name }} = f
		{{ end }}
	} else if i, ok := v.(int64); ok {
		tf := float64(i)
		{{ if .MustNull }}
			(*m.Mapper.Current).{{ .Name }} = &tf
		{{ else }}
			(*m.Mapper.Current).{{ .Name }} = tf
		{{ end }}
	} else if i, ok := v.([]byte); ok {
		f, err := strconv.ParseFloat(string(i), 64)
		if err != nil {
			return err
		}
		{{ if .MustNull }}
			(*m.Mapper.Current).{{ .Name }} = &f
		{{ else }}
			(*m.Mapper.Current).{{ .Name }} = f
		{{ end }}
	}	else {
		return fmt.Errorf("Value not recognized as float64, received %v", v)
	}

	return nil
{{ end }}

{{ define "float32_mapper" }}
	if v == nil {
		// it is false or null, the zero values
	} else if f, ok := v.(float64); ok {
		mf := float32(f)
		{{ if .MustNull }}
			(*m.Mapper.Current).{{ .Name }} = &mf
		{{ else }}
			(*m.Mapper.Current).{{ .Name }} = mf
		{{ end }}
	} else if i, ok := v.(int64); ok {
		tf := float32(i)
		{{ if .MustNull }}
			(*m.Mapper.Current).{{ .Name }} = &tf
		{{ else }}
			(*m.Mapper.Current).{{ .Name }} = tf
		{{ end }}
	} else if i, ok := v.([]byte); ok {
		f, err := strconv.ParseFloat(string(i), 32)
		if err != nil {
			return err
		}
		sf := float32(f)
		{{ if .MustNull }}
			(*m.Mapper.Current).{{ .Name }} = &sf
		{{ else }}
			(*m.Mapper.Current).{{ .Name }} = sf
		{{ end }}
	}	else {
		return fmt.Errorf("Value not recognized as float32, received %v", v)
	}

	return nil
{{ end }}

{{ define "byte_mapper" }}
	if v == nil {
		// do nothing, use zero value
	} else if s, ok := v.([]byte); ok {
		{{ if .MustNull }}
			(*m.Mapper.Current).{{ .Name }} = &s
		{{ else }}
			(*m.Mapper.Current).{{ .Name }} = s
		{{ end }}
	}

	return nil
{{ end }}
`
