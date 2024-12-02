package main

import (
	"fmt"

	"github.com/binzume/gotosh/bash"
)

func routine(i int) {
	fmt.Println("a")
	bash.Sleep(1.0)
	fmt.Println("b")
	bash.Sleep(1.0)
	fmt.Println("c")
}

func main() {
	for i := 0; i < 10; i++ {
		go routine(i)
	}
	bash.Sleep(0.5)
	fmt.Println("Started")

	// No way to communicate with goroutines... so just sleep.
	bash.Sleep(3)

	fmt.Println("Finished!(maybe)")
}
