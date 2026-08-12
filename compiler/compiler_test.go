package compiler

import (
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

func Test_readType(t *testing.T) {
	fixture := []struct {
		src  string
		t    Type
		next rune
	}{
		{"", "", scanner.EOF},
		{"int=1", "int", '='},
		{"[]int{}", "[]int", '{'},
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
		{"1.5 * 1.5", shExpression{expr: "1.5*1.5", typ: "FLOAT_EXPR", retTypes: []Type{"float32"}}, scanner.EOF},
		{`"ABC" == "DEF"`, shExpression{expr: `"ABC" == "DEF"`, typ: "STR_CMP", retTypes: []Type{"bool"}}, scanner.EOF},
		{`f(1,"abc", x, s)`, shExpression{expr: `f 1 "abc" $x "$s"`, typ: "", retTypes: []Type{"int"}}, scanner.EOF},
		{`f`, shExpression{expr: `f`, typ: "", retTypes: []Type{"func(int, string)"}}, scanner.EOF},
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
		if len(e.retTypes) != len(f.t.retTypes) {
			t.Errorf("readExpression: %v != %v", f.t.retTypes, e.retTypes)
		} else {
			for i, tt := range e.retTypes {
				if tt != f.t.retTypes[i] {
					t.Errorf("readExpression: %v != %v", tt, f.t.retTypes[i])
				}
			}

		}
		if tok := s.Scan(); tok != f.next {
			t.Errorf("readExpression %v != %v", f.next, tok)
		}
	}
}
