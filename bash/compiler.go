package bash

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"text/scanner"
)

func trimQuote(s string) string {
	return strings.Trim(s, "\"`")
}

func addQuote(s string) string {
	if strings.HasPrefix(s, "$") && !strings.HasPrefix(s, "$((") {
		return `"` + s + `"`
	}
	return s
}

func varName(s string) string {
	return trimQuote(s)[1:]
}

func escapeShellString(s string) string {
	return strings.ReplaceAll(s, "$", "\\$")
}

type shFunc struct {
	shName   string
	retTypes []string
	convFunc func(arg ...string) string
}

type shExpression struct {
	cmd      string
	retVar   string
	retTypes []string
}

func (f *shExpression) AsValue() string {
	cmd := f.cmd
	if len(f.retTypes) > 0 && f.retTypes[0] == "StatusCode" {
		// TODO: stdout...
		cmd += " >&2; echo $?"
	} else if len(f.retTypes) > 0 && f.retTypes[0] == "TempVarString" {
		cmd += " && echo \"$_tmp\""
	}
	if len(f.retTypes) > 0 && f.retTypes[0] != "_DIRECT" && f.retTypes[0] != "_ARG1" {
		cmd = "\"`" + cmd + "`\""
	}
	return strings.TrimSpace(cmd)
}

func (f *shExpression) AsExec() string {
	cmd := f.cmd
	if len(f.retTypes) > 0 && f.retTypes[0] == "StdoutString" {
		cmd += " >/dev/null"
	}
	return strings.TrimSpace(cmd)
}

type state struct {
	scanner.Scanner
	imports   map[string]string
	funcs     map[string]shFunc
	vars      map[string]string
	cl        []string
	useExFor  bool
	lastToken rune
	funcName  string
	w         io.Writer
	bufLine   string
}

func (s *state) FlushLine() {
	if s.bufLine != "" {
		t := s.bufLine
		s.bufLine = ""
		s.Indent()
		fmt.Fprintln(s.w, t)
	}
}

func (s *state) Write(str ...any) {
	s.FlushLine()
	fmt.Fprint(s.w, str...)
}

func (s *state) Writeln(str ...any) {
	s.FlushLine()
	fmt.Fprintln(s.w, str...)
}

func (s *state) Indent() {
	s.Write(strings.Repeat("  ", len(s.cl)))
}

func (s *state) EndBlock() {
	s.FlushLine()
	t := s.cl[len(s.cl)-1]
	s.cl = s.cl[:len(s.cl)-1]
	s.bufLine = t + "\n" // for "else"
	if len(s.cl) == 0 {
		s.funcName = ""
	}
}

func (s *state) parseImport() {
	tok := s.Scan()
	if tok == '(' {
		for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
			if tok == ')' {
				break
			}
			if tok == scanner.Ident {
				name := s.TokenText()
				s.Scan()
				s.imports[name] = trimQuote(s.TokenText())
			} else {
				pkg := trimQuote(s.TokenText())
				name := path.Base(pkg)
				s.imports[name] = pkg
			}
		}
	} else {
		if tok == scanner.Ident {
			name := s.TokenText()
			s.Scan()
			s.imports[name] = trimQuote(s.TokenText())
		} else {
			pkg := trimQuote(s.TokenText())
			name := path.Base(pkg)
			s.imports[name] = pkg
		}
	}
}

func (s *state) readFuncCall(pkgref, name string) *shExpression {
	if pkgref != "" {
		pkg := s.imports[pkgref]
		name = path.Base(pkg) + "." + name
	}
	f, ok := s.funcs[name]
	if ok {
		name = f.shName
	}

	var args []string
	for s.lastToken != scanner.EOF {
		args = append(args, addQuote(s.readExpression().AsValue()))
		if s.lastToken == ')' {
			break
		}
	}

	cmd := name + " " + strings.Join(args, " ")
	if f.convFunc != nil {
		cmd = f.convFunc(args...)
	}
	retVar := ""
	if len(f.retTypes) > 0 && f.retTypes[0] == "_ARG1" && len(args) > 0 {
		retVar = varName(args[0])
	}
	return &shExpression{cmd: cmd, retTypes: f.retTypes, retVar: retVar}
}

func (s *state) readExpression() *shExpression {
	exp := ""
	tokens := 0
	nest := 0
	var funcRet *shExpression
	var mode rune = scanner.Int
	if s.Peek() == 13 || s.Peek() == 10 {
		return &shExpression{cmd: ""}
	}
	for s.lastToken = s.Scan(); s.lastToken != scanner.EOF; s.lastToken = s.Scan() {
		tok := s.lastToken
		if nest == 0 && tok == ')' || tok == ',' || tok == ';' || tok == '{' {
			break
		} else if tok == '(' {
			nest++
		} else if tok == ')' {
			nest--
		}
		if mode == scanner.String && tok == '+' {
			continue
		}
		t := s.TokenText()
		if tok == scanner.String {
			t = escapeShellString(t)
		}
		if mode != scanner.String {
			mode = tok // TODO: detect var type
		}
		if tok == scanner.Ident {
			if t == "true" {
				tok = scanner.Int
				t = "1"
			} else if t == "false" || t == "nil" {
				tok = scanner.Int
				t = "0"
			}
			if s.vars[t] == "string" {
				mode = scanner.String
			}
		}
		if tok == scanner.Ident && (s.Peek() == '(' || s.Peek() == '.') {
			// TODO func call
			mode = scanner.String
			tok2 := s.Scan()
			pkgref := ""
			if tok2 == '.' {
				s.Scan()
				pkgref = t
				t = s.TokenText()
				s.Scan()
			}
			funcRet = s.readFuncCall(pkgref, t)
			exp += funcRet.AsValue()
		} else if tok == scanner.Ident && mode == scanner.String {
			exp += "\"$" + t + "\""
		} else {
			exp += t
		}
		tokens++
		if nest == 0 && s.Peek() == 13 || s.Peek() == 10 { // TODO
			break
		}
	}
	if tokens == 1 && mode == scanner.Ident {
		exp = "$" + exp
	} else if tokens > 1 && mode != scanner.String {
		exp = "$(( " + exp + " ))"
	}
	if funcRet != nil && exp == funcRet.AsValue() {
		return funcRet
	}
	return &shExpression{cmd: exp}
}

func (s *state) procVar(name string, readonly bool) {
	s.Indent()
	if s.funcName != "" {
		s.Write("local ")
		if readonly {
			s.Write("-r ")
		}
	}
	v := s.readExpression().AsValue()
	if s.vars[name] == "" {
		if strings.HasPrefix(v, `"`) {
			s.vars[name] = "string"
		} else {
			s.vars[name] = "" // TODO
		}
	}
	if v == "" && strings.HasPrefix(s.vars[name], "[]") {
		v = "()"
	}
	if v != "" {
		s.Writeln(name + "=" + v)
	} else if s.funcName != "" {
		s.Writeln(name)
	}
}

func (s *state) procReturn() {
	v := s.readExpression().AsValue()
	retTypes := s.funcs[s.funcName].retTypes
	if len(retTypes) > 0 && retTypes[0] == "StatusCode" {
		s.Indent()
		s.Writeln("return ", v)
	} else {
		s.Indent()
		if v != "" {
			s.Write("echo " + v + ";") // TODO _ret=?
		}
		s.Writeln("return")
	}

}

func (s *state) procFunc() {
	name := ""
	arg := ""
	for tok := s.Scan(); tok != scanner.EOF && tok != ')'; tok = s.Scan() {
		if name == "" && tok == scanner.Ident {
			name = s.TokenText()
			if name == "main" {
				s.Writeln("# function " + name + "()")
				s.cl = append(s.cl, "# end of "+name)
			} else {
				s.funcName = name
				s.Writeln("function " + name + "() {")
				s.cl = append(s.cl, "}")
			}
		}
		if tok == scanner.Ident && arg != "" {
			s.vars[arg] = s.TokenText()
			arg = ""
		}
		if tok == '(' || tok == ',' {
			tok = s.Scan()
			if tok == scanner.Ident {
				arg = s.TokenText()
				s.Indent()
				s.Writeln("local " + arg + `="$1"; shift`)
			} else if tok == ')' {
				break
			}
		}
	}
	retType := ""
	for tok := s.Scan(); tok != scanner.EOF && tok != '{'; tok = s.Scan() {
		if tok == scanner.Ident {
			retType = s.TokenText()
		}
	}
	s.funcs[name] = shFunc{shName: name, retTypes: []string{retType}}
}

func (s *state) procFor() {
	f := []string{"", "", ""}

	n := 0
	for s.lastToken != scanner.EOF && n < 3 {
		// TODO
		f[n] = strings.ReplaceAll(s.readExpression().AsValue(), ":=", "=")
		f[n] = strings.TrimPrefix(f[n], "$(( ")
		f[n] = strings.TrimSuffix(f[n], " ))")
		n++
		if s.lastToken == '{' {
			break
		}
	}

	if s.useExFor {
		s.Indent()
		s.Writeln("for (( " + f[0] + "; " + f[1] + "; " + f[2] + " )); do")
		s.cl = append(s.cl, "done")
		return
	}

	if f[0] != "" {
		s.Indent()
		s.Writeln(f[0])
	}
	cond := "true"
	if f[1] != "" {
		cond = "[ $(( " + f[1] + " )) -ne 0 ]"
	}
	s.Indent()
	s.Writeln("while " + cond + "; do")
	end := "done"
	if f[2] != "" {
		end = "let \"" + f[2] + "\"; done" // TODO continue...
	}
	s.cl = append(s.cl, end)
}

func (s *state) procIf() {
	condition := s.readExpression().AsValue()
	s.Indent()
	s.Writeln("if [ " + condition + " -ne 0 ]; then")
	s.cl = append(s.cl, "fi")
}

func (s *state) procElse() {
	s.bufLine = "" // cancel fi
	s.Indent()
	condition := ""
	for tok := s.Scan(); tok != scanner.EOF && tok != '{'; tok = s.Scan() {
		if tok == scanner.Ident && s.TokenText() == "if" {
			condition = s.readExpression().AsValue()
			break
		}
	}
	if condition == "" {
		s.Writeln("else")
	} else {
		s.Writeln("elif [ " + condition + " -ne 0 ]; then")
	}
	s.cl = append(s.cl, "fi")
}

func (s *state) procSentense(t string) {
	second := ""
	tok := s.Scan()
	for tok == ',' {
		s.Scan()
		second = s.TokenText()
		tok = s.Scan()
	}
	if tok == ':' {
		s.Scan()
		s.procVar(t, false)
		if second != "" {
			s.Indent()
			s.Writeln(second + "=$?")
		}
	} else if tok == '=' {
		s.Indent()
		v := s.readExpression()
		if t == v.retVar {
			s.Writeln(v.AsExec())
		} else {
			s.Writeln(t + "=" + v.AsValue())
		}
		if second != "" {
			s.Indent()
			s.Writeln(second + "=$?")
		}
	} else if tok == '(' || s.imports[t] != "" && tok == '.' {
		if tok == '.' {
			s.Scan()
			name := s.TokenText()
			s.Scan() // (
			s.Indent()
			s.Writeln(s.readFuncCall(t, name).AsExec())
		} else {
			s.Indent()
			s.Writeln(s.readFuncCall("", t).AsExec())
		}
	} else {
		fmt.Printf("# %s: %s %s\n", s.Position, s.TokenText(), scanner.TokenString(tok))
	}
}

func CompileFile(srcPath string) error {
	r, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer r.Close()
	var s state
	return s.Compile(r, srcPath)
}

func (s *state) Compile(r io.Reader, srcName string) error {

	s.Init(r)
	s.Filename = srcName
	s.Mode ^= scanner.SkipComments
	s.w = os.Stdout

	s.Writeln("#!/bin/bash")
	s.Writeln("")

	s.imports = map[string]string{}
	s.funcs = map[string]shFunc{
		"bash.Echo":    {shName: "echo"},
		"bash.EchoN":   {shName: "echo -n"},
		"bash.Printf":  {shName: "printf"},
		"bash.Sprintf": {shName: "printf", retTypes: []string{"StdoutString"}},
		"bash.Sleep":   {shName: "sleep"},
		"bash.Exit":    {shName: "exit"},
		"bash.Export":  {shName: "export"},
		"bash.Pwd":     {shName: "pwd", retTypes: []string{"StdoutString"}},
		"bash.Cd":      {shName: "cd"},
		"bash.Exec": {shName: "", retTypes: []string{"StdoutString", "StatusCode"},
			convFunc: func(arg ...string) string { return trimQuote(arg[0]) }},
		"bash.Read": {shName: `read _tmp`, retTypes: []string{"TempVarString", "StatusCode"}},
		"bash.SubStr": {shName: "", retTypes: []string{"_DIRECT"},
			convFunc: func(arg ...string) string {
				return "\"${" + varName(arg[0]) + ":" + arg[1] + ":" + arg[2] + "}\""
			}},
		// fmt
		"fmt.Println": {shName: "echo"},
		"fmt.Print":   {shName: "echo -n"},
		// os
		"os.Exit": {shName: "exit"},
		// TODO: cast
		"int":             {shName: "", retTypes: []string{"_DIRECT"}},
		"string":          {shName: "", retTypes: []string{"_DIRECT"}},
		"strconv.Atoi":    {shName: "", retTypes: []string{"_DIRECT"}},
		"strconv.Itoa":    {shName: "", retTypes: []string{"_DIRECT"}},
		"bash.StatusCode": {shName: "", retTypes: []string{"_DIRECT"}},
		// slice
		"len": {shName: "", retTypes: []string{"_DIRECT"},
			convFunc: func(arg ...string) string {
				if s.vars[varName(arg[0])] == "string" {
					return "${#" + varName(arg[0]) + "}"
				}
				return "${#" + varName(arg[0]) + "[@]}"
			}},
		"append": {shName: "", retTypes: []string{"_ARG1"},
			convFunc: func(arg ...string) string {
				return varName(arg[0]) + "+=(" + strings.Join(arg[1:], " ") + ")"
			}},
	}
	s.vars = map[string]string{}

	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		if tok == '}' && len(s.cl) > 0 {
			s.EndBlock()
		} else if tok == '{' {
			s.cl = append(s.cl, "")
		} else if tok == scanner.Comment {
			for _, c := range strings.Split(strings.Trim(s.TokenText(), "/* "), "\n") {
				s.Indent()
				s.Writeln("# " + c)
			}
		} else if tok == scanner.Ident {
			t := s.TokenText()
			switch t {
			case "package":
				s.Scan()
			case "import":
				s.parseImport()
			case "for":
				s.procFor()
			case "if":
				s.procIf()
			case "else":
				s.procElse()
			case "break":
				s.Indent()
				s.Writeln("break")
			case "continue":
				s.Indent()
				s.Writeln("continue")
			case "return":
				s.procReturn()
			case "func":
				s.procFunc()
			case "var":
				s.Scan()
				t = s.TokenText()
				s.vars[t] = ""
				if s.Peek() != '\n' && s.Peek() != '\r' {
					for tok = s.Scan(); tok != scanner.EOF && tok != '=' && s.Peek() != '\n' && s.Peek() != '\r'; tok = s.Scan() {
						s.vars[t] += s.TokenText()
					}
				}
				s.procVar(t, false)
			case "const":
				s.Scan()
				t = s.TokenText()
				s.vars[t] = ""
				if s.Peek() != '\n' && s.Peek() != '\r' {
					for tok = s.Scan(); tok != scanner.EOF && tok != '=' && s.Peek() != '\n' && s.Peek() != '\r'; tok = s.Scan() {
						s.vars[t] += s.TokenText()
					}
				}
				s.procVar(t, true)
			default:
				s.procSentense(t)
			}
		} else {
			fmt.Printf("# Unknown token %s: %s %s\n", s.Position, s.TokenText(), scanner.TokenString(tok))
		}
	}
	s.FlushLine()
	return nil
}
