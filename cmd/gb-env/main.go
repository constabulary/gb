package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GB_") {
			fmt.Println(e)
		}
	}
}
