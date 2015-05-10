# gb

[![wercker status](https://app.wercker.com/status/494a8ac6b836f39cc7e67036d957a43e/m "wercker status")](https://app.wercker.com/project/bykey/494a8ac6b836f39cc7e67036d957a43e)

`gb` is a proof of concept replacement build tool for the [Go programming language](https://golang.org).

I gave a talk about `gb` and the rational for its creation at GDG Berlin in April 2015, [video](https://www.youtube.com/watch?v=c3dW80eO88I) and [slides](http://go-talks.appspot.com/github.com/davecheney/presentations/reproducible-builds.slide#1).

## Project based

`gb` operates on the concept of a project. A gb project is a workspace for all the Go code that is required to build your project.

A gb project is a folder on disk that contains a subdirectory named <code>src/</code>. That's it, no environment variables to set. For the rest of this document we'll refer to your <code>gb</code> project as <code>$PROJECT</code>.

You can create as many projects as you like and move between them simply by changing directories.

## Installation

    go get github.com/constabulary/gb/...

## Getting started

See the [getting started](getting-started.md) document.

## Commands

`gb` is the main command. It supports subcommands, of which there are currently two:

- `build` - which takes one or more import paths, ie `gb build github.com/constabulary/gb/cmd/gb`, if executed inside `$PROJECT/src/some/path/`, `gb build` will build that path.
- `test` - behaves identically to `gb build`, but runs tests.

## Project root auto detection

A `gb` project is defined as any directory that contains a `src/` subdirectory. `gb` automatically detects the root of the project by looking at the current working directory and walking backwards until it finds a directory called `src/`.

## Arguments

Arguments to `gb` subcommands are package import paths or globs relative to the project `src/` directory

- `gb build github.com/a/b` - builds `github.com/a/b`
- `gb build github.com/a/b/...` - builds `github.com/a/b` and all packages below it
- `gb build .../cmd/...` - builds anything that matches `.*/cmd/.*`
- `gb build` - shorthand for `go build ...`, depending on the current working directory this will be the entire project, or a subtree.

Other subcommands, like `test`, `vendor`, etc follow the same rule.

*note*: only import paths within the `src/` directory will match, it is not possible to build source from the `vendor/src/` directory; it will be built if needed by virtue of being imported by a package in the `src/` directory.

## Incremental compilation

By default `gb` always performs incremental compilation and caches the results in `$PROJECT/pkg/`. See the Flags section for options to alter this behaviour.

## Flags

The following flags are supported by `gb`. Note that these are flags to subcommands, so must come *after* the subcommand.
- `-R` - sets the base of the project root search path from the current working directory to the value supplied. Effectively `gb` changes working directory to this path before searching for the project root.
- `-v` - increases verbosity, effectively lowering the output level from INFO to DEBUG.
- `-q` - decreases verbosity, effectively raising the output level to ERROR. In a successful build, no output will be displayed.
- `-goroot` - alters the path to the go toolchain in use, eg `gb build -goroot=$HOME/go1.4`.
- `-goos`, `-goarch` - analogous to `env GOOS=... GOARCH=... gb`.
- `-f` - ignore cached packages if present, new packages built will overwrite any cached packages. This effectively disables incremental compilation.
- `-F` - do not cache packages, cached packages will still be used for incremental compilation, `-f -F` is advised to disable the package caching system.

## Plugins

`gb` supports git style plugins, anything in the path that starts with `gb-` is considered a plugin. Plugins are executed from the main `gb` tool. At the moment there are two plugins shipped with `gb`.

- `env` - analogous to `go env`, useful for debugging the environment passed to `gb` plugins, tranditionally all environment variables in this set begin with `GB_`.
- `vendor` - is a simple wrapper around `go get` to allow easy bootstrapping of a project by fetching dependencies in to the `vendor/src/` directory.
