package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

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

// Implements strings.Index()
func GOTOSH_FUNC_strings_Index(s, f string) int {
	fl := len(f)
	end := len(s) - fl + 1
	for i := 0; i < end; i++ {
		if s[i:i+fl] == f {
			return i
		}
	}
	return -1
}

func returnStringAndStatus2() (StatusCode, string) {
	return 111, "bbb"
}

func returnStringMulti() (bash.TempVarString, bash.TempVarString, bash.TempVarString) {
	return "abc", "def", "ghi"
}

type Person struct {
	Name string
	Age  int
}

func (a Person) Hello() {
	fmt.Println("I am " + a.Name + "(" + strconv.Itoa(a.Age) + ").")
}

func main() {
	//  args
	for i := 1; i < bash.NArgs(); i++ {
		fmt.Println("arg", i, bash.Arg(i))
	}

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

	// method call TODO: struct support
	var p Person
	p.Name = "test"
	p.Age = 123
	d := p
	d.Hello()

	// for debugging
	fmt.Println(reflect.TypeOf(msg))

	fmt.Println(strings.Index("hello, world", "ld"))

	// TODO: remove "(,)"
	if (("") == "a") || (1+1 == 2) {
		fmt.Println("true")
	}
}
