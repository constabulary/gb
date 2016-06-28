package gb

import "github.com/constabulary/gb/internal/importer"

type vendorImporter struct {
	importer.Importer
}

func (i vendorImporter) Import(path string) (*importer.Package, error) {
	return i.Importer.Import(path)
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

type gorootImporter struct {
	Importer
	im importer.Importer
}

func (i *gorootImporter) Import(path string) (*importer.Package, error) {
	pkg, err := i.im.Import(path)
	if err != nil {
		return i.Importer.Import(path)
	}
	return pkg, nil
}
