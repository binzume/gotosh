package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/binzume/gotosh/shell"
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

func addInt2(x, y int) shell.StatusCode {
	fmt.Println("adding", x, "and", y)
	return shell.StatusCode(x + y)
}

func returnStringAndStatus() (string, shell.StatusCode) {
	return "aaa", 123
}

// Implements strings.Index()
func GOTOSH_FUNC_strings_Index(s, f string) int {
	i := 0
	shell.Do(`_tmp=${s#*$f}; i=$((${#s}-${#_tmp}-${#f}))`)
	if i < 0 {
		return -1
	}
	return i
}

func returnStringAndStatus2() (shell.StatusCode, string) {
	return 111, "bbb"
}

func returnStringMulti() (shell.TempVarString, shell.TempVarString, shell.TempVarString) {
	return "abc", "def", "ghi"
}

type Date struct {
	// TODO: support "Year, Mmonth, Day int"
	Year  int
	Month int
	Day   int
}
type Person struct {
	Name     string `tag:"aa"`
	Age      int
	Birthday Date
}

func NewPerson(name string, age int) Person {
	// TODO: support "Person{Name: name, Age: age}"
	return Person{name, age, Date{2001, 2, 3}}
}

func (a Person) Hello() {
	fmt.Println("I am " + a.Name + "(" + strconv.Itoa(a.Age) + ").")
	fmt.Println(" ", a.Birthday.Year, a.Birthday.Month, a.Birthday.Day)
}

func main() {
	//  args
	for i := 1; i < shell.NArgs(); i++ {
		fmt.Println("arg", i, shell.Arg(i))
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

	// struct and method call
	p := NewPerson("test", 123)
	d := p
	d.Hello()

	// for debugging
	fmt.Println(reflect.TypeOf(msg))

	fmt.Println(strings.Index("hello, wo rl'\" d!", "l'\" d"))

	// TODO: remove "(,)"
	if (("") == "a") || (1+1 == 2) {
		fmt.Println("true")
	}
}
