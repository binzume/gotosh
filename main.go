package main

import (
	"fmt"
	"os"

	"github.com/binzume/gotobsh/compiler"
)

func main() {
	err := compiler.CompileFile(os.Args[1])
	if err != nil {
		fmt.Println(err)
	}
}
