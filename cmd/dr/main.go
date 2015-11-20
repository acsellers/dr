package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/acsellers/dr/gen"
	"github.com/acsellers/dr/gp"
	"github.com/acsellers/dr/old/parse"
	"github.com/codegangsta/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "dr"
	app.Usage = "build a RDBMS access library"
	app.Commands = []cli.Command{
		{
			Name:      "init",
			ShortName: "i",
			Usage:     "Create base code and go generate task",
			Action: func(c *cli.Context) {
				pkg := gp.Package{}
				pkg.SetName(c.Args().First())
				gen.WriteLibraryFiles(pkg)
				gen.WriteStarterFile(pkg)
			},
		},
		{
			Name:      "build",
			ShortName: "b",
			Usage:     "Create the access library",
			Action: func(c *cli.Context) {
				pkg := parse.Package{Funcs: make(map[string][]parse.Func)}
				names, _ := filepath.Glob("*.gp")
				files := make([]*os.File, 0, len(names))
				for _, name := range names {
					f, err := os.Open(name)
					if err != nil {
						log.Fatal("Couldn't open file:", name, "got error:", err)
					}
					files = append(files, f)
					defer f.Close()
				}
				err := pkg.ParseSrc(files...)
				if err != nil {
					log.Fatal("Couldn't parse files got error:", err)
				}
				gen.WriteLibraryFiles(pkg)
			},
		},
	}
	app.Run(os.Args)
}
