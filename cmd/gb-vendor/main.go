package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/constabulary/gb/cmd"
)

func main() {
	var projectdir string

	flag.StringVar(&projectdir, "p", os.Getenv("GB_PROJECT_DIR"), "project directory")
	flag.Parse()

	vendor := filepath.Join(projectdir, "vendor")
	env := cmd.MergeEnv(os.Environ(), map[string]string{
		"GOPATH": vendor,
	})

	gotool, err := exec.LookPath("go")
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Cmd{
		Path: gotool,
		Args: append([]string{"go", "get", "-d"}, flag.Args()...),
		Env:  env,

		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
