package gb

// NullToolchain configures the Context to use the null toolchain.
func NullToolchain(c *Context) error {
	c.tc = new(nulltoolchain)
	return nil
}

type nulltoolchain struct{}

func (nulltoolchain) Gc(importpath []string, srcdir, _, outfile string, files []string, _ bool) error {
	Debugf("null:gc %v %v %v %v", importpath, srcdir, outfile, files)
	return nil
}

func (nulltoolchain) Asm(srcdir, ofile, sfile string) error {
	Debugf("null:asm %v %v %v", srcdir, ofile, sfile)
	return nil
}
func (nulltoolchain) Pack(afiles ...string) error {
	Debugf("null:pack %v %v", afiles)
	return nil
}
func (nulltoolchain) Ld(_, _ []string, aout string, afile string) error {
	Debugf("null:ld %v %v", aout, afile)
	return nil
}
func (nulltoolchain) Cc(srcdir, objdir, ofile, cfile string) error {
	Debugf("null:cc %v %v %v %v", srcdir, objdir, ofile, cfile)
	return nil
}
