package main

import (
	"fmt"
	"os"

	"github.com/binzume/gotosh/compiler"
)

func main() {
	err := compiler.CompileFiles(os.Args[1:])
	if err != nil {
		fmt.Println(err)
	}
}
