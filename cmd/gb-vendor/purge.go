package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/cmd/gb-vendor/vendor"
)

// Packages from the standard lib.
var stdlib = map[string]bool{
	"C":                   true,
	"archive/tar":         true,
	"archive/zip":         true,
	"bufio":               true,
	"bytes":               true,
	"compress/bzip2":      true,
	"compress/flate":      true,
	"compress/gzip":       true,
	"compress/lzw":        true,
	"compress/zlib":       true,
	"container/heap":      true,
	"container/list":      true,
	"container/ring":      true,
	"crypto":              true,
	"crypto/aes":          true,
	"crypto/cipher":       true,
	"crypto/des":          true,
	"crypto/dsa":          true,
	"crypto/ecdsa":        true,
	"crypto/elliptic":     true,
	"crypto/hmac":         true,
	"crypto/md5":          true,
	"crypto/rand":         true,
	"crypto/rc4":          true,
	"crypto/rsa":          true,
	"crypto/sha1":         true,
	"crypto/sha256":       true,
	"crypto/sha512":       true,
	"crypto/subtle":       true,
	"crypto/tls":          true,
	"crypto/x509":         true,
	"crypto/x509/pkix":    true,
	"database/sql":        true,
	"database/sql/driver": true,
	"debug/dwarf":         true,
	"debug/elf":           true,
	"debug/gosym":         true,
	"debug/macho":         true,
	"debug/pe":            true,
	"encoding":            true,
	"encoding/ascii85":    true,
	"encoding/asn1":       true,
	"encoding/base32":     true,
	"encoding/base64":     true,
	"encoding/binary":     true,
	"encoding/csv":        true,
	"encoding/gob":        true,
	"encoding/hex":        true,
	"encoding/json":       true,
	"encoding/pem":        true,
	"encoding/xml":        true,
	"errors":              true,
	"expvar":              true,
	"flag":                true,
	"fmt":                 true,
	"go/ast":              true,
	"go/build":            true,
	"go/doc":              true,
	"go/format":           true,
	"go/parser":           true,
	"go/printer":          true,
	"go/scanner":          true,
	"go/token":            true,
	"hash":                true,
	"hash/adler32":        true,
	"hash/crc32":          true,
	"hash/crc64":          true,
	"hash/fnv":            true,
	"html":                true,
	"html/template":       true,
	"image":               true,
	"image/color":         true,
	"image/draw":          true,
	"image/gif":           true,
	"image/jpeg":          true,
	"image/png":           true,
	"index/suffixarray":   true,
	"io":                  true,
	"io/ioutil":           true,
	"log":                 true,
	"log/syslog":          true,
	"math":                true,
	"math/big":            true,
	"math/cmplx":          true,
	"math/rand":           true,
	"mime":                true,
	"mime/multipart":      true,
	"net":                 true,
	"net/http":            true,
	"net/http/cgi":        true,
	"net/http/cookiejar":  true,
	"net/http/fcgi":       true,
	"net/http/httptest":   true,
	"net/http/httputil":   true,
	"net/http/pprof":      true,
	"net/mail":            true,
	"net/rpc":             true,
	"net/rpc/jsonrpc":     true,
	"net/smtp":            true,
	"net/textproto":       true,
	"net/url":             true,
	"os":                  true,
	"os/exec":             true,
	"os/signal":           true,
	"os/user":             true,
	"path":                true,
	"path/filepath":       true,
	"reflect":             true,
	"regexp":              true,
	"regexp/syntax":       true,
	"runtime":             true,
	"runtime/cgo":         true,
	"runtime/debug":       true,
	"runtime/pprof":       true,
	"sort":                true,
	"strconv":             true,
	"strings":             true,
	"sync":                true,
	"sync/atomic":         true,
	"syscall":             true,
	"testing":             true,
	"testing/iotest":      true,
	"testing/quick":       true,
	"text/scanner":        true,
	"text/tabwriter":      true,
	"text/template":       true,
	"text/template/parse": true,
	"time":                true,
	"unicode":             true,
	"unicode/utf16":       true,
	"unicode/utf8":        true,
	"unsafe":              true,
}

func init() {
	registerCommand(PurgeCmd)
}

func parseImports(root string) (map[string]struct{}, error) {
	var found = struct{}{}            // Does not take any space
	pkgs := make(map[string]struct{}) // Set

	var walkFn = func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" { // Parse only go source files
			return nil
		}

		fs := token.NewFileSet()
		f, err := parser.ParseFile(fs, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}

		for _, s := range f.Imports {
			if _, ok := stdlib[s.Path.Value]; !ok {
				pkgs[strings.Replace(s.Path.Value, "\"", "", -1)] = found
			}
		}
		return nil
	}

	if err := filepath.Walk(root, walkFn); err != nil {
		return pkgs, err
	}

	return pkgs, nil
}

var PurgeCmd = &cmd.Command{
	Name:      "purge",
	ShortDesc: "purges all unreferenced dependencies",
	Run: func(ctx *gb.Context, args []string) error {
		m, err := vendor.ReadManifest(manifestFile(ctx))
		if err != nil {
			return fmt.Errorf("could not load manifest: %v", err)
		}

		imports, err := parseImports(ctx.Projectdir())
		if err != nil {
			return fmt.Errorf("import could not be parsed: %v", err)
		}

		dependencies := make([]vendor.Dependency, len(m.Dependencies))
		copy(dependencies, m.Dependencies)

		for _, d := range dependencies {
			if _, ok := imports[d.Importpath]; !ok {
				dep, err := m.GetDependencyForImportpath(d.Importpath)
				if err != nil {
					return fmt.Errorf("could not get get dependency: %v", err)
				}

				if err := m.RemoveDependency(dep); err != nil {
					return fmt.Errorf("dependency could not be removed: %v", err)
				}

				localClone := vendor.GitClone{
					Path: filepath.Join(ctx.Projectdir(), "vendor", "src", dep.Importpath),
				}
				if err := localClone.Destroy(); err != nil {
					return fmt.Errorf("dependency could not be deleted: %v", err)
				}
			}
		}

		return vendor.WriteManifest(manifestFile(ctx), m)
	},
}
