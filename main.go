package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/acsellers/dr/parse"
)

func main() {
	pkg := parse.Package{Funcs: make(map[string][]parse.Func)}
	names, _ := filepath.Glob("*.gp")

	for _, name := range names {
		f, err := os.Open(name)
		if err != nil {
			log.Fatal("Couldn't open file:", name, "got error:", err)
		}
		err = pkg.ParseSrc(f)
		f.Close()
		if err != nil {
			log.Fatal("Couldn't parse file:", name, "got error:", err)
		}
	}

	pkg.OutputTemplates()
}
