package gb

/**
// Build returns a Target representing the result of compiling the Package pkg
// and its dependencies. If pkg is a command, then the results of build include
// linking the final binary into pkg.Context.Bindir().
func Build(ctx *Context, pkg *Package) Target {
        if pkg.Name() == "main" {
                return buildCommand(ctx, pkg)
        }
        return buildPackage(ctx, pkg)
}

// buildPackage returns a Target repesenting the results of compiling
// pkg and its dependencies.
func buildPackage(ctx *Context, pkg *Package) Target {
        var deps []Future
        for _, dep := range pkg.Imports {
                // TODO(dfc) use project.Spec
                pkg, err := ctx.ResolvePackage(runtime.GOOS, runtime.GOARCH, dep).Result()
                if err != nil {
                        return &errFuture{err}
                }
                deps = append(deps, buildPackage(ctx, pkg))
        }
        return ctx.addTargetIfMissing(pkg, func() Future { return Compile(ctx, pkg, deps) })
}

// buildCommand returns a Target repesenting the results of compiling
// pkg as a command and linking the result into pkg.Context.Bindir().
func buildCommand(ctx *Context, pkg *Package) Target {
        var deps []Future
        for _, dep := range pkg.Imports {
                // TODO(dfc) use project.Spec
                pkg, err := ctx.ResolvePackage(runtime.GOOS, runtime.GOARCH, dep).Result()
                if err != nil {
                        return errFuture{err}
                }
                deps = append(deps, buildPackage(ctx, pkg))
        }
        compile := Compile(ctx, pkg, deps)
        ld := Ld(ctx, pkg, compile)
        return ld
}
**/
