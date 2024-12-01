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

type shFunc struct {
	shName   string
	retTypes []string
	convFunc func(arg []string) string
}

type shExpression struct {
	cmd      string
	retVar   string
	stdout   bool
	retTypes []string
}

func (f *shExpression) StdoutValue() bool {
	return f.stdout || len(f.retTypes) > 0 && (f.retTypes[0] == "StdoutString" || f.retTypes[0] == "StdoutInt")
}

func (f *shExpression) AsValue() string {
	cmd := f.cmd
	if len(f.retTypes) > 0 && f.retTypes[0] == "StatusCode" {
		// TODO: stdout...
		cmd = "`" + cmd + " >&2; echo $?`"
	} else if len(f.retTypes) > 0 && f.retTypes[0] == "TempVarString" {
		cmd = "`" + cmd + " >&2 && echo \"$_tmp0\"`"
	} else if len(f.retTypes) > 0 && f.retTypes[0] == "_INT_EXP" {
		cmd = "$(( " + cmd + " ))"
	} else if f.StdoutValue() && len(f.retTypes) > 0 && strings.HasPrefix(f.retTypes[0], "[]") {
		cmd = "(`" + cmd + "`)"
	} else if f.StdoutValue() {
		cmd = "\"`" + cmd + "`\""
	}
	return strings.TrimSpace(cmd)
}

func (f *shExpression) VarName(i int) string {
	if len(f.retTypes) > i {
		if f.retTypes[i] == "StatusCode" {
			return "?"
		} else if f.retTypes[i] == "TempVarString" {
			return "_tmp" + fmt.Sprint(i)
		}
	}
	return ""
}

func (f *shExpression) AsExec() string {
	if f.StdoutValue() {
		return strings.TrimSpace(f.cmd + " >/dev/null")
	}
	return strings.TrimSpace(f.cmd)
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

func (s *state) Scan() rune {
	s.lastToken = s.Scanner.Scan()
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
	if !s.middleofline {
		s.Indent()
	}
	s.FlushLine()
	fmt.Fprint(s.w, str...)
	s.middleofline = true
}

func (s *state) Writeln(str ...any) {
	if !s.middleofline {
		s.Indent()
	}
	s.FlushLine()
	fmt.Fprintln(s.w, str...)
	s.middleofline = false
}

func (s *state) Indent() {
	s.middleofline = true
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
		s.Scan() // ]
		t += s.TokenText()
		t += s.readType(false)
	}
	return strings.TrimPrefix(t, "bash.")
}

func (s *state) readFuncCall(name string) *shExpression {
	pkgref := ""
	if s.lastToken == '.' {
		pkgref = name
		s.Scan()
		name = s.TokenText()
		s.Scan() // (
	}
	if pkgref != "" {
		pkg := s.imports[pkgref]
		name = path.Base(pkg) + "." + name
	}
	f, ok := s.funcs[name]
	if ok {
		name = f.shName
	}

	var args []string
	for s.lastToken != scanner.EOF && s.lastToken != ')' {
		e := s.readExpression("")
		args = append(args, e.AsValue())
		for i := range e.retTypes {
			if i != 0 && e.VarName(i) != "" {
				args = append(args, `"`+varValue(e.VarName(i))+`"`)
			}
		}
	}

	cmd := name + " " + strings.Join(args, " ")
	if f.convFunc != nil {
		cmd = f.convFunc(args)
	}
	retVar := ""
	if len(f.retTypes) > 0 && f.retTypes[0] == "_ARG1" && len(args) > 0 {
		retVar = varName(args[0])
	}
	return &shExpression{cmd: cmd, retTypes: f.retTypes, retVar: retVar, stdout: len(f.retTypes) > 0 && !strings.HasPrefix(f.retTypes[0], "_")}
}

func (s *state) readExpression(typeHint string) *shExpression {
	exp := ""
	if s.Peek() == 13 || s.Peek() == 10 {
		return &shExpression{cmd: ""}
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
		return &shExpression{cmd: exp, retTypes: []string{t}}
	}
	tokens := 0
	nest := 0
	var funcRet *shExpression
	var mode rune = scanner.Int
	if typeHint == "string" {
		mode = scanner.String
	}
	for ; s.lastToken != scanner.EOF; s.Scan() {
		tok := s.lastToken
		if nest == 0 && tok == ')' || typeHint != "" && tok == ':' || tok == ',' || tok == ';' || tok == ']' || tok == '{' || tok == '}' {
			break
		} else if tok == '(' {
			nest++
		} else if tok == ')' {
			nest--
		}
		if mode == scanner.String && tok == '+' || tok == ':' {
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
			if s.vars[t] == "string" {
				mode = scanner.String
			}
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
					t += "[" + idx[0].AsValue() + "]"
				} else if len(idx) >= 2 {
					if strings.HasPrefix(s.vars[t], "[]") {
						t += "[@]"
					}
					t += ":" + idx[0].AsValue() + ":$(( " + idx[1].AsValue() + " - " + idx[0].AsValue() + " ))"
				}
			} else if strings.HasPrefix(s.vars[t], "[]") {
				t += "[@]"
			}
		}
		if tok == scanner.Ident && (s.Peek() == '(' || s.Peek() == '.') {
			mode = scanner.String
			s.Scan()
			funcRet = s.readFuncCall(t)
			exp += funcRet.AsValue()
		} else if tok == scanner.Ident && mode == scanner.String {
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
	if tokens == 1 && mode == scanner.Ident {
		exp = "\"" + varValue(exp) + "\""
	}
	if funcRet != nil && exp == funcRet.AsValue() {
		return funcRet
	}
	retTypes := []string{}
	if tokens > 1 && mode != scanner.String {
		retTypes = append(retTypes, "_INT_EXP")
	} else if typeHint != "" {
		retTypes = append(retTypes, typeHint)
	} else if mode == scanner.String {
		retTypes = append(retTypes, "string")
	}
	return &shExpression{cmd: exp, retTypes: retTypes}
}

func (s *state) procAssign(names []string, local bool, readonly bool) {
	if len(names) == 0 {
		s.Scan()
		name := s.TokenText()
		s.vars[name] = s.readType(false)
		names = append(names, name)
	}
	e := s.readExpression("")
	v := e.AsValue()
	stdoutIndex := -1
	for i, name := range names {
		if e.retVar != name && e.VarName(i) == "" {
			stdoutIndex = i
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
	if stdoutIndex < 0 {
		s.Writeln(e.AsExec())
	} else {
		if local {
			s.Write("local ")
			if readonly {
				s.Write("-r ")
			}
		}
		if v != "" {
			s.Writeln(names[stdoutIndex] + "=" + v)
		} else if local {
			s.Writeln(names[stdoutIndex])
		}
	}
	for ii := range names {
		i := len(names) - ii - 1 // TODO priority on $?
		if i != stdoutIndex {
			if local {
				s.Write("local ")
				if readonly {
					s.Write("-r ")
				}
			}
			vn := e.VarName(i)
			if vn != "" && names[i] != e.retVar {
				s.Writeln(names[i] + "=\"$" + vn + "\"")
			} else if local {
				s.Writeln(names[i])
			}
		}

	}
}

func (s *state) procReturn() {
	var status *shExpression
	for i, t := range s.funcs[s.funcName].retTypes {
		e := s.readExpression("")
		if t == "StatusCode" {
			status = e
		} else if t == "TempVarString" {
			s.Write("_tmp" + fmt.Sprint(i) + "=" + e.AsValue() + ";")
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
	name := ""
	var args []string
	var argTypes []string
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
			continue
		}
		if tok == '(' || tok == ',' {
			tok = s.Scan()
			if tok == scanner.Ident {
				args = append(args, s.TokenText())
			} else if tok == ')' {
				break
			}
		} else if (tok == scanner.Ident || tok == '[' || tok == '*') && len(args) > len(argTypes) {
			t := s.readType(true)
			for len(args) > len(argTypes) {
				s.vars[args[len(argTypes)]] = t
				argTypes = append(argTypes, t)
			}
		}
	}
	for _, arg := range args {
		if strings.HasPrefix(s.vars[arg], "[]") {
			s.Writeln("local " + arg + `=("$@")`)
		} else {
			s.Writeln("local " + arg + `="$1"; shift`)
		}
	}
	s.Scan() // '(' or '{' or Ident
	var retTypes []string
	for s.lastToken != scanner.EOF && s.lastToken != ')' && s.lastToken != '{' {
		retTypes = append(retTypes, s.readType(s.lastToken == scanner.Ident))
		s.Scan() // , or ')' or '{'
	}
	for ; s.lastToken != '{' && s.lastToken != scanner.EOF; s.Scan() {
	}
	s.funcs[name] = shFunc{shName: name, retTypes: retTypes}
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
	condition := s.readExpression("bool").AsValue()
	s.Writeln("if [ " + condition + " -ne 0 ]; then")
	s.cl = append(s.cl, "fi")
}

func (s *state) procElse() {
	s.bufLine = "" // cancel fi
	condition := ""
	for tok := s.Scan(); tok != scanner.EOF && tok != '{'; tok = s.Scan() {
		if tok == scanner.Ident && s.TokenText() == "if" {
			condition = s.readExpression("bool").AsValue()
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
	} else if tok == '(' || s.imports[t] != "" && tok == '.' {
		s.Writeln(s.readFuncCall(t).AsExec())
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
	s.imports = map[string]string{}
	s.vars = map[string]string{}
	s.funcs = map[string]shFunc{
		"bash.Sleep":  {shName: "sleep"},
		"bash.Exit":   {shName: "exit"},
		"bash.Export": {shName: "export"},
		"bash.Exec": {shName: "", retTypes: []string{"StdoutString", "StatusCode"},
			convFunc: func(arg []string) string { return trimQuote(arg[0]) }},
		"bash.Read": {shName: `read _tmp0`, retTypes: []string{"TempVarString", "StatusCode"}},
		"bash.SubStr": {shName: "", retTypes: []string{"_"},
			convFunc: func(arg []string) string {
				return "\"${" + varName(arg[0]) + ":" + arg[1] + ":" + arg[2] + "}\""
			}},
		"bash.UnixTimeMs": {shName: "date +%s000", retTypes: []string{"int"}},
		// fmt
		"fmt.Print":   {shName: "echo -n"},
		"fmt.Println": {shName: "echo"},
		"fmt.Printf":  {shName: "printf"},
		"fmt.Sprint":  {shName: "echo -n", retTypes: []string{"StdoutString"}},
		"fmt.Sprintln": {shName: "echo", retTypes: []string{"_string"}, convFunc: func(arg []string) string {
			return "`echo " + strings.Join(arg, " ") + "`$'\\n'"
		}},
		"fmt.Sprintf": {shName: "printf", retTypes: []string{"StdoutString"}},
		// strings
		"strings.ReplaceAll": {shName: "", retTypes: []string{"_string"}, convFunc: func(arg []string) string {
			return "\"${" + varName(arg[0]) + "//" + arg[1] + "/" + arg[2] + "}\""
		}},
		"strings.ToUpper": {shName: "", retTypes: []string{"string"}, convFunc: func(arg []string) string {
			return "echo " + arg[0] + "|tr '[:lower:]' '[:upper:]'"
		}},
		"strings.ToLower": {shName: "", retTypes: []string{"string"}, convFunc: func(arg []string) string {
			return "echo " + arg[0] + "|tr '[:upper:]' '[:lower:]'"
		}},
		"strings.TrimSpace": {shName: "", retTypes: []string{"string"}, convFunc: func(arg []string) string {
			return "echo " + arg[0] + "| sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//'"
		}},
		"strings.TrimPrefix": {shName: "", retTypes: []string{"_string"}, convFunc: func(arg []string) string {
			return "\"${" + varName(arg[0]) + "#" + arg[1] + "}\""
		}},
		"strings.TrimSuffix": {shName: "", retTypes: []string{"_string"}, convFunc: func(arg []string) string {
			return "\"${" + varName(arg[0]) + "%" + arg[1] + "}\""
		}},
		"strings.Split": {shName: "", retTypes: []string{"[]string"}, convFunc: func(arg []string) string {
			return "(`IFS=" + arg[1] + " _tmp0=(" + trimQuote(arg[0]) + ") ;echo \"${_tmp0[@]}\" `)"
		}},
		"strings.Contains": {shName: "", retTypes: []string{"bool"}, convFunc: func(arg []string) string {
			return "case " + arg[0] + " in *" + arg[1] + "* ) echo 1;; *) echo 0;; esac"
		}},
		"strings.IndexAny": {shName: "", retTypes: []string{"int"}, convFunc: func(arg []string) string {
			return "expr '(' index " + arg[0] + " " + arg[1] + " ')' - 1"
		}},
		// os
		"os.Exit":     {shName: "exit"},
		"os.Getwd":    {shName: "pwd", retTypes: []string{"StdoutString", "StatusCode"}},
		"os.Chdir":    {shName: "cd", retTypes: []string{"StatusCode", "StatusCode"}},
		"os.Getpid":   {shName: "$$"},
		"os.Getppid":  {shName: "$PPID"},
		"os.Getuid":   {shName: "$UID"},
		"os.Geteuid":  {shName: "${EUID:-$UID}"},
		"os.Getgid":   {shName: "$GID"},
		"os.Getegid":  {shName: "${EGID:-$GID}"},
		"os.Hostname": {shName: "hostname", retTypes: []string{"StdoutString", "StatusCode"}},
		"os.Getenv": {shName: "", convFunc: func(arg []string) string {
			return "\"${" + trimQuote(arg[0]) + "}\""
		}},
		"os.Setenv": {shName: "", convFunc: func(arg []string) string {
			return "export " + trimQuote(arg[0]) + "=" + arg[1]
		}},
		// TODO: cast
		"int":             {shName: "", retTypes: []string{"_int"}},
		"byte":            {shName: "", retTypes: []string{"_int"}},
		"string":          {shName: "", retTypes: []string{"_string"}},
		"strconv.Atoi":    {shName: "", retTypes: []string{"_int"}},
		"strconv.Itoa":    {shName: "", retTypes: []string{"_string"}},
		"bash.StatusCode": {shName: "", retTypes: []string{"_int"}},
		// slice
		"len": {shName: "", retTypes: []string{"_int"},
			convFunc: func(arg []string) string {
				if strings.HasPrefix(s.vars[varName(arg[0])], "[]") {
					return "${#" + varName(arg[0]) + "[@]}"
				}
				return "${#" + varName(arg[0]) + "}"
			}},
		"append": {shName: "", retTypes: []string{"_ARG1"},
			convFunc: func(arg []string) string {
				return varName(arg[0]) + "+=(" + strings.Join(arg[1:], " ") + ")"
			}},
	}

	s.Writeln("#!/bin/sh")
	s.Writeln("")

	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
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
