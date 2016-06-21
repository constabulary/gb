package main_test

import "testing"

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
