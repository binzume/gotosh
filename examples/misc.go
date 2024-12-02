package main

import (
	"fmt"
	"reflect"

	"github.com/binzume/gotosh/bash"
)

func testFunc(x, y int, z string) {
	fmt.Println("testFunc:", x, y, z)
}

func addInt(x, y int) int {
	return x + y
}

func concatStr(x, y string) string {
	return x + y
}

func addInt2(x, y int) bash.StatusCode {
	fmt.Println("adding", x, "and", y)
	return bash.StatusCode(x + y)
}

func returnStringAndStatus() (string, bash.StatusCode) {
	return "aaa", 123
}

// Same as bash.StatusCode
type StatusCode = int8

func returnStringAndStatus2() (StatusCode, string) {
	return 111, "bbb"
}

func returnStringMulti() (bash.TempVarString, bash.TempVarString, bash.TempVarString) {
	return "abc", "def", "ghi"
}

// only available int or string type...
type User string

func (a User) Hello() {
	fmt.Println("I am " + a + ".")
}

func main() {
	// func call
	testFunc(1, 2, "test")
	var n = int(111 + 222*3)
	fmt.Println(addInt(n-444, 999))

	fmt.Println(addInt2(4, 5))
	addInt2(6, 7)

	msg, status := returnStringAndStatus()
	fmt.Println(msg, status)

	status2, msg2 := returnStringAndStatus2()
	fmt.Println(status2, msg2)

	msg3, msg4, msg5 := returnStringMulti()
	fmt.Println(msg3, msg4, msg5)

	// method call
	var t User = "test"
	t.Hello()

	// for debugging
	fmt.Println(reflect.TypeOf(msg))
}
