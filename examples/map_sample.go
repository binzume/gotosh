package main

import "fmt"

func testMap(m map[string]string) {
	fmt.Println("func testMap", m["abc"], m["ZZZ"])
	fmt.Println("func testMap", len(m))
	fmt.Println("func testMap", len(map[string]int{}))
	fmt.Println("func testMap", len(map[string]int{"a": 1, "b": 2, "c": 3}))
	m["hello"] = "world"
}

func main() {
	a := map[string]string{"abc": "def"}
	testMap(a)
	fmt.Println(a["hello"])
	b := a
	a["aaa"] = "bbb"
	fmt.Println(b["aaa"], b["abc"])
}
