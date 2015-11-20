package parse

import (
	"os"
	"testing"
)

func TestTableParse(t *testing.T) {
	pkg := Package{Funcs: make(map[string][]Func)}
	os.Chdir("example")

	file, err := os.Open("test_file.gp")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	err = pkg.ParseSrc(file)
	if err != nil {
		t.Fatal(err)
	}

	pkg.OutputTemplates()
}
