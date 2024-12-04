package main

import (
	"fmt"
	"os"

	"github.com/binzume/gotosh/bash"
)

func printLine(n int, s string) {
	fmt.Println("Line", n, ":", s)
}

func main() {
	w, err := os.Create("file_test.txt")
	w.WriteString("This is a test file.\n")
	w.WriteString("Hello,")
	w.WriteString("world!\n")
	w.Close()

	r, err := os.Open("file_test.txt")
	if err != nil {
		fmt.Println("Can't open file")
		return
	}

	for i := 1; ; i++ {
		s, status := bash.ReadLine(r)
		if status != 0 {
			break
		}
		printLine(i, s)
	}
	r.Close()

}
