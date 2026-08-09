package main

import "fmt"

type PointerTest struct {
	Name  string `tag:"name field"`
	Value int    `tag:"name value"`
}

func (p *PointerTest) PtrTest() {
	p.Value += 10
	p.Name += ", world"
}

func (p PointerTest) ValueTest() {
	p.Name += ", world"
	p.Value = 123
	fmt.Println(p.Name, p.Value)
}

func Add(p *int, n int, s *string) (*int, *string) {
	*p += n
	*s += "a"
	return p, s
}

func main() {
	x := 10
	p1 := &x
	p2 := p1
	p3 := *p1
	fmt.Println(*p1, *p2, p3)
	x++
	fmt.Println(*p1, *p2, p3)
	*p2++
	fmt.Println(*p1, *p2, p3)
	p3++
	fmt.Println(*p1, *p2, p3)
	// p1, p2 = p2, p1
	// fmt.Println(*p1, *p2, p3)
	msg := "a"
	z, _ := Add(&x, 2, &msg)
	fmt.Println(*p1, *p2, p3, msg, *z)
	fmt.Println(*p1, 2**p1, *p1*3)

	p := &PointerTest{"Hello", 1}
	p.ValueTest()
	fmt.Println(p.Name, p.Value)
	p.PtrTest()
	fmt.Println(p.Name, p.Value)
}
