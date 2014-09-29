package parse

import (
	"bytes"
	"fmt"
	"io"
	"text/template"

	"code.google.com/p/go.tools/imports"
)

var schemaTemplate = `/*
  This code was generated by the Doctor ORM Generator and isn't meant to be edited.
	If at all possible, please regenerate this file from your gp files instead of
	attempting to edit it to add changes.
*/

package {{ .Name }}

import "github.com/acsellers/dr/schema"

var Schema = schema.Schema{
	Tables: map[string]*schema.Table{
		{{ range $table := .Tables }}
			"{{ .Name }}": &schema.Table{
				Name: "{{ .Name }}",
				Columns: []schema.Column{
					{{ range $column := .Columns }}
						{{ if $column.SimpleType }}
							schema.Column{
								Name: "{{ $column.Name }}",
								Type: "{{ $column.Type }}",
								Length: {{ $column.Length }},
							},
						{{ end }}
						{{ if $column.Subrecord }}
							{{ range $subcolumn := $column.Subcolumns }}
								schema.Column{
									Name: "{{ $subcolumn.Name }}",
									Type: "{{ $subcolumn.Type }}",
									Length: {{ $subcolumn.Length }},
									IncludeName: "{{ $subcolumn.IncludeName }}",
								},
							{{ end }}
						{{ end }}
					{{ end }}
				},
			},
		{{ end }}
	},
}

func init() {
	{{ range $table := .Tables }}
		Schema.Tables["{{ .Name }}"].HasMany = []schema.ManyRelationship{
			{{ range $column := $table.Columns }}
				{{ if $column.IsHasMany }}
					schema.ManyRelationship{
						Schema.Tables["{{ $table.Name }}"],
						Schema.Tables["{{ $column.GoType }}"],
						Schema.Tables["{{ $column.GoType }}"].FindColumn("{{ $column.ChildColumn }}"),
					},
				{{ end }}
			{{ end }}
		}
	{{ end }}

	{{ range $table := .Tables }}
		Schema.Tables["{{ .Name }}"].ChildOf = []schema.ManyRelationship{
			{{ range $column := $table.Columns }}
				{{ if $column.IsChildHasMany }}
					schema.ManyRelationship{
						Schema.Tables["{{ $column.ParentCol.Tbl.Name }}"],
						Schema.Tables["{{ $table.Name }}"],
						Schema.Tables["{{ $table.Name }}"].FindColumn("{{ $column.Name }}"),
					},
				{{ end }}
			{{ end }}
		}
	{{ end }}

	{{ range $table := .Tables }}
		Schema.Tables["{{ .Name }}"].HasOne = []schema.OneRelationship{
			{{ range $column := $table.Columns }}
				{{ if $column.IsHasOne }}
					schema.OneRelationship{
						Schema.Tables["{{ $table.Name }}"],
						Schema.Tables["{{ $column.GoType }}"],
						Schema.Tables["{{ $column.GoType }}"].FindColumn("{{ $column.ChildColumn }}"),
					},
				{{ end }}
			{{ end }}
		}
	{{ end }}

	{{ range $table := .Tables }}
		Schema.Tables["{{ .Name }}"].BelongsTo = []schema.OneRelationship{
			{{ range $column := $table.Columns }}
				{{ if $column.IsBelongsTo }}
					schema.OneRelationship{
						Schema.Tables["{{ $column.ParentCol.Tbl.Name }}"],
						Schema.Tables["{{ $column.ParentCol.GoType }}"],
						Schema.Tables["{{ $column.ParentCol.GoType }}"].FindColumn("{{ $column.Name }}"),
					},
				{{ end }}
			{{ end }}
		}
	{{ end }}
}

{{ range $table := .Tables }}
func (t *{{ $table.Name }}) Save(c *Conn) error {

	// check the primary key vs the zero value, if they match then
	// we will assume we have a new record
	var pkz {{ .PrimaryKeyColumn.GoType }}
	if t.{{ .PrimaryKeyColumn.Name }} == pkz {
		return t.create(c)
	} else {
		return t.update(c)
	}
}

func (t *{{ $table.Name }}) create(c *Conn) error {
	vals := []interface{}{}
	cols := []string{}
	{{ range $column := $table.Columns }}
		{{ if and (ne $column.Name $table.PrimaryKeyColumn.Name) $column.SimpleType }}
			vals = append(vals, t.{{ $column.Name }})
			cols = append(cols, c.SQLColumn("{{ $table.Name }}", "{{ $column.Name }}"))
		{{ end }}
		{{ if $column.Subrecord }}
			{{ range $subcolumn := $column.Subcolumns }}
				{{ if $subcolumn.SimpleType }}
					if t.{{ $column.Subrecord.Name }}.{{ $subcolumn.Name }}{{ $subcolumn.NonZeroCheck }} {
						vals = append(vals, t.{{ $column.Subrecord.Name }}.{{ $subcolumn.Name }})
						cols = append(cols, c.SQLColumn("{{ $table.Name }}", "{{ $subcolumn.Name }}"))
					}
				{{ end }}
			{{ end }}
		{{ end }}
	{{ end }}

	sql := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		c.SQLTable("{{ $table.Name }}"),
		strings.Join(cols, ", "),
		questions(len(cols)),
	)
	result, err := c.Exec(sql, vals...)
	if err != nil {
		return err
	}
	
	id, err := result.LastInsertId()
	if err == nil {
		t.{{ $table.PrimaryKeyColumn.Name }} = {{ $table.PrimaryKeyColumn.GoType }}(id)
	}

	return nil
}

func (t *{{ $table.Name }}) update(c *Conn) error {
	vals := []interface{}{}
	cols := []string{}
	{{ range $column := $table.Columns }}
		{{ if and (ne $column.Name $table.PrimaryKeyColumn.Name) $column.SimpleType }}
			vals = append(vals, t.{{ $column.Name }})
			cols = append(cols, c.SQLColumn("{{ $table.Name }}", "{{ $column.Name }}") + "= ?")
		{{ end }}
	{{ end }}

	sql := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s=?",
		c.SQLTable("{{ $table.Name }}"),
		strings.Join(cols, ", "),
		c.SQLColumn("{{ $table.Name }}", "{{ $table.PrimaryKeyColumn.Name }}"),
	)
	_, err := c.Exec(sql, append(vals, t.{{ $table.PrimaryKeyColumn.Name }})...)
	return err
}

func (t {{ $table.Name }}) Delete(c *Conn) error {
	sql := fmt.Sprintf(
		"DELETE FROM %s WHERE %s = ?",
		c.SQLTable("{{ $table.Name }}"),
		c.SQLColumn("{{ $table.Name }}", "{{ $table.PrimaryKeyColumn.Name }}"),
	)
	_, err := c.Exec(sql, t.{{ $table.PrimaryKeyColumn.Name }})
	return err
}
{{ end }}
`

var sTmpl *template.Template

func init() {
	var err error
	sTmpl, err = template.New("schema").Parse(schemaTemplate)
	if err != nil {
		panic(err)
	}
}

func (pkg *Package) WriteSchemaFile(w io.Writer) {
	b := &bytes.Buffer{}
	err := sTmpl.Execute(b, pkg)
	if err != nil {
		panic(err)
	}
	ib, err := imports.Process(pkg.ActiveFiles[0].AST.Name.Name+"_schema.go", b.Bytes(), nil)
	if err != nil {
		fmt.Println("Error in Gen File:", err)
		w.Write(b.Bytes())
		return
	}
	w.Write(ib)
}
