package gb

import (
	"os"

	"github.com/constabulary/gb/internal/importer"
	"github.com/pkg/errors"
)

type nullImporter struct{}

func (i *nullImporter) Import(path string) (*importer.Package, error) {
	return nil, errors.Errorf("import %q not found")
}

type srcImporter struct {
	Importer
	im importer.Importer
}

func (i *srcImporter) Import(path string) (*importer.Package, error) {
	pkg, err := i.im.Import(path)
	if err == nil {
		return pkg, nil
	}

	// gb expects, when there is a failure to resolve packages that
	// live in $PROJECT/src that the importer for that directory
	// will report them.

	pkg, err2 := i.Importer.Import(path)
	if err2 == nil {
		return pkg, nil
	}
	return nil, err
}

type _importer struct {
	Importer
	im importer.Importer
}

func (i *_importer) Import(path string) (*importer.Package, error) {
	pkg, err := i.im.Import(path)
	if err != nil {
		return i.Importer.Import(path)
	}
	return pkg, nil
}

type fixupImporter struct {
	Importer
}

func (i *fixupImporter) Import(path string) (*importer.Package, error) {
	pkg, err := i.Importer.Import(path)
	switch err.(type) {
	case *os.PathError:
		return nil, errors.Wrapf(err, "import %q: not found", path)
	default:
		return pkg, err
	}
}
