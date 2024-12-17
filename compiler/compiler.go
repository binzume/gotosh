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
	if len(s) >= 2 && s[0] == '\'' {
		return strings.ReplaceAll(s[1:len(s)-2], "\\'", "'")
	}
	return strings.Trim(s, "\"`") // TODO: unescape
}

func varName(s string) string {
	return strings.ReplaceAll(strings.Trim(trimQuote(s), "${}[@]"), ".", "__")
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

type Type string

func (t Type) IsArray() bool {
	return strings.HasPrefix(string(t), "[]")
}

func (t Type) MemberName(name string) string {
	return strings.TrimPrefix(string(t), "*") + "__" + name
}

type TypedName struct {
	Name string
	Type Type
}

type shExpression struct {
	exp        string
	typ        string
	retVar     string
	stdout     bool
	retTypes   []Type
	primaryIdx int
	values     []string // for array, slice, struct
}

func (f *shExpression) AsValue() string {
	exp := strings.TrimSpace(f.exp)
	if f.typ == "FLOAT_EXP" {
		exp = `$(echo "scale=10;` + exp + `" | bc)`
	} else if f.typ == "INT_EXP" {
		exp = "$(( " + exp + " ))"
	} else if f.typ == "STR_CMP" {
		exp = "$([[ " + exp + " ]] && echo 1 || echo 0)"
	} else if len(f.retTypes) > 0 && f.primaryIdx < 0 && f.retTypes[0] == "StatusCode" {
		// TODO: stdout...
		exp = "$(" + exp + " >&2; echo $?)"
	} else if len(f.retTypes) > 0 && f.primaryIdx < 0 {
		exp = "$(" + exp + " >&2 && echo \"$_tmp0\")"
	} else if f.stdout && len(f.retTypes) > 0 && (f.retTypes[0] == "int" || f.retTypes[0].IsArray()) {
		exp = "$(" + exp + ")"
	} else if f.stdout {
		exp = "\"$(" + exp + ")\""
	}
	return exp
}

func RetVarName(retTypes []Type, i int) string {
	if len(retTypes) > i && retTypes[i] == "StatusCode" {
		return "?"
	}
	return "_tmp" + fmt.Sprint(i)
}

func (f *shExpression) AsExec() string {
	if f.stdout && f.exp != "" {
		return strings.TrimSpace(f.exp + " >/dev/null")
	}
	return strings.TrimSpace(f.exp)
}

type shFunc struct {
	exp        string
	stdout     bool
	retTypes   []Type
	primaryIdx int
	convFunc   func(arg []string) string
}

type state struct {
	scanner.Scanner
	imports      map[string]string
	funcs        map[string]shFunc
	vars         map[string]Type
	types        map[Type]Type
	cl           []string
	useExFor     bool
	lastToken    rune
	funcName     string
	w            io.Writer
	bufLine      string
	middleofline bool
	skipNextScan bool
}

func newState() *state {
	var s state
	s.w = os.Stdout
	s.imports = map[string]string{}
	s.vars = map[string]Type{}
	s.types = map[Type]Type{}
	InitBuiltInFuncs(&s)
	return &s
}

func (s *state) Scan() rune {
	if s.skipNextScan {
		s.skipNextScan = false
	} else {
		s.lastToken = s.Scanner.Scan()
	}
	return s.lastToken
}

func (s *state) ScanWC() rune {
	s.Mode &^= scanner.SkipComments
	s.Scan()
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

func (s *state) WriteString(str string) {
	s.FlushLine()
	s.Indent()
	fmt.Fprint(s.w, str)
	s.middleofline = true
}

func (s *state) Writeln(str ...any) {
	s.FlushLine()
	s.Indent()
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

func (s *state) parseImportPkg() {
	if s.lastToken == scanner.Ident {
		name := s.TokenText()
		s.Scan()
		s.imports[name] = trimQuote(s.TokenText())
	} else {
		pkg := trimQuote(s.TokenText())
		s.imports[path.Base(pkg)] = pkg
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

func (s *state) readType(scaned bool) Type {
	if !scaned {
		s.Scan()
	}
	t := ""
	if s.lastToken == scanner.Ident {
		t = s.TokenText()
		if t == "map" {
			s.Scan() // [
			t += s.TokenText()
			t += string(s.readType(false))
			s.Scan() // ]
			t += s.TokenText()
		} else if t == "struct" {
			s.Scan() // {
			for ; s.lastToken != '}' && s.lastToken != scanner.EOF; s.Scan() {
				if s.lastToken == scanner.RawString || s.lastToken == scanner.String {
					continue // ignore tag
				}
				t += s.TokenText() + ":" // TODO
			}
			t += s.TokenText()
		} else if _, ok := s.imports[t]; ok {
			s.Scan() // .
			t += s.TokenText()
			s.Scan()
			t += s.TokenText()
		}
	} else if s.lastToken == '*' {
		t = s.TokenText()
		t += string(s.readType(false))
	} else if s.lastToken == '[' {
		t = s.TokenText()
		s.readExpression("int", ']') // ignore array size
		t += s.TokenText()
		t += string(s.readType(false))
	}
	return Type(strings.TrimPrefix(t, "shell."))
}

func (s *state) fields(t Type, name string) []TypedName {
	for s.types[t] != "" {
		t = s.types[t]
	}
	f := strings.Split(string(t), ":")
	if len(f) < 4 {
		return []TypedName{{name, t}}
	}
	var ret []TypedName
	for i := 1; i < len(f)-2; i += 2 {
		ret = append(ret, s.fields(Type(f[i+1]), name+"."+f[i])...)
	}
	return ret
}

func (s *state) readName() string {
	name := s.TokenText()
	for tok := s.Scan(); tok == '.'; tok = s.Scan() {
		s.Scan()
		name += "." + s.TokenText()
	}
	return name
}

func (s *state) readFuncCall(name string, variable bool) *shExpression {
	var args []string
	if p := strings.LastIndex(name, "."); p >= 0 {
		ns := name[:p]
		name = name[p+1:]
		if s.vars[ns] != "" {
			name = s.vars[ns].MemberName(name)
			for _, field := range s.fields(s.vars[ns], ns) {
				args = append(args, `"`+varValue(varName(field.Name))+`"`)
			}
		} else if s.imports[ns] != "" {
			name = path.Base(s.imports[ns]) + "." + name
		}
	}
	for !variable && s.lastToken != scanner.EOF && s.lastToken != ')' {
		e := s.readExpression("", ',')
		for i, t := range e.retTypes {
			if i == e.primaryIdx && len(e.values) > 0 {
				args = append(args, e.values...)
			} else if i == 0 {
				if fields := s.fields(t, ""); fields[0].Name != "" {
					for _, field := range fields {
						args = append(args, `"$`+varName(e.AsValue()+field.Name)+`"`)
					}
				} else {
					args = append(args, e.AsValue())
				}
			} else if e.primaryIdx != i {
				args = append(args, `"`+varValue(RetVarName(e.retTypes, i))+`"`)
			}
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
	e := &shExpression{exp: exp, retTypes: f.retTypes, primaryIdx: f.primaryIdx, stdout: f.stdout}
	if len(f.retTypes) > 0 && f.retTypes[0] == "_ARG1" && len(args) > 0 {
		e.retVar = varName(args[0])
	}
	return e
}

func (s *state) readExpression(typeHint Type, endTok rune) *shExpression {
	exp := ""
	l := s.Line
	s.Scan()
	if s.lastToken == '=' {
		s.Scan()
	}
	if s.lastToken == '[' || s.lastToken == scanner.Ident && s.vars[s.TokenText()] == "" && s.types[Type(s.TokenText())] != "" {
		e := &shExpression{retTypes: []Type{s.readType(true)}}
		s.Scan() // {
		for s.lastToken != scanner.EOF && s.lastToken != '}' {
			elm := s.readExpression("", ',')
			e.values = append(e.values, elm.values...)
			if len(elm.values) == 0 {
				e.values = append(e.values, elm.AsValue())
			}
		}
		s.readExpression(typeHint, endTok) // scan endTok
		return e
	}
	tokens := 0
	var funcRet *shExpression
	var lastVar string
	var expressionType Type = typeHint
	for ; s.lastToken != scanner.EOF && (endTok != 0 || s.Line == l); s.Scan() {
		tok := s.lastToken
		if tok == ')' || tok == endTok || tok == ',' || tok == ';' || tok == ']' || tok == '}' {
			break
		} else if tok == '(' {
			funcRet = s.readExpression("", ')')
			if expressionType != "string" && (funcRet.typ == "INT_EXP" || funcRet.typ == "FLOAT_EXP") {
				exp += "(" + funcRet.exp + ")"
			} else {
				exp += funcRet.AsValue()
			}
			continue
		}
		l = s.Line
		t := s.TokenText()

		if tok == scanner.Float {
			expressionType = "float32"
		} else if tok == scanner.String {
			expressionType = "string"
			t = escapeShellString(t)
		} else if tok == scanner.RawString {
			expressionType = "string"
			t = "'" + strings.ReplaceAll(strings.Trim(t, "`"), "'", "\\'") + "'"
		} else if tok == scanner.Ident {
			t = s.readName()
			if s.lastToken != '(' && s.lastToken != '[' {
				s.skipNextScan = true
			}
			if s.vars[t] != "" {
				expressionType = s.vars[t]
			}
			ot := t
			t = varName(t)
			if s.vars[ot].IsArray() {
				t += "[@]"
			}
			if s.lastToken == '[' {
				var idx []*shExpression
				for s.lastToken != scanner.EOF && s.lastToken != ']' {
					idx = append(idx, s.readExpression("int", ':'))
				}
				if len(idx) == 1 {
					t = ot + "[" + idx[0].AsValue() + "]"
				} else if len(idx) >= 2 {
					t += ":" + idx[0].AsValue() + ":$(( " + idx[1].AsValue() + " - " + idx[0].AsValue() + " ))"
				}
			}
			lastVar = t
			if t == "true" {
				t = "1"
			} else if t == "false" || t == "nil" {
				t = "0"
			}
			if s.lastToken == '(' || s.funcs[ot].exp != "" {
				funcRet = s.readFuncCall(ot, s.lastToken != '(')
				t = funcRet.AsValue()
				if len(funcRet.retTypes) > 0 && funcRet.retTypes[0] != "" {
					expressionType = funcRet.retTypes[0]
				}
			} else if expressionType == "string" {
				t = "\"" + varValue(t) + "\""
			} else if expressionType == "float32" {
				t = varValue(t)
			}
		} else if strings.Contains("=!<>", t) && s.Peek() == '=' {
			s.Scan()
			t = " " + t + "= "
			typeHint = "bool"
		} else if expressionType == "string" && tok == '+' {
			t = ""
		}
		exp += t
		tokens++
	}
	if typeHint == "" {
		typeHint = expressionType
	}
	s.skipNextScan = s.skipNextScan || (endTok == 0 && s.Line != l)
	e := &shExpression{exp: exp, retTypes: []Type{typeHint}}
	if funcRet != nil && exp == funcRet.AsValue() {
		return funcRet
	} else if lastVar != "" && exp == lastVar {
		e.retTypes = []Type{s.vars[strings.ReplaceAll(strings.TrimSuffix(exp, "[@]"), "__", ".")]} // TODO
		e.exp = varValue(exp)
		if fields := s.fields(e.retTypes[0], exp); len(fields) > 1 {
			for _, f := range fields {
				e.values = append(e.values, `"`+varValue(varName(f.Name))+`"`)
			}
		}
	} else if expressionType == "string" && typeHint == "bool" {
		e.typ = "STR_CMP"
	} else if tokens > 1 && expressionType == "float32" {
		e.typ = "FLOAT_EXP"
	} else if tokens > 1 && expressionType != "string" {
		e.typ = "INT_EXP"
	}
	return e
}

func (s *state) procAssign(names []string, declare, readonly bool) {
	var typ Type
	if len(names) == 0 { // var or const
		for ; len(names) == 0 || s.lastToken == ','; s.Scan() {
			s.Scan()
			names = append(names, s.TokenText())
		}
		s.skipNextScan = true
		typ = s.readType(false)
	}
	e := s.readExpression(typ, 0)
	statusIndex := -1
	for i, name := range names {
		if name != "_" && RetVarName(e.retTypes, i) == "?" {
			statusIndex = i
		}
		if typ != "" {
			s.vars[name] = typ
		} else if declare || s.vars[name] == "" {
			if len(e.retTypes) > i && e.retTypes[i] != "" {
				s.vars[name] = e.retTypes[i]
			} else {
				s.vars[name] = "any"
			}
		}
		if s.vars[name] == "TempVarString" { // TODO
			s.vars[name] = "string"
		} else if s.vars[name] == "StatusCode" {
			s.vars[name] = "int"
		}
	}
	writeAssign := func(i int, v, vn string) {
		local := declare && s.funcName != ""
		for vi, field := range s.fields(s.vars[names[i]], "") {
			if field.Name != "" {
				s.vars[names[i]+field.Name] = field.Type
			}
			name := varName(names[i] + field.Name)
			if local {
				s.WriteString("local ")
				if readonly {
					s.WriteString("-r ")
				}
			}
			if vn != "" && len(e.retTypes) > i {
				s.Writeln(name + "=\"$" + varName(vn+field.Name) + "\"")
			} else if local || v != "" || len(e.values) > vi {
				if local && statusIndex >= 0 {
					s.Writeln(name) // to avoid 'local' modify status code
				}
				tv := v
				if field.Type.IsArray() {
					tv = "(" + strings.Join(e.values, " ") + v + ")"
				} else if len(e.values) > vi {
					tv = e.values[vi]
				} else if tv == "" && field.Type == "int" {
					tv = "0"
				}
				s.Writeln(name + "=" + tv)
			}
		}
	}
	if v := e.AsValue(); e.primaryIdx >= 0 {
		writeAssign(e.primaryIdx, v, "")
	} else if v != "" {
		s.Writeln(e.AsExec())
	}
	if statusIndex >= 0 {
		writeAssign(statusIndex, "", "?")
	}
	for i, name := range names {
		if i != e.primaryIdx && i != statusIndex && name != "_" && name != e.retVar {
			writeAssign(i, "", RetVarName(e.retTypes, i))
		}
	}
}

func (s *state) procReturn() {
	var status *shExpression
	for i, t := range s.funcs[s.funcName].retTypes {
		e := s.readExpression("", 0)

		if t == "StatusCode" {
			status = e
		} else if fields := s.fields(t, ""); len(fields) > 1 {
			for vi, field := range fields {
				if len(e.values) > vi {
					s.WriteString(varName("_tmp"+fmt.Sprint(i)+field.Name) + "=" + e.values[vi] + "; ")
				}
			}
		} else if i == s.funcs[s.funcName].primaryIdx {
			s.WriteString("echo " + e.AsValue() + "; ")
		} else {
			s.WriteString(RetVarName(s.funcs[s.funcName].retTypes, i) + "=" + e.AsValue() + "; ")
		}
		if s.lastToken != ',' {
			break
		}
	}
	if status != nil {
		s.Writeln("return", status.AsValue())
	} else {
		s.Writeln("return")
	}
}

func (s *state) procFunc() {
	var args []string
	var argTypes []Type
	tok := s.Scan()
	name := s.TokenText()
	if tok == '(' {
		s.Scan()
		args = append(args, s.TokenText())
		t := s.readType(false)
		s.vars[args[len(argTypes)]] = Type(t)
		argTypes = append(argTypes, Type(t))
		s.Scan() // .
		s.Scan() // name
		name = t.MemberName(s.TokenText())
	}
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
	f := shFunc{exp: name, primaryIdx: -1}
	statusIndex := -1
	stdoutIndex := -1
	for s.lastToken != scanner.EOF && s.lastToken != ')' && s.lastToken != '{' {
		t := s.readType(s.lastToken != '(' && s.lastToken != ',')
		if t == "StatusCode" {
			statusIndex = len(f.retTypes)
		} else if t != "TempVarString" && len(s.fields(t, "")) == 1 {
			stdoutIndex = len(f.retTypes)
		}
		f.retTypes = append(f.retTypes, t)
		s.Scan() // , or ')' or '{'
	}
	for ; s.lastToken != '{' && s.lastToken != scanner.EOF; s.Scan() {
	}
	if stdoutIndex >= 0 && (len(f.retTypes) == 1 || len(f.retTypes) == 2 && statusIndex >= 0) {
		f.primaryIdx = stdoutIndex
		f.stdout = true
	}

	s.Writeln("function " + name + "() {")
	s.cl = append(s.cl, "}")
	for _, arg := range args {
		for _, field := range s.fields(s.vars[arg], arg) {
			if field.Type.IsArray() {
				s.Writeln("local " + varName(field.Name) + `=("$@")`)
			} else {
				s.Writeln("local " + varName(field.Name) + `="$1"; shift`)
			}
		}
	}
	s.funcs[name] = f
	if n, found := strings.CutPrefix(name, "GOTOSH_FUNC_"); found {
		s.funcs[strings.ReplaceAll(n, "_", ".")] = f
	}
}

func (s *state) procFor() {
	f := []*shExpression{{}, {}, {}}

	n := 0
	for ; s.lastToken != scanner.EOF && s.lastToken != '{' && n < 3; n++ {
		f[n] = s.readExpression("", '{')
	}

	if s.useExFor {
		s.Writeln("for (( " + f[0].AsExec() + "; " + f[1].AsExec() + "; " + f[2].AsExec() + " )); do")
		s.cl = append(s.cl, "done")
		return
	}

	condIdx := 0
	if n > 1 {
		if init := strings.Split(f[0].AsExec(), ":="); len(init) == 1 {
			s.Writeln(init[0])
		} else {
			s.Writeln("local " + init[0] + "=" + init[1])
		}
		condIdx = 1
	}
	cond := "true"
	if f[condIdx].AsValue() != "" {
		cond = "[ " + f[condIdx].AsValue() + " -ne 0 ]"
	}
	s.Writeln("while " + cond + "; do :")
	end := "done"
	if f[2].AsExec() != "" {
		end = ": $(( " + f[2].AsExec() + " )); done" // TODO continue...
	}
	s.cl = append(s.cl, end)
}

func (s *state) procIf() {
	s.Writeln("if [ " + s.readExpression("bool", '{').AsValue() + " -ne 0 ]; then :")
	s.cl = append(s.cl, "fi")
}

func (s *state) procElse() {
	s.bufLine = "" // cancel fi
	if s.Scan() == scanner.Ident && s.TokenText() == "if" {
		s.Writeln("elif [ " + s.readExpression("bool", '{').AsValue() + " -ne 0 ]; then :")
	} else {
		s.Writeln("else")
	}
	s.cl = append(s.cl, "fi")
}

func (s *state) procSentense() {
	names := []string{s.readName()}
	for s.lastToken == ',' {
		s.Scan()
		names = append(names, s.readName())
	}
	tok := s.lastToken
	if tok == ':' {
		s.Scan() // =
		s.procAssign(names, true, false)
	} else if tok == '=' {
		s.procAssign(names, false, false)
	} else if tok == '(' {
		s.Writeln(s.readFuncCall(names[0], false).AsExec())

	} else if strings.ContainsRune("+-*/%&|^", tok) {
		op := s.TokenText()
		tok = s.Scan()
		op += s.TokenText()
		if tok == '=' {
			op += s.readExpression("int", 0).AsValue()
		}
		s.Writeln(": $((" + varName(names[0]) + op + "))")
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
				s.Scan()
				name := s.TokenText()
				s.Scan()
				s.types[Type(name)] = s.readType(s.lastToken != '=')
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
				s.procAssign(nil, true, false)
			case "const":
				s.procAssign(nil, true, true)
			case "go":
				s.Writeln(s.readExpression("", 0).AsExec() + " &")
			default:
				s.procSentense()
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
	s.Writeln("#!/bin/bash")
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
		s.Writeln("main \"${@}\"")
	}
	return nil
}
