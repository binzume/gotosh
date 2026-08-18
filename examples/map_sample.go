package main

import "fmt"

func testMap(m map[string]string) {
	fmt.Println("testMap lookup", m["abc"], m["ZZZ"])
	fmt.Println("testMap len()", len(m))
	fmt.Println("testMap len()", len(map[string]int{}))
	fmt.Println("testMap len()", len(map[string]int{"a": 1, "b": 2, "c": 3}))
	m["hello"] = "world"
}

func main() {
	a := map[string]string{"abc": "def"}
	testMap(a)
	fmt.Println(a["hello"])
	b := a
	a["aaa"] = "bbb"
	fmt.Println(b["aaa"], b["abc"])

	m := map[string]string{"key": "value"}
	for k, v := range m {
		fmt.Println(k, v)
	}
}
