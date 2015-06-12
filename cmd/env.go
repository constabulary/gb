package cmd

import (
	"os"

	"github.com/constabulary/gb"
)

func MustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		gb.Fatalf("unable to determine current working directory: %v", err)
	}
	return wd
}
