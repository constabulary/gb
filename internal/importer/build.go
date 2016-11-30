// Copyright 2011 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package importer

import (
	"bytes"
	"fmt"
	"go/build"
	"path/filepath"
	"strings"
)

// A Package describes the Go package found in a directory.
type Package struct {
	*build.Package
	Standard bool // package found in GOROOT
}

// MultiplePackageError describes a directory containing
// multiple buildable Go source files for multiple packages.
type MultiplePackageError struct {
	Dir      string   // directory containing files
	Packages []string // package names found
	Files    []string // corresponding files: Files[i] declares package Packages[i]
}

func (e *MultiplePackageError) Error() string {
	// Error string limited to two entries for compatibility.
	return fmt.Sprintf("found packages %s (%s) and %s (%s) in %s", e.Packages[0], e.Files[0], e.Packages[1], e.Files[1], e.Dir)
}

// expandSrcDir expands any occurrence of ${SRCDIR}, making sure
// the result is safe for the shell.
func expandSrcDir(str string, srcdir string) (string, bool) {
	// "\" delimited paths cause safeCgoName to fail
	// so convert native paths with a different delimeter
	// to "/" before starting (eg: on windows).
	srcdir = filepath.ToSlash(srcdir)

	// Spaces are tolerated in ${SRCDIR}, but not anywhere else.
	chunks := strings.Split(str, "${SRCDIR}")
	if len(chunks) < 2 {
		return str, safeCgoName(str, false)
	}
	ok := true
	for _, chunk := range chunks {
		ok = ok && (chunk == "" || safeCgoName(chunk, false))
	}
	ok = ok && (srcdir == "" || safeCgoName(srcdir, true))
	res := strings.Join(chunks, srcdir)
	return res, ok && res != ""
}

// NOTE: $ is not safe for the shell, but it is allowed here because of linker options like -Wl,$ORIGIN.
// We never pass these arguments to a shell (just to programs we construct argv for), so this should be okay.
// See golang.org/issue/6038.
const safeString = "+-.,/0123456789=ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz:$"
const safeSpaces = " "

var safeBytes = []byte(safeSpaces + safeString)

func safeCgoName(s string, spaces bool) bool {
	if s == "" {
		return false
	}
	safe := safeBytes
	if !spaces {
		safe = safe[len(safeSpaces):]
	}
	for i := 0; i < len(s); i++ {
		if c := s[i]; c < 0x80 && bytes.IndexByte(safe, c) < 0 {
			return false
		}
	}
	return true
}
