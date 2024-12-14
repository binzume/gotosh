package main

import (
	"fmt"
	"os"

	"github.com/binzume/gotosh/shell"
)

func printLine(n int, s string) {
	fmt.Println("Line", n, ":", s)
}

func main() {
	fname := "file_test.txt"
	w, err := os.Create(fname)
	fmt.Fprint(w, "This is ")
	fmt.Fprintln(w, "a test file.")
	w.WriteString("Hello,")
	w.WriteString("world!\n")
	w.Close()

	r, err := os.Open(fname)
	if err != nil {
		fmt.Println("Can't open file")
		return
	}

	for i := 1; ; i++ {
		s, status := shell.ReadLine(r)
		if status != 0 {
			break
		}
		printLine(i, s)
	}
	r.Close()

	os.Remove(fname)

	os.MkdirAll("testdir/foo/bar", 0)
	os.RemoveAll("testdir")
}
