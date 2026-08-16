package compiler

import (
	"bytes"
	"slices"
	"strings"
	"testing"
	"text/scanner"
)

func Test_types(t *testing.T) {
	if et := Type("[]string").ElementType(); et != "string" {
		t.Errorf("ElementType: %v != %v", "string", et)
	}
	if et := Type("map[string]string").ElementType(); et != "string" {
		t.Errorf("ElementType: %v != %v", "string", et)
	}
	if et := Type("map[int]int").ElementType(); et != "int" {
		t.Errorf("ElementType: %v != %v", "int", et)
	}
}

func TestAnonymousFunction(t *testing.T) {
	const src = `package main
import "fmt"
func invoke(f func(string), msg string) { f(msg) }
func main() {
  f := func(msg string) {
    if msg != "" {
      fmt.Println(msg)
    }
  }
  invoke(f, "hello")
  invoke(func(msg string) { fmt.Println(msg) }, "world")
}`
	s := newState()
	var out bytes.Buffer
	s.w = &out
	if err := s.Compile(strings.NewReader(src), "test.go"); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{
		"GOTOSH_ANON_0() {",
		"local msg=\"$1\"; shift",
		"$f \"$msg\"",
		"f=GOTOSH_ANON_0",
		"invoke $f \"hello\"",
		"invoke GOTOSH_ANON_1 \"world\"",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("compiled output does not contain %q:\n%s", want, got)
		}
	}
}

func Test_readType(t *testing.T) {
	fixture := []struct {
		src  string
		t    Type
		next rune
	}{
		{"", "", scanner.EOF},
		{"int=1", "int", '='},
		{"[]int{}", "[]int", '{'},
		{"[123+456]int{}", "[]int", '{'},
		{"map[string]string{}", "map[string]string", '{'},
		{"struct{Func func(string)} *", "struct{:Func:func(string):}", '*'},
		{"func() {", "func()", '{'},
		{"func(int) {", "func(int)", '{'},
		{"func(x int) {", "func(int)", '{'},
		{"func(x,y int) {", "func(int, int)", '{'},
		{"func(x int,y string) {", "func(int, string)", '{'},
		{"func(int,string) {", "func(int, string)", '{'},
		{"func(x int) int {", "func(int)int", '{'},
		{"func(x *int) *int {", "func(*int)*int", '{'},
		{"func(x int)(string, error) {", "func(int)(string, error)", '{'},
		{"func(struct{A int}) {", "func(struct{:A:int:})", '{'},
		{"func(f func()) {", "func(func())", '{'},
		{"func(A ...int) {", "func(...int)", '{'},
		{"func(A ...int)\nvar a int", "func(...int)", scanner.Ident},
		{"{", "", '{'},
	}
	for _, f := range fixture {
		s := newState()
		s.Init(strings.NewReader(f.src))
		if typ := s.readType(false); typ != f.t {
			t.Errorf("readType: %v != %v", f.t, typ)
		}
		if tok := s.Scan(); tok != f.next {
			t.Errorf("nextToken %v != %v", f.next, tok)
		}
	}
}

func Test_readExpression(t *testing.T) {
	fixture := []struct {
		src  string
		t    shExpression
		next rune
	}{
		{"1 + 1", shExpression{expr: "1+1", typ: "INT_EXPR", retTypes: []Type{"int"}}, scanner.EOF},
		{"1 == 1", shExpression{expr: "1 == 1", typ: "INT_EXPR", retTypes: []Type{"bool"}}, scanner.EOF},
		{"!true", shExpression{expr: "!1", typ: "INT_EXPR", retTypes: []Type{"bool"}}, scanner.EOF},
		{"1.5 * 1.5", shExpression{expr: "1.5*1.5", typ: "FLOAT_EXPR", retTypes: []Type{"float32"}}, scanner.EOF},
		{`"ABC" == "DEF"`, shExpression{expr: `"ABC" == "DEF"`, typ: "STR_CMP", retTypes: []Type{"bool"}}, scanner.EOF},
		{`len([]int{1,2,3})`, shExpression{expr: `3`, retTypes: []Type{"int"}}, scanner.EOF},
		{`len(map[int]int{1:2, 2:3})`, shExpression{expr: `2`, retTypes: []Type{"int"}}, scanner.EOF},
		{`len(map[int]int{})`, shExpression{expr: `0`, retTypes: []Type{"int"}}, scanner.EOF},
		{`[]int{1,2,3}`, shExpression{expr: ``, retTypes: []Type{"[]int"}, values: []string{"1", "2", "3"}}, scanner.EOF},
		{`map[int]int{1:2}`, shExpression{expr: ``, retTypes: []Type{"map[int]int"}, values: []string{"1", "2"}}, scanner.EOF},
		{`f(1,"abc", x, s)`, shExpression{expr: `f 1 "abc" $x "$s"`, typ: "", retTypes: []Type{"int"}}, scanner.EOF},
		{`f`, shExpression{expr: `f`, typ: "", retTypes: []Type{"func(int, string)int"}}, scanner.EOF},
		{`int(123.4)`, shExpression{expr: `printf '%.0f' 123.4`, typ: "", retTypes: []Type{"int"}}, scanner.EOF},
		{`string('s')`, shExpression{expr: `'s'`, typ: "", retTypes: []Type{"string"}}, scanner.EOF},
		{`float64(123.4)`, shExpression{expr: `123.4`, typ: "", retTypes: []Type{"float64"}}, scanner.EOF},
		{`float32(123.4)`, shExpression{expr: `123.4`, typ: "", retTypes: []Type{"float32"}}, scanner.EOF},
	}
	for _, f := range fixture {
		s := newState()
		s.funcs["f"] = shExpression{expr: "f", argTypes: []Type{"int", "string"}, retTypes: []Type{"int"}}
		s.vars["x"] = TypedName{"x", "int"}
		s.vars["s"] = TypedName{"s", "string"}
		s.Init(strings.NewReader(f.src))
		e := s.readExpression("", ";", false)
		if e.expr != f.t.expr || e.typ != f.t.typ {
			t.Errorf("readExpression: %v != %v", f.t, e)
		}

		if !slices.Equal(f.t.retTypes, e.retTypes) {
			t.Errorf("readExpression.retTypes: %v != %v", f.t.retTypes, e.retTypes)
		}
		if !slices.Equal(f.t.values, e.values) {
			t.Errorf("readExpression.values: %v != %v", f.t.values, e.values)
		}
		if tok := s.Scan(); tok != f.next {
			t.Errorf("readExpression %v != %v", f.next, tok)
		}
	}
}
