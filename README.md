# gb

### Build status
Unix:
[![travis-cs status](https://travis-ci.org/constabulary/gb.svg "travis-ci status")](https://travis-ci.org/constabulary/gb)

Windows:
[![Build status](https://ci.appveyor.com/api/projects/status/rjttg1agmp2sra3h/branch/master?svg=true)](https://ci.appveyor.com/project/davecheney/gb/branch/master)

`gb` is a proof of concept replacement build tool for the [Go programming language](https://golang.org).

I gave a talk about `gb` and the rational for its creation at GDG Berlin in April 2015, [video](https://www.youtube.com/watch?v=c3dW80eO88I) and [slides](http://go-talks.appspot.com/github.com/davecheney/presentations/reproducible-builds.slide#1).

## Project based

`gb` operates on the concept of a project. A gb project is a workspace for all the Go code that is required to build your project.

A gb project is a folder on disk that contains a subdirectory named <code>src/</code>. That's it, no environment variables to set. For the rest of this document we'll refer to your <code>gb</code> project as <code>$PROJECT</code>.

You can create as many projects as you like and move between them simply by changing directories.

## Installation

    go get github.com/constabulary/gb/...

## Read more

gb has its own site, [getgb.io](http://getgb.io/), head over there for more information.

## Contributing

### Contribution guidelines

We welcome pull requests, bug fixes and issue reports.

Before proposing a large change, please discuss your change by raising an issue.

### Road map

#### Completed

- [Cross Compilation](https://github.com/constabulary/gb/milestones/cross-compilation)
- Tag handling, unify -tags, ENVVARS and GOOS/GOARCH into a single format for binary names and pkg cache
- gb test improvements, test output, test flag handling
- [Race detector support](https://github.com/constabulary/gb/issues/96)

#### Todo

- 0.4 series: gb vendor updates and bug fixes
- 0.5 series: new package resolver (replace go/build)

### Big ticket items 

Big ticket items that are not on the road map yet

- Package BuildID support (make stale detection work like the Go 1.5)
- `gccgo` toolchain support.
