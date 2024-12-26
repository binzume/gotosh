package main

import (
	"fmt"

	"github.com/binzume/gotosh/shell"
)

func printArgs(_ []int) {
	// NOTE: slice type variable is supported only on bash. so use shell.Args() here
	for i, v := range shell.Args() {
		fmt.Println("printSlice", i, v)
	}
}

func main() {
	fmt.Println("loop 1:")
	for i := 0; i < 10; i++ {
		fmt.Println(i)
	}

	fmt.Println("loop 2:")
	i := 0
	for i < 10 {
		fmt.Println(i)
		i++
	}

	fmt.Println("loop 3:")
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

	fmt.Println("slice literal 1:")
	for i, v := range []int{2, 4, 6, 8, 10, 12, 14} {
		fmt.Println(i, v)
	}

	fmt.Println("slice literal 2:")
	for i := range []int{2, 4, 6, 8, 10, 12, 14} {
		fmt.Println(i)
	}

	fmt.Println("slice literal 3:")
	for range []int{1, 2, 3} {
		fmt.Println(".")
	}

	fmt.Println("slice literal 2:")
	for i := range []int{2, 4, 6, 8, 10, 12, 14} {
		fmt.Println(i)
	}

	fmt.Println("array literal:")
	for i, v := range [3]int{2, 4, 6} {
		fmt.Println(i, v)
	}

	if !shell.IsShellScript {
		shell.SetArgs("4", "5", "6", "7", "8", "9")
	}
	printArgs([]int{4, 5, 6, 7, 8, 9})
}
