package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"

	"github.com/binzume/gotosh/shell"
)

func routine1(w *os.File) {
	for i := 0; i < 6; i++ {
		shell.Sleep(0.5)
		fmt.Println("write", i)
		w.WriteString("data" + strconv.Itoa(i) + "\n")
	}
	fmt.Println("finished.")
	w.Close()
}

func main() {
	r, w, err := os.Pipe()
	if err != nil {
		fmt.Println("os.Pipe() error", err)
		return
	}
	go routine1(w)
	if runtime.Compiler == "gotosh" {
		// Workaroud: the fd shoud be closed from both processes
		w.Close()
	}

	fmt.Println("Waiting...")
	for {
		ret, err := shell.ReadLine(r)
		if err != 0 {
			break
		}
		fmt.Println("read:", ret)
	}
	r.Close()
	fmt.Println("ok.")
}
