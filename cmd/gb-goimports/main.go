package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	c := exec.Command("goimports", os.Args[1:]...)
	c.Env = env()
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func env() []string {
	env := os.Environ()
	for i, v := range env {
		if strings.HasPrefix(v, "GOPATH=") {
			env[i] = "GOPATH=" + gopath()
			return env
		}
	}
	return append(env, "GOPATH="+gopath())
}

var projexp = regexp.MustCompile(`GB_PROJECT_DIR=([^\n]*)`)

func gopath() string {
	// When run via `gb goimports`, the project dir is available via
	// GB_PROJECT_DIR.
	proj := os.Getenv("GB_PROJECT_DIR")
	if proj == "" {
		// When run via gb-goimports, we won't have the env. Get it via
		// `gb env`.
		b, err := exec.Command("gb", "env").Output()
		if err != nil {
			fmt.Println(string(b))
			fmt.Println(err)
			os.Exit(1)
		}
		matches := projexp.FindStringSubmatch(string(b))
		if len(matches) < 2 {
			fmt.Println("unable to find project directory")
			os.Exit(1)
		}
		proj = matches[1]
	}
	return proj + ":" + filepath.Join(proj, "vendor")
}
