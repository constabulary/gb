package gb

import (
	"path/filepath"
	"testing"
)

type testproject struct {
	Project
}

func testProject(t *testing.T) *testproject {
	cwd := getwd(t)
	root := filepath.Join(cwd, "testdata")
	return &testproject{
		&project{
			rootdir: root,
		},
	}
}
