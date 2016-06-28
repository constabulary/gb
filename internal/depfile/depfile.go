// depfile loads a file of tagged key value pairs.
package depfile

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
)

// ParseFile parses path into a tagged key value map.
// See Parse for the syntax of the file.
func ParseFile(path string) (map[string]map[string]string, error) {
	r, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "ParseFile")
	}
	defer r.Close()
	return Parse(r)
}

// Parse parses the contents of r into a tagged key value map.
// If successful Parse returns a map[string]map[string]string.
// The format of the line is
//
//     name key=value [key=value]...
//
// Elements can be seperated by whitespace (space and tab).
// Lines that do not begin with a letter or number are ignored. This
// provides a simple mechanism for commentary
//
//     # some comment
//     github.com/pkg/profile version=0.1.0
//
//     ; some other comment
//     // third kind of comment
//       lines starting with blank lines are also ignored
//     github.com/pkg/sftp version=0.2.1
func Parse(r io.Reader) (map[string]map[string]string, error) {
	sc := bufio.NewScanner(r)
	m := make(map[string]map[string]string)
	var lineno int
	for sc.Scan() {
		line := sc.Text()
		lineno++

		// skip blank line
		if line == "" {
			continue
		}

		// valid lines start with a letter or number everything else is ignored.
		// we don't need to worry about unicode because import paths are restricted
		// to the DNS character set, which is a subset of ASCII.
		if !isLetterOrNumber(line[0]) {
			continue
		}

		name, kv, err := parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("%d: %v", lineno, err)
		}
		m[name] = kv
	}
	return m, sc.Err()
}

func parseLine(line string) (string, map[string]string, error) {
	args := splitLine(line)
	name, rest := args[0], args[1:]
	if len(rest) == 0 {
		return "", nil, fmt.Errorf("%s: expected key=value pair after name", name)
	}

	kv, err := parseKeyVal(rest)
	if err != nil {
		return "", nil, fmt.Errorf("%s: %v", name, err)
	}
	return name, kv, nil
}

func parseKeyVal(args []string) (map[string]string, error) {
	m := make(map[string]string)
	for _, kv := range args {
		if strings.HasPrefix(kv, "=") {
			return nil, fmt.Errorf("expected key=value pair, missing key %q", kv)
		}
		if strings.HasSuffix(kv, "=") {
			return nil, fmt.Errorf("expected key=value pair, missing value %q", kv)
		}
		args := strings.Split(kv, "=")
		switch len(args) {
		case 2:
			key := args[0]
			if v, ok := m[key]; ok {
				return nil, fmt.Errorf("duplicate key=value pair, have \"%s=%s\" got %q", key, v, kv)
			}
			m[key] = args[1]
		default:
			return nil, fmt.Errorf("expected key=value pair, got %q", kv)
		}
	}
	return m, nil
}

func isLetterOrNumber(r byte) bool {
	switch {
	case r > '0'-1 && r < '9'+1:
		return true
	case r > 'a'-1 && r < 'z'+1:
		return true
	case r > 'A'-1 && r < 'Z'+1:
		return true
	default:
		return false
	}
}

// splitLine is like strings.Split(string, " "), but splits
// strings by any whitespace characters, discarding them in
// the process.
func splitLine(line string) []string {
	var s []string
	var start, end int
	for ; start < len(line); start++ {
		c := line[start]
		if !isWhitespace(c) {
			break
		}
	}
	var ws bool
	for end = start; end < len(line); end++ {
		c := line[end]
		if !isWhitespace(c) {
			ws = false
			continue
		}
		if ws == true {
			start++
			continue
		}
		ws = true
		s = append(s, line[start:end])
		start = end + 1
	}
	if start != end {
		s = append(s, line[start:end])
	}
	return s
}

func isWhitespace(c byte) bool { return c == ' ' || c == '\t' }
