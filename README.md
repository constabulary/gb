# gb

[![wercker status](https://app.wercker.com/status/494a8ac6b836f39cc7e67036d957a43e/m "wercker status")](https://app.wercker.com/project/bykey/494a8ac6b836f39cc7e67036d957a43e)

`gb` is a proof of concept replacement build tool for the [Go programming language](https://golang.org).

I gave a talk about `gb` and the rational for its creation at GDG Berlin in April 2015, [video](https://www.youtube.com/watch?v=c3dW80eO88I) and [slides](http://go-talks.appspot.com/github.com/davecheney/presentations/reproducible-builds.slide#1).

## Project based

`gb` operates on the concept of a project. A project has the following properties:

- A project is the consumer of your own source code, and possibly dependencies that your code consumes; nothing consumes the code from a project. Another way of thinking about it is, a project is where package `main` is.
- A project is conceptually a `$GOPATH` workspace dedicated to your project's code.
- A project supports multiple locations for source code, at the moment `src/` for your source code, and `vendor/src/` for third party code that you have copied, cloned, forked, or otherwise included in the project.
- The code that represents an `import` path is controlled by the project, by virtue of being present in one of the source code directories in the project.

## Installation

    go get github.com/constabulary/gb/...

## Getting started

See the [getting started](getting-started.md) document.

## Commands

`gb` is the main command. It supports subcommands, of which there are currently two:

- `build` - which takes one or more import paths, ie `gb build github.com/constabulary/gb/cmd/gb`, if executed inside `$PROJECT/src/some/path/`, `gb build` will build that path.
- `test` - behaves identically to `gb build`, but runs tests.

## Arguments

Arguments to `gb` subcommands are package import paths or globs relative to the project `src/` directory

- `gb build github.com/a/b` - builds `github.com/a/b`
- `gb build github.com/a/b/...` - builds `github.com/a/b` and all packages below it
- `gb build .../cmd/...` - builds anything that matches `.*/cmd/.*`
- `gb build` - builds the package in the current directory.

Other subcommands, like `test`, `vendor`, etc follow the same rule.

*note*: only import paths within the `src/` directory will match, it is not possible to build source from the `vendor/src/` directory; it will be built if needed by virtue of being imported by a package in the `src/` directory.

## Incremental compilation

By default `gb` always performs incremental compilation and caches the results in `$PROJECT/pkg/`. See the Flags section for options to alter this behaviour.

## Flags

The following flags are supported by `gb`. Note that these are flags to subcommands, so must come *after* the subcommand.
- `-v` - increases verbosity, effectively lowering the output level from INFO to DEBUG.
- `-q` - decreases verbosity, effectively raising the output level to ERROR. In a successful build, no output will be displayed.
- `-goroot` - alters the path to the go toolchain in use, eg `gb build -goroot=$HOME/go1.4`.
- `-goos`, `-goarch` - analogous to `env GOOS=... GOARCH=... gb`.
- `-f` - ignore cached packages if present, new packages built will overwrite any cached packages.
- `-F` - do not cache packages, cached packages will still be used for incremental compilation, `-f -F` is advised to disable the package caching system.

## Plugins

`gb` supports git style plugins, anything in the path that starts with `gb-` is considered a plugin. Plugins are executed from the main `gb` tool. At the moment there are two plugins shipped with `gb`.

- `env` - analogous to `go env`, useful for debugging the environment passed to `gb` plugins, tranditionally all environment variables in this set begin with `GB_`.
- `vendor` - is a simple wrapper around `go get` to allow easy bootstrapping of a project by fetching dependencies in to the `vendor/src/` directory.

