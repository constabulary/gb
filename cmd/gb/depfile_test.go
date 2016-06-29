package main_test

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestMissingDepfile(t *testing.T) {
	gb := T{T: t}
	defer gb.cleanup()

	gb.tempDir("src/github.com/user/proj")
	gb.tempFile("src/github.com/user/proj/main.go", `package main

import "fmt"
import "github.com/a/b" // would be in depfile

func main() {
	fmt.Println(b.B)
}
`)

	gb.cd(gb.tempdir)
	gb.runFail("build")
	gb.grepStderr(`FATAL: command "build" failed:.+import "github.com/a/b": not found`, `import "github.com/a/b": not found`)
}

func TestDepfileFetchMissingByVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test during -short")
	}

	gb := T{T: t}
	defer gb.cleanup()

	gb.tempDir("src/github.com/user/proj/a/")
	gb.tempFile("src/github.com/user/proj/a/main.go", `package main

import "github.com/pkg/profile"

func main() {
	defer profile.Start().Stop()
}
`)
	gb.tempFile("depfile", `
github.com/pkg/profile	version=1.1.0
`)

	gb.cd(gb.tempdir)
	gbhome := gb.tempDir(".gb")
	gb.setenv("GB_HOME", gbhome)
	gb.run("build")
	name := "a"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	gb.wantExecutable(gb.path("bin", name), "expected $PROJECT/bin/"+name)
	gb.grepStdout("^fetching github.com/pkg/profile", "fetching pkg/profile not found")
	gb.mustExist(filepath.Join(gbhome, "cache", "8fd41ea4fa48cd8435005bad56faeefdc57a25d6", "src", "github.com", "pkg", "profile", "profile.go"))
}

func TestDepfileFetchMissingByTag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test during -short")
	}

	gb := T{T: t}
	defer gb.cleanup()

	gb.tempDir("src/github.com/user/proj/a/")
	gb.tempFile("src/github.com/user/proj/a/main.go", `package main

import "github.com/pkg/profile"

func main() {
	defer profile.Start().Stop()
}
`)
	gb.tempFile("depfile", `
github.com/pkg/profile	tag=v1.1.0
`)

	gb.cd(gb.tempdir)
	gbhome := gb.tempDir(".gb")
	gb.setenv("GB_HOME", gbhome)
	gb.setenv("DEBUG", ".")
	gb.run("build")
	name := "a"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	gb.wantExecutable(gb.path("bin", name), "expected $PROJECT/bin/"+name)
	gb.grepStdout("^fetching github.com/pkg/profile", "fetching pkg/profile not found")
	gb.mustExist(filepath.Join(gbhome, "cache", "e693c641ace92b5910c4a64d3241128094f74f19", "src", "github.com", "pkg", "profile", "profile.go"))
}
