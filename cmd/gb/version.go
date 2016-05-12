package main

import (
	"fmt"
	"os"
	"os/exec"
)

func version() {
	gopath := os.Getenv("GOPATH")
	git, err := exec.LookPath("git")
	if err != nil {
		fmt.Fprintf(os.Stderr, "you need to install git\n\n")
		exit(2)
	}

	cmd := exec.Command(git, "--git-dir="+gopath+"/src/github.com/constabulary/gb/.git", "describe", "--abbrev=0", "--tags")
	out, err := cmd.Output()

	if err != nil {
		fmt.Fprintf(os.Stderr, "gb repository can not be found\n\n")
		exit(2)
	}
	fmt.Print("gb version ", string(out))
}
