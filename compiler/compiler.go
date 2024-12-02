package compiler

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

func varName(s string) string {
	return strings.Trim(trimQuote(s), "${}[@]")
}

func varValue(name string) string {
	if strings.ContainsAny(name, "#@[:]") {
		return "${" + name + "}"
	}
	return "$" + name
}

func escapeShellString(s string) string {
	if strings.Contains(s, "\\") {
		return "$'" + strings.ReplaceAll(s[1:len(s)-1], "'", "\\'") + "'"
	}
	return strings.ReplaceAll(s, "$", "\\$")
}

type shExpression struct {
	exp      string
	retVar   string
	stdout   bool
	retTypes []string
}

func (f *shExpression) StdoutValue() bool {
	return f.stdout || len(f.retTypes) > 0 && (f.retTypes[0] == "StdoutString" || f.retTypes[0] == "StdoutInt")
}

func (f *shExpression) AsValue() string {
	exp := f.exp
	if len(f.retTypes) > 0 && f.retTypes[0] == "StatusCode" {
		// TODO: stdout...
		exp = "`" + exp + " >&2; echo $?`"
	} else if len(f.retTypes) > 0 && f.retTypes[0] == "TempVarString" {
		exp = "`" + exp + " >&2 && echo \"$_tmp0\"`"
	} else if len(f.retTypes) > 0 && f.retTypes[0] == "_INT_EXP" {
		exp = "$(( " + exp + " ))"
	} else if f.StdoutValue() && len(f.retTypes) > 0 && strings.HasPrefix(f.retTypes[0], "[]") {
		exp = "(`" + exp + "`)"
	} else if f.StdoutValue() && len(f.retTypes) > 0 && f.retTypes[0] == "int" {
		exp = "`" + exp + "`"
	} else if f.StdoutValue() {
		exp = "\"`" + exp + "`\""
	}
	return strings.TrimSpace(exp)
}

func RetVarName(retTypes []string, i int) string {
	if len(retTypes) > i {
		if retTypes[i] == "StatusCode" {
			return "?"
		} else if retTypes[i] == "TempVarString" || i > 0 {
			return "_tmp" + fmt.Sprint(i)
		}
	}
	return ""
}

func (f *shExpression) AsExec() string {
	if f.StdoutValue() {
		return strings.TrimSpace(f.exp + " >/dev/null")
	}
	return strings.TrimSpace(f.exp)
}

type shFunc struct {
	exp      string
	retTypes []string
	convFunc func(arg []string) string
}

type state struct {
	scanner.Scanner
	imports      map[string]string
	funcs        map[string]shFunc
	vars         map[string]string
	cl           []string
	useExFor     bool
	lastToken    rune
	funcName     string
	w            io.Writer
	bufLine      string
	middleofline bool
}

func newState() *state {
	var s state
	s.w = os.Stdout
	s.imports = map[string]string{}
	s.vars = map[string]string{}
	s.funcs = map[string]shFunc{
		"bash.Sleep":      {exp: "sleep"},
		"bash.Exit":       {exp: "exit"},
		"bash.Export":     {exp: "export"},
		"bash.Exec":       {exp: "", retTypes: []string{"StdoutString", "StatusCode"}},
		"bash.Read":       {exp: `read _tmp0`, retTypes: []string{"TempVarString", "StatusCode"}},
		"bash.SubStr":     {exp: "\"${{*0}:{1}:{2}}\"", retTypes: []string{"_"}},
		"bash.UnixTimeMs": {exp: "date +%s000", retTypes: []string{"int"}},
		// fmt
		"fmt.Print":   {exp: "echo -n"},
		"fmt.Println": {exp: "echo"},
		"fmt.Printf":  {exp: "printf"},
		"fmt.Sprint":  {exp: "echo -n", retTypes: []string{"StdoutString"}},
		"fmt.Sprintln": {exp: "echo", retTypes: []string{"_string"}, convFunc: func(arg []string) string {
			return "`echo " + strings.Join(arg, " ") + "`$'\\n'"
		}},
		"fmt.Sprintf": {exp: "printf", retTypes: []string{"StdoutString"}},
		// strings
		"strings.ReplaceAll": {exp: "\"${{*0}//{1}/{2}}\"", retTypes: []string{"_string"}},
		"strings.ToUpper":    {exp: "echo {0}|tr '[:lower:]' '[:upper:]'", retTypes: []string{"string"}},
		"strings.ToLower":    {exp: "echo {0}|tr '[:upper:]' '[:lower:]'", retTypes: []string{"string"}},
		"strings.TrimSpace":  {exp: "echo {0}| sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//'", retTypes: []string{"string"}},
		"strings.TrimPrefix": {exp: "\"${{*0}#{1}}\"", retTypes: []string{"_string"}},
		"strings.TrimSuffix": {exp: "\"${{*0}%{1}}\"", retTypes: []string{"_string"}},
		"strings.Split": {exp: "", retTypes: []string{"[]string"}, convFunc: func(arg []string) string {
			return "(`IFS=" + arg[1] + " _tmp0=(" + trimQuote(arg[0]) + ") ;echo \"${_tmp0[@]}\" `)"
		}},
		"strings.Join":     {exp: "(IFS={1}; echo \"${{*0}[*]}\")", retTypes: []string{"string"}},
		"strings.Contains": {exp: "case {0} in *{1}* ) echo 1;; *) echo 0;; esac", retTypes: []string{"bool"}},
		"strings.IndexAny": {exp: "expr '(' index {0} {1} ')' - 1", retTypes: []string{"int"}},
		// os
		"os.Exit":     {exp: "exit"},
		"os.Getwd":    {exp: "pwd", retTypes: []string{"StdoutString", "StatusCode"}},
		"os.Chdir":    {exp: "cd", retTypes: []string{"StatusCode", "StatusCode"}},
		"os.Getpid":   {exp: "$$"},
		"os.Getppid":  {exp: "$PPID"},
		"os.Getuid":   {exp: "${UID:--1}"},
		"os.Geteuid":  {exp: "${EUID:-${UID:--1}}"},
		"os.Getgid":   {exp: "${GID:--1}"},
		"os.Getegid":  {exp: "${EGID:-${GID:--1}}"},
		"os.Hostname": {exp: "hostname", retTypes: []string{"StdoutString", "StatusCode"}},
		"os.Getenv": {exp: "", convFunc: func(arg []string) string {
			return "\"${" + trimQuote(arg[0]) + "}\""
		}},
		"os.Setenv": {exp: "", convFunc: func(arg []string) string {
			return "export " + trimQuote(arg[0]) + "=" + arg[1]
		}},
		// TODO: cast
		"int":             {exp: "", retTypes: []string{"_int"}},
		"byte":            {exp: "", retTypes: []string{"_int"}},
		"string":          {exp: "", retTypes: []string{"_string"}},
		"strconv.Atoi":    {exp: "", retTypes: []string{"_int"}},
		"strconv.Itoa":    {exp: "", retTypes: []string{"_string"}},
		"bash.StatusCode": {exp: "", retTypes: []string{"_int"}},
		// slice
		"len": {exp: "", retTypes: []string{"_int"},
			convFunc: func(arg []string) string { return "${#" + s.maybeArraySuffix(varName(arg[0])) + "}" }},
		"append": {exp: "", retTypes: []string{"_ARG1"},
			convFunc: func(arg []string) string {
				return varName(arg[0]) + "+=(" + strings.Join(arg[1:], " ") + ")"
			}},
	}
	return &s
}

func (s *state) Scan() rune {
	s.lastToken = s.Scanner.Scan()
	return s.lastToken
}

func (s *state) ScanWC() rune {
	s.Mode &^= scanner.SkipComments
	s.lastToken = s.Scanner.Scan()
	s.Mode |= scanner.SkipComments
	return s.lastToken
}

func (s *state) FlushLine() {
	if s.bufLine != "" {
		t := s.bufLine
		s.bufLine = ""
		s.Indent()
		s.Writeln(t)
	}
}

func (s *state) Write(str ...any) {
	s.Indent()
	s.FlushLine()
	fmt.Fprint(s.w, str...)
	s.middleofline = true
}

func (s *state) Writeln(str ...any) {
	s.Indent()
	s.FlushLine()
	fmt.Fprintln(s.w, str...)
	s.middleofline = false
}

func (s *state) Indent() {
	if !s.middleofline {
		s.middleofline = true
		fmt.Fprint(s.w, strings.Repeat("  ", len(s.cl)))
	}
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

func (s *state) maybeArraySuffix(t string) string {
	if strings.HasPrefix(s.vars[t], "[]") {
		t += "[@]"
	}
	return t
}

func (s *state) parseImportPkg() {
	if s.lastToken == scanner.Ident {
		name := s.TokenText()
		s.Scan()
		s.imports[name] = trimQuote(s.TokenText())
	} else {
		pkg := trimQuote(s.TokenText())
		name := path.Base(pkg)
		s.imports[name] = pkg
	}
}

func (s *state) parseImport() {
	tok := s.Scan()
	if tok == '(' {
		for tok := s.Scan(); tok != scanner.EOF && tok != ')'; tok = s.Scan() {
			s.parseImportPkg()
		}
	} else {
		s.parseImportPkg()
	}
}

func (s *state) readType(scaned bool) string {
	if !scaned {
		s.Scan()
	}
	t := ""
	if s.lastToken == scanner.Ident {
		t += s.TokenText()
		if t == "map" {
			s.Scan() // [
			t += s.TokenText()
			t += s.readType(false)
			s.Scan() // ]
			t += s.TokenText()
		} else if _, ok := s.imports[t]; ok {
			s.Scan() // .
			t += s.TokenText()
			s.Scan()
			t += s.TokenText()
		}
	} else if s.lastToken == '*' {
		t += s.TokenText()
		t += s.readType(false)
	} else if s.lastToken == '[' {
		t += s.TokenText()
		s.readExpression("int") // ignore array size
		t += s.TokenText()
		t += s.readType(false)
	}
	return strings.TrimPrefix(t, "bash.")
}

func (s *state) readFuncCall(name string) *shExpression {
	ns := ""
	if s.lastToken == '.' {
		ns = name
		s.Scan()
		name = s.TokenText()
		s.Scan() // (
	}

	var args []string
	for s.lastToken != scanner.EOF && s.lastToken != ')' {
		e := s.readExpression("")
		args = append(args, e.AsValue())
		for i := range e.retTypes {
			if i != 0 && RetVarName(e.retTypes, i) != "" {
				args = append(args, `"`+varValue(RetVarName(e.retTypes, i))+`"`)
			}
		}
	}

	if ns != "" {
		if s.vars[ns] != "" {
			name = s.vars[ns] + "__" + name
			args = append([]string{`"` + varValue(ns) + `"`}, args...)
		} else {
			pkg := s.imports[ns]
			name = path.Base(pkg) + "." + name
		}
	}

	exp := name
	f, ok := s.funcs[name]
	if ok {
		exp = f.exp
	}
	if f.convFunc != nil {
		exp = f.convFunc(args)
	} else if strings.Contains(exp, "{0}") || strings.Contains(exp, "{1}") {
		for i, a := range args {
			exp = strings.ReplaceAll(exp, fmt.Sprintf("{%d}", i), a)
			exp = strings.ReplaceAll(exp, fmt.Sprintf("{*%d}", i), varName(a))
		}
	} else {
		exp += " " + strings.Join(args, " ")
	}
	retVar := ""
	if len(f.retTypes) > 0 && f.retTypes[0] == "_ARG1" && len(args) > 0 {
		retVar = varName(args[0])
	}
	return &shExpression{exp: exp, retTypes: f.retTypes, retVar: retVar, stdout: len(f.retTypes) > 0 && !strings.HasPrefix(f.retTypes[0], "_")}
}

func (s *state) readExpression(typeHint string) *shExpression {
	exp := ""
	if s.Peek() == 13 || s.Peek() == 10 {
		return &shExpression{exp: ""}
	}
	s.Scan()
	if s.lastToken == '=' {
		s.Scan()
	}
	if s.lastToken == '[' {
		t := s.readType(true)
		s.Scan() // {
		exp += "("
		for s.lastToken != scanner.EOF && s.lastToken != '}' {
			exp += " " + s.readExpression(t[2:]).AsValue()
		}
		exp += ")"
		return &shExpression{exp: exp, retTypes: []string{t}}
	}
	tokens := 0
	nest := 0
	var funcRet *shExpression
	var singleVar = true
	var isString = typeHint == "string"
	for ; s.lastToken != scanner.EOF; s.Scan() {
		tok := s.lastToken
		if nest == 0 && tok == ')' || typeHint != "" && tok == ':' || tok == ',' || tok == ';' || tok == ']' || tok == '{' || tok == '}' {
			break
		} else if tok == '(' {
			nest++
		} else if tok == ')' {
			nest--
		}
		if isString && tok == '+' || tok == ':' {
			continue
		}
		singleVar = singleVar && tok == scanner.Ident
		isString = isString || tok == scanner.String
		t := s.TokenText()
		if tok == scanner.String {
			t = escapeShellString(t)
		}
		if tok == scanner.Ident {
			if s.vars[t] == "string" {
				isString = true
			}
			ot := t
			t = s.maybeArraySuffix(t)
			if t == "true" {
				tok = scanner.Int
				t = "1"
			} else if t == "false" || t == "nil" {
				tok = scanner.Int
				t = "0"
			} else if s.Peek() == '[' {
				s.Scan()
				var idx []*shExpression
				for s.lastToken != scanner.EOF && s.lastToken != ']' {
					idx = append(idx, s.readExpression("int"))
				}
				if len(idx) == 1 {
					t = ot + "[" + idx[0].AsValue() + "]"
				} else if len(idx) >= 2 {
					t += ":" + idx[0].AsValue() + ":$(( " + idx[1].AsValue() + " - " + idx[0].AsValue() + " ))"
				}
			}
		}
		if tok == scanner.Ident && (s.Peek() == '(' || s.Peek() == '.') {
			s.Scan()
			funcRet = s.readFuncCall(t)
			exp += funcRet.AsValue()
			singleVar = false
			if len(funcRet.retTypes) > 0 && funcRet.retTypes[0] == "string" {
				isString = true
			}
		} else if tok == scanner.Ident && isString {
			exp += "\"" + varValue(t) + "\""
		} else {
			exp += t
		}
		if exp == "[]" {
			s.readType(false)
		}
		tokens++
		if nest == 0 && s.Peek() == 13 || s.Peek() == 10 { // TODO
			break
		}
	}
	if funcRet != nil && exp == funcRet.AsValue() {
		return funcRet
	}
	if tokens == 1 && !isString && singleVar {
		exp = "\"" + varValue(exp) + "\""
	}
	retTypes := []string{}
	if tokens > 1 && !isString {
		retTypes = append(retTypes, "_INT_EXP")
	} else if typeHint != "" {
		retTypes = append(retTypes, typeHint)
	} else if isString {
		retTypes = append(retTypes, "string")
	}
	return &shExpression{exp: exp, retTypes: retTypes}
}

func (s *state) procAssign(names []string, local bool, readonly bool) {
	if len(names) == 0 {
		s.Scan()
		name := s.TokenText()
		s.vars[name] = s.readType(false)
		names = append(names, name)
	}
	e := s.readExpression(s.vars[names[0]])
	v := e.AsValue()
	stdoutIndex := -1
	statusIndex := -1
	for i, name := range names {
		vn := RetVarName(e.retTypes, i)
		if e.retVar != name && vn == "" {
			stdoutIndex = i
		}
		if vn == "?" {
			statusIndex = i
		}
		if s.vars[name] == "" {
			if len(e.retTypes) > i && e.retTypes[i] != "" {
				s.vars[name] = e.retTypes[i]
			} else {
				s.vars[name] = "" // TODO
			}
		}
	}
	if v == "" && strings.HasPrefix(s.vars[names[0]], "[]") {
		v = "()"
	}
	writeAssign := func(i int) {
		if local {
			s.Write("local ")
			if readonly {
				s.Write("-r ")
			}
		}
		vn := RetVarName(e.retTypes, i)
		if i == stdoutIndex && v != "" {
			if local && statusIndex >= 0 {
				s.Writeln(names[i]) // to avoid local modify status code
			}
			s.Writeln(names[i] + "=" + v)
		} else if vn != "" && names[i] != e.retVar {
			s.Writeln(names[i] + "=\"$" + vn + "\"")
		} else if local {
			s.Writeln(names[i])
		}
	}
	if stdoutIndex >= 0 {
		writeAssign(stdoutIndex)
	} else {
		s.Writeln(e.AsExec())
	}
	if statusIndex >= 0 {
		writeAssign(statusIndex)
	}
	for i := range names {
		if i != stdoutIndex && i != statusIndex {
			writeAssign(i)
		}
	}
}

func (s *state) procReturn() {
	var status *shExpression
	for i, t := range s.funcs[s.funcName].retTypes {
		e := s.readExpression("")
		if t == "StatusCode" {
			status = e
		} else if RetVarName(s.funcs[s.funcName].retTypes, i) != "" {
			s.Write(RetVarName(s.funcs[s.funcName].retTypes, i) + "=" + e.AsValue() + ";")
		} else {
			s.Write("echo " + e.AsValue() + ";")
		}
		if s.lastToken != ',' {
			break
		}
	}
	if status != nil {
		s.Writeln("return ", status.AsValue())
	} else {
		s.Writeln("return")
	}
}

func (s *state) procFunc() {
	var args []string
	var argTypes []string
	tok := s.Scan()
	prefix := ""
	if tok == '(' {
		s.Scan()
		args = append(args, s.TokenText())
		t := strings.ReplaceAll(s.readType(false), "*", "") // pointer is not supported
		s.vars[args[len(argTypes)]] = t
		argTypes = append(argTypes, t)
		s.Scan()
		prefix = t + "__"
		s.Scan()
	}
	name := prefix + s.TokenText()
	s.funcName = name
	for tok = s.Scan(); tok != scanner.EOF && tok != ')'; tok = s.Scan() {
		if tok == '(' || tok == ',' {
			tok = s.Scan()
			if tok == ')' {
				break
			}
			args = append(args, s.TokenText())
		} else {
			t := s.readType(true)
			for len(args) > len(argTypes) {
				s.vars[args[len(argTypes)]] = t
				argTypes = append(argTypes, t)
			}
		}
	}
	s.Scan() // '(' or '{' or Ident
	var retTypes []string
	for s.lastToken != scanner.EOF && s.lastToken != ')' && s.lastToken != '{' {
		retTypes = append(retTypes, s.readType(s.lastToken != '(' && s.lastToken != ','))
		s.Scan() // , or ')' or '{'
	}
	for ; s.lastToken != '{' && s.lastToken != scanner.EOF; s.Scan() {
	}

	s.Writeln("function " + name + "() {")
	for _, arg := range args {
		if strings.HasPrefix(s.vars[arg], "[]") {
			s.Writeln("local " + arg + `=("$@")`)
		} else {
			s.Writeln("local " + arg + `="$1"; shift`)
		}
	}
	s.cl = append(s.cl, "}")
	s.funcs[name] = shFunc{exp: name, retTypes: retTypes}
}

func (s *state) procFor() {
	f := []string{"", "", ""}

	n := 0
	for s.lastToken != scanner.EOF && n < 3 {
		f[n] = s.readExpression("").AsExec()
		n++
		if s.lastToken == '{' {
			break
		}
	}

	if s.useExFor {
		s.Writeln("for (( " + f[0] + "; " + f[1] + "; " + f[2] + " )); do")
		s.cl = append(s.cl, "done")
		return
	}

	condIdx := 0
	if n > 1 {
		s.Writeln(f[0])
		condIdx = 1
	}
	cond := "true"
	if n >= condIdx && f[condIdx] != "" {
		cond = "[ $(( " + f[condIdx] + " )) -ne 0 ]"
	}
	s.Writeln("while " + cond + "; do")
	end := "done"
	if n > 2 {
		end = "let \"" + f[2] + "\"; done" // TODO continue...
	}
	s.cl = append(s.cl, end)
}

func (s *state) procIf() {
	s.Writeln("if [ " + s.readExpression("bool").AsValue() + " -ne 0 ]; then")
	s.cl = append(s.cl, "fi")
}

func (s *state) procElse() {
	s.bufLine = "" // cancel fi
	tok := s.Scan()
	if tok == scanner.Ident && s.TokenText() == "if" {
		s.Writeln("elif [ " + s.readExpression("bool").AsValue() + " -ne 0 ]; then")
	} else {
		s.Writeln("else")
	}
	s.cl = append(s.cl, "fi")
}

func (s *state) procSentense(t string) {
	tok := s.Scan()
	names := []string{t}
	for tok == ',' {
		s.Scan()
		names = append(names, s.TokenText())
		tok = s.Scan()
	}
	if tok == ':' {
		s.Scan() // =
		s.procAssign(names, s.funcName != "", false)
	} else if tok == '=' {
		s.procAssign(names, false, false)
	} else if tok == '(' || tok == '.' {
		s.Writeln(s.readFuncCall(t).AsExec())
	} else {
		fmt.Printf("# Unknown token %s: %s %s\n", s.Position, s.TokenText(), scanner.TokenString(tok))
	}
}

func (s *state) Compile(r io.Reader, srcName string) error {
	s.Init(r)
	s.Filename = srcName

	for tok := s.ScanWC(); tok != scanner.EOF; tok = s.ScanWC() {
		if tok == '}' && len(s.cl) > 0 {
			s.EndBlock()
		} else if tok == '{' {
			s.cl = append(s.cl, "")
		} else if tok == scanner.Comment {
			for _, c := range strings.Split(strings.Trim(s.TokenText(), "/* "), "\n") {
				s.Writeln("# " + c)
			}
		} else if tok == scanner.Ident {
			t := s.TokenText()
			switch t {
			case "package":
				s.Scan()
			case "import":
				s.parseImport()
			case "type":
				s.Writeln("## type ", s.readExpression("").AsExec()) // TODO
			case "for":
				s.procFor()
			case "if":
				s.procIf()
			case "else":
				s.procElse()
			case "break":
				s.Writeln("break")
			case "continue":
				s.Writeln("continue")
			case "return":
				s.procReturn()
			case "func":
				s.procFunc()
			case "var":
				s.procAssign(nil, s.funcName != "", false)
			case "const":
				s.procAssign(nil, s.funcName != "", true)
			case "go":
				s.Writeln(s.readExpression("").AsExec() + " &")
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

func CompileFiles(sources []string) error {
	s := newState()
	s.Writeln("#!/bin/sh")
	s.Writeln("")

	for _, srcPath := range sources {
		r, err := os.Open(srcPath)
		if err != nil {
			return err
		}
		defer r.Close()
		if err := s.Compile(r, srcPath); err != nil {
			return err
		}
	}
	if _, ok := s.funcs["main"]; ok {
		s.Writeln("main")
	}
	return nil
}
