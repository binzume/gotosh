package main

import (
	"fmt"
	"strings"
)

func getMessage(name string) string {
	return "hi, " + name + "!"
}

func main() {
	s := "abcdefghijk\nxyz"
	fmt.Println("s=", s)
	fmt.Println("len(s)", len(s))
	fmt.Println("slice string", s[1+2:5+3])

	s0 := "123" + "ABC" + "\"qqq\""
	s0 = "zzz" + s0 + " yyy"
	fmt.Println("s=", s0)
	fmt.Println("to upper", strings.ToUpper(s0))
	fmt.Println("to lower", strings.ToLower(s0))
	fmt.Println("replace", strings.ReplaceAll(s0, "z", "0"))
	fmt.Println("trim prefix", strings.TrimPrefix(s0, "zzz"))
	fmt.Println("trim suffix", strings.TrimSuffix(s0, "yyy"))
	if strings.Contains(s0, "ABC\"") {
		fmt.Println("contains ABC\"")
	}
	if strings.Contains(s0, "ABCD") {
		fmt.Println("contains ABCD")
	}
	fmt.Println("indexAny", strings.IndexAny(s0, "BCD"))
	s0 = "  " + s0 + "  "
	fmt.Println("s=", s0)
	fmt.Println("trim space", strings.TrimSpace(s0))

	//fmt.Println("split", strings.Split(s0, "B")) // not POSIX

	fmt.Println("msg:", getMessage("foobar"))

	fmt.Print(123, 456)
	fmt.Println("#Println#", 123)
	fmt.Printf("#Printf %d %04d %s\n", 123, 45, "#test#")
	fmt.Sprint("#Println#", "#test#")
	fmt.Sprintf("#Printf %d %04d %s", 123, 45, "#test#")
	s1 := fmt.Sprint("#Sprint#")
	s2 := fmt.Sprintln("#Sprintln#", "#test#")
	s3 := fmt.Sprintf("#Sprintf %d %04d %s", 123, 45, "#test#")
	fmt.Println(s1, s2, s3)
}
