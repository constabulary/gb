// Package gb is a tool kit for compiling and testing Go programs.
//
// The executable, cmd/gb, is located in the respective subdirectory
// along with several plugin programs.
package gb

import "io/ioutil"

func mktmpdir() string {
	d, err := ioutil.TempDir("", "gb")
	if err != nil {
		Fatalf("could not create temporary directory: %v", err)
	}
	return d
}
