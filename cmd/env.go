package cmd

import (
	"fmt"
	"strings"
)

// MergeEnv merges args into env, overwriting entries.
func MergeEnv(env []string, args map[string]string) []string {
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
