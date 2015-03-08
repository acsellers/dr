package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/acsellers/dr/parse"
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
				pkg := parse.Package{}
				pkg.SetName(c.Args().First())
				pkg.WriteLibraryFiles()
				pkg.WriteStarterFile()
			},
		},
		{
			Name:      "build",
			ShortName: "b",
			Usage:     "Create the access library",
			Action: func(c *cli.Context) {
				pkg := parse.Package{Funcs: make(map[string][]parse.Func)}
				pkg.WriteLibraryFiles()
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
				err = pkg.ParseSrc(files...)
				if err != nil {
					log.Fatal("Couldn't parse files:", name, "got error:", err)
				}
			},
		},
	}
	app.Run(os.Args)
}
