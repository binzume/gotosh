package main

import (
	"fmt"

	"github.com/binzume/gotobsh/bash"
)

func printLine(n int, s string) {
	fmt.Println("Line", n, ":", s)
}

func main() {
	for i := 1; ; i++ {
		s, status := bash.Read()
		if status != 0 {
			break
		}
		printLine(i, s)
	}
}
