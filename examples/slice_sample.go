package main

import (
	"fmt"
	"strings"
)

func printSlice(msg string, s []int) {
	fmt.Println(msg, "len:", len(s), "values:", s)
}

func getSlice() []int {
	return []int{1, 2, 3, 4}
}

func main() {
	var a = []int{1, 2, 3, 4}
	a = append(a, 123, 456, 789)
	a = append(a, 111, 444, 777)
	printSlice("a:", a)
	a[1] = 111
	a[1+1] = 222
	a[3] = 333
	printSlice("a:", a)
	a = getSlice()
	printSlice("a:", a)

	for i, v := range a {
		fmt.Println(i, v)
	}

	// range is not supported yet
	// for i, v := range a {
	//	fmt.Println(i, v)
	//}

	var ss []string
	ss = append(ss, "abc", "def", "ghi")
	fmt.Println("len", len(ss))
	for i := 0; i < len(ss); i++ {
		fmt.Println(i, ss[i])
	}
	fmt.Println(strings.Join(ss, ":"))

	s := "abcdefghijk:01234:uvwxyz"
	fmt.Println(s[1+2 : 5+3])
	sa := strings.Split(s, ":")

	fmt.Println("split1", sa)
	fmt.Println("split2", strings.Split(s, ":"))
}
