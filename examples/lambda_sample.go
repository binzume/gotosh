package main

import "fmt"

func invoke(f func(string, string), msg string) { f(msg, "world") }

func main() {
	f := func(msg, msg2 string) {
		if msg != "" {
			fmt.Println(msg + "," + msg2)
		}
	}
	invoke(f, "hello")
	invoke(func(msg, _ string) { fmt.Println(msg) }, "HELLO")
}
