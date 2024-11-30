package main

import (
	"fmt"
)

func getSlice() []int {
	a := []int{1, 2, 3, 4}
	return a
}

func main() {
	var a = []int{1, 2, 3, 4}
	a = append(a, 123, 456, 789)
	a = append(a, 111, 444, 777)
	fmt.Println("len", len(a))
	for i := 0; i < len(a); i++ {
		fmt.Println(i, a[i])
	}
	fmt.Println("slice", a[1:2])

	// range not supported yet
	// for i, v := range a {
	//	fmt.Println(i, v)
	//}

	var ss []string
	ss = append(ss, "abc", "def", "ghi")
	fmt.Println("len", len(ss))
	for i := 0; i < len(ss); i++ {
		fmt.Println(i, ss[i])
	}

	fmt.Println(a)
	fmt.Println(ss)
	a = getSlice()
	fmt.Println(a)

	s := "abcdefghijk"
	fmt.Println(s[1+2 : 5+3])
}
