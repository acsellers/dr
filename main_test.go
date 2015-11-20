package drtest

import (
	"testing"

	"github.com/acsellers/dr/gen"
	"github.com/acsellers/dr/gp"
)

func TestInit(t *testing.T) {
	pkg := gp.Package{}
	pkg.SetName("blah")
	gen.WriteLibraryFiles(pkg)
	gen.WriteStarterFile(pkg)
}
