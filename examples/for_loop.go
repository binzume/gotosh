package main

import (
	"fmt"

	"github.com/binzume/gotosh/shell"
)

func main() {
	fmt.Println("loop1:")
	for i := 0; i < 10; i++ {
		fmt.Println(i)
	}

	fmt.Println("loop2:")
	i := 0
	for i < 10 {
		fmt.Println(i)
		i++
	}

	fmt.Println("loop3:")
	i = 0
	for {
		if i > -10 {
			break
		}
		fmt.Println(i)
		i++
	}

	fmt.Println("shell.Args():")
	for i, v := range shell.Args() {
		fmt.Println(i, v)
	}

	fmt.Println("slice literal:")
	for i, v := range []int{2, 4, 6, 8, 10, 12, 14} {
		fmt.Println(i, v)
	}

	fmt.Println("array index:")
	for i := range [7]int{2, 4, 6, 8, 10, 12, 14} {
		fmt.Println(i)
	}
}
