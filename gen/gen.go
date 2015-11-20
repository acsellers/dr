package gen

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"text/template"

	"github.com/acsellers/dr/gp"
	"github.com/acsellers/inflections"

	"golang.org/x/tools/imports"
)

var tmpl *template.Template

func init() {
	rg := regexp.MustCompile(`^[A-Z].*`)
	tmpl = template.New("dr").
		Funcs(template.FuncMap{
		"plural": inflections.Pluralize,
		"public": func(s string) bool {
			return rg.MatchString(s)
		},
	})

	var err error
	names, _ := AssetDir("gen/assets")
	for _, name := range names {
		asset, _ := Asset("gen/assets/"+name)
		tmpl, err = tmpl.New(name).Parse(string(asset))
		if err != nil {
			panic(err)
		}
	}
	fmt.Println(len(tmpl.Templates()))
	for _, tmp := range tmpl.Templates() {
		fmt.Println(tmp.Name())
	}
}

func Output(pkg *gp.Package) {
	OutputTemplates(pkg)
}

func OutputTemplates(pkg *gp.Package) {
	b := &bytes.Buffer{}
	err := tmpl.ExecuteTemplate(b, "gen", pkg)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(pkg.ActiveFiles[0].AST.Name.Name + "_gen.go")
	if err != nil {
		fmt.Println("Could not write schema file")
	}
	defer f.Close()

	ib, err := imports.Process(pkg.ActiveFiles[0].AST.Name.Name+"_gen.go", b.Bytes(), nil)
	if err != nil {
		fmt.Println("Error in Gen File:", err)
		f.Write(b.Bytes())
		return
	}
	f.Write(ib)

	w, err := os.Create(pkg.ActiveFiles[0].AST.Name.Name + "_schema.go")
	if err != nil {
		panic("Couldn't open schema file for writing: " + err.Error())
	}
	b = &bytes.Buffer{}
	err = tmpl.ExecuteTemplate(b, "schema", pkg)
	if err != nil {
		panic(err)
	}
	ib, err = imports.Process(pkg.ActiveFiles[0].AST.Name.Name+"_schema.go", b.Bytes(), nil)
	if err != nil {
		fmt.Println("Error in Gen File:", err)
		w.Write(b.Bytes())
		return
	}
	w.Write(ib)

	WriteLibraryFiles(pkg)
}

func WriteLibraryFiles(pkg *gp.Package) {
	b := &bytes.Buffer{}
	err := tmpl.ExecuteTemplate(b, "lib", pkg)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(pkg.Name() + "_lib.go")
	if err != nil {
		fmt.Println("Could not write schema file:", err)
	}

	ib, err := imports.Process(pkg.Name()+"_lib.go", b.Bytes(), nil)
	if err != nil {
		fmt.Println("Error in Gen File:", err)
		f.Write(b.Bytes())
		f.Close()
		return
	}
	f.Write(ib)
	f.Close()

	filename := "db_config.go"
	if _, err = os.Stat(filename); err == nil {
		// fmt.Println("Library file already written")
		return
	}
	b = &bytes.Buffer{}
	err = tmpl.ExecuteTemplate(b, "config", pkg)
	if err != nil {
		panic(err)
	}

	f, err = os.Create(filename)
	if err != nil {
		fmt.Println("Could not write schema file")
	}

	ib, err = imports.Process(filename, b.Bytes(), nil)
	if err != nil {
		fmt.Println("Error in Gen File:", err)
		f.Write(b.Bytes())
		f.Close()
		return
	}
	f.Write(ib)
	f.Close()

}

func WriteStarterFile(pkg *gp.Package) {
	if _, err := os.Stat(pkg.Name() + ".gp"); err == nil {
		fmt.Println("Starter file already written")
		return
	}

	f, err := os.Create(pkg.Name() + ".gp")
	if err != nil {
		fmt.Println("Could not write schema file")
	}
	defer f.Close()

	err = tmpl.ExecuteTemplate(f, "starter_file", pkg)
	if err != nil {
		panic(err)
	}
}
