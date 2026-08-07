package main

import "fmt"

type PointerTest struct {
	Name  string `tag:"name field"`
	Value int    `tag:"name value"`
}

func (p *PointerTest) S() {
	fmt.Println(p.Name)
	p.Value = 123
	p.Name += ", world"
}

func Add(p *int, n int, s *string) *int {
	*p += n
	*p++
	*s += "a"
	return p
}

func main() {
	p := &PointerTest{"hell2o", 0}
	p.S()
	x := 10
	b := &x
	msg := "a"
	z := Add(&x, 2, &msg)
	x += 200
	fmt.Println(x, msg, "age", p.Name, p.Value, "p", *b, 2**b, *z)
}
