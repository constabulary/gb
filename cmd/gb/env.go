package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("unable to determine current working directory: %v", err)
	}
	return wd
}

// mergeEnv merges args into env, overwriting entries.
func mergeEnv(env []string, args map[string]string) []string {
	m := make(map[string]string)
	for _, e := range env {
		v := strings.SplitN(e, "=", 2)
		m[v[0]] = v[1]
	}
	for k, v := range args {
		m[k] = v
	}
	env = make([]string, 0, len(m))
	for k, v := range m {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
}
