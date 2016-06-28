package gb

import (
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
