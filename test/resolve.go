package test

import (
	"github.com/constabulary/gb"
)

type Resolver interface {
	Projectdir() string
	ResolvePackage(string) (*gb.Package, error)
}

// TestResolver returns a Resolver that resolves packages, their
// dependencies including any internal or external test dependencies.
func TestResolver(r Resolver) Resolver {
	return &testResolver{r}
}

type testResolver struct {
	Resolver
}

func (r *testResolver) ResolvePackage(path string) (*gb.Package, error) {
	p, err := r.Resolver.ResolvePackage(path)
	if err != nil {
		return nil, err
	}
	var imports []string
	imports = append(imports, p.Package.TestImports...)
	imports = append(imports, p.Package.XTestImports...)
	for _, i := range imports {
		_, err := r.Resolver.ResolvePackage(i)
		if err != nil {
			return nil, err
		}
	}
	return p, nil
}
