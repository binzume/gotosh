package main

import (
	"fmt"
	"os"

	"github.com/binzume/gotobsh/bash"
)

func main() {
	err := bash.CompileFile(os.Args[1])
	if err != nil {
		fmt.Println(err)
	}
}
