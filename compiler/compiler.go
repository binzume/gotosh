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
	exp      string
	typ      string
	retVar   string
	stdout   bool
	retTypes []Type
}

func (f *shExpression) AsValue() string {
	exp := f.exp
	if f.typ == "INT_EXP" {
		exp = "$(( " + exp + " ))"
	} else if f.typ == "STR_CMP" {
		exp = "$([[ " + exp + " ]] && echo 1 || echo 0)"
	} else if len(f.retTypes) > 0 && f.retTypes[0] == "StatusCode" {
		// TODO: stdout...
		exp = "$(" + exp + " >&2; echo $?)"
	} else if len(f.retTypes) > 0 && f.retTypes[0] == "TempVarString" {
		exp = "$(" + exp + " >&2 && echo \"$_tmp0\")"
	} else if f.stdout && len(f.retTypes) > 0 && (f.retTypes[0] == "int" || f.retTypes[0].IsArray()) {
		exp = "$(" + exp + ")"
	} else if f.stdout {
		exp = "\"$(" + exp + ")\""
	}
	return strings.TrimSpace(exp)
}

func RetVarName(retTypes []Type, i int) string {
	if len(retTypes) > i {
		if retTypes[i] == "StatusCode" {
			return "?"
		} else if retTypes[i] == "TempVarString" || retTypes[i] == "*os.File" || i > 0 { // TODO
			return "_tmp" + fmt.Sprint(i)
		}
	}
	return ""
}

func (f *shExpression) AsExec() string {
	if f.stdout {
		return strings.TrimSpace(f.exp + " >/dev/null")
	}
	return strings.TrimSpace(f.exp)
}

type shFunc struct {
	exp      string
	stdout   bool
	retTypes []Type
	convFunc func(arg []string) string
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
	s.funcs = map[string]shFunc{
		"bash.Sleep":      {exp: "sleep"},
		"bash.Exit":       {exp: "exit"},
		"bash.Export":     {exp: "export"},
		"bash.Exec":       {retTypes: []Type{"string", "StatusCode"}, stdout: true},
		"bash.Read":       {exp: `IFS= read -r -s _tmp0`, retTypes: []Type{"TempVarString", "StatusCode"}},
		"bash.ReadLine":   {exp: `IFS= read -r -s _tmp0 <&{0}`, retTypes: []Type{"TempVarString", "StatusCode"}},
		"bash.SubStr":     {exp: "\"${{*0}:{1}:{2}}\"", retTypes: []Type{"string"}},
		"bash.Arg":        {exp: `eval echo \\${{0}}`, retTypes: []Type{"string"}, stdout: true},
		"bash.NArgs":      {exp: `$(( $# + 1 ))`, retTypes: []Type{"int"}},
		"bash.UnixTimeMs": {exp: `printf '%.0f' $( echo "${EPOCHREALTIME:-$(date +%s)} * 1000" | bc )`, retTypes: []Type{"int"}, stdout: true},
		// fmt
		"fmt.Print":   {exp: "echo -n"},
		"fmt.Println": {exp: "echo"},
		"fmt.Printf":  {exp: "printf"},
		"fmt.Sprint":  {exp: "echo -n", retTypes: []Type{"string"}, stdout: true},
		"fmt.Sprintln": {retTypes: []Type{"string"}, convFunc: func(arg []string) string {
			return "$(echo " + strings.Join(arg, " ") + ")$'\\n'"
		}},
		"fmt.Sprintf": {exp: "printf", retTypes: []Type{"string"}, stdout: true},
		// strings
		"strings.ReplaceAll": {exp: "\"${{*0}//{1}/{2}}\"", retTypes: []Type{"string"}},
		"strings.ToUpper":    {exp: "echo {0}|tr '[:lower:]' '[:upper:]'", retTypes: []Type{"string"}, stdout: true},
		"strings.ToLower":    {exp: "echo {0}|tr '[:upper:]' '[:lower:]'", retTypes: []Type{"string"}, stdout: true},
		"strings.TrimSpace":  {exp: "echo {0}| sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//'", retTypes: []Type{"string"}, stdout: true},
		"strings.TrimPrefix": {exp: "\"${{*0}#{1}}\"", retTypes: []Type{"string"}},
		"strings.TrimSuffix": {exp: "\"${{*0}%{1}}\"", retTypes: []Type{"string"}},
		"strings.Split": {retTypes: []Type{"[]string"}, stdout: true, convFunc: func(arg []string) string {
			return "IFS=" + arg[1] + " _tmp0=(" + trimQuote(arg[0]) + ") ;echo \"${_tmp0[@]}\""
		}},
		"strings.Join":     {exp: "IFS={1}; echo \"${{*0}[*]}\"", retTypes: []Type{"string"}, stdout: true},
		"strings.Contains": {exp: "case {0} in *{1}* ) echo 1;; *) echo 0;; esac", retTypes: []Type{"bool"}, stdout: true},
		"strings.IndexAny": {exp: "expr '(' index {0} {1} ')' - 1", retTypes: []Type{"int"}, stdout: true},
		// os
		"os.Stdin":    {exp: "0", retTypes: []Type{"io.Reader"}}, // variable
		"os.Stdout":   {exp: "1", retTypes: []Type{"io.Reader"}}, // variable
		"os.Exit":     {exp: "exit"},
		"os.Getwd":    {exp: "pwd", retTypes: []Type{"string", "StatusCode"}, stdout: true},
		"os.Chdir":    {exp: "cd", retTypes: []Type{"StatusCode"}, stdout: true},
		"os.Getpid":   {exp: "$$"},
		"os.Getppid":  {exp: "$PPID"},
		"os.Getuid":   {exp: "${UID:--1}"},
		"os.Geteuid":  {exp: "${EUID:-${UID:--1}}"},
		"os.Getgid":   {exp: "${GID:--1}"},
		"os.Getegid":  {exp: "${EGID:-${GID:--1}}"},
		"os.Hostname": {exp: "hostname", retTypes: []Type{"string", "StatusCode"}, stdout: true},
		"os.Getenv": {convFunc: func(arg []string) string {
			return "\"${" + trimQuote(arg[0]) + "}\""
		}},
		"os.Setenv": {convFunc: func(arg []string) string {
			return "export " + trimQuote(arg[0]) + "=" + arg[1]
		}},
		"os.Open":              {exp: `eval "exec "$(( ++GOTOSH_fd + 2 ))"<{0}" && _tmp0=$(( GOTOSH_fd + 2 ))`, retTypes: []Type{"*os.File", "StatusCode"}},
		"os.Create":            {exp: `eval "exec "$(( ++GOTOSH_fd + 2 ))">{0}" && _tmp0=$(( GOTOSH_fd + 2 ))`, retTypes: []Type{"*os.File", "StatusCode"}},
		"os.Mkdir":             {exp: "mkdir {0}", retTypes: []Type{"StatusCode"}},
		"os.MkdirAll":          {exp: "mkdir -p {0}", retTypes: []Type{"StatusCode"}},
		"os.Remove":            {exp: "rm -f", retTypes: []Type{"StatusCode"}},
		"os.RemoveAll":         {exp: "rm -rf", retTypes: []Type{"StatusCode"}},
		"os.Rename":            {exp: "mv", retTypes: []Type{"StatusCode"}},
		"exec.Command":         {exp: "echo -n ", retTypes: []Type{"*exec.Cmd"}, stdout: true}, // TODO escape command string...
		"exec.Cmd__Output":     {exp: "bash -c", retTypes: []Type{"string", "StatusCode"}, stdout: true},
		"reflect.TypeOf":       {retTypes: []Type{"_string"}, convFunc: func(arg []string) string { return `"` + string(s.vars[varName(arg[0])]) + `"` }},
		"os.File__Close":       {exp: `eval "exec {0}<&-;exec {0}>&-"`}, // TODO
		"os.File__WriteString": {exp: `echo -n {1}>&{0}`},               // TODO
		// TODO: cast
		"int":             {retTypes: []Type{"int"}},
		"byte":            {retTypes: []Type{"int"}},
		"string":          {retTypes: []Type{"string"}},
		"strconv.Atoi":    {retTypes: []Type{"int", "StatusCode"}},
		"strconv.Itoa":    {retTypes: []Type{"string"}},
		"bash.StatusCode": {retTypes: []Type{"int"}},
		// slice
		"len": {retTypes: []Type{"int"}, convFunc: func(arg []string) string { return "${#" + strings.Trim(trimQuote(arg[0]), "${}") + "}" }},
		"append": {retTypes: []Type{"_ARG1"},
			convFunc: func(arg []string) string {
				return varName(arg[0]) + "+=(" + strings.Join(arg[1:], " ") + ")"
			}},
	}
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
		t += s.TokenText()
		if t == "map" {
			s.Scan() // [
			t += s.TokenText()
			t += string(s.readType(false))
			s.Scan() // ]
			t += s.TokenText()
		} else if t == "struct" {
			s.Scan() // {
			for ; s.lastToken != '}' && s.lastToken != scanner.EOF; s.Scan() {
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
		t += s.TokenText()
		t += string(s.readType(false))
	} else if s.lastToken == '[' {
		t += s.TokenText()
		s.readExpression("int", true) // ignore array size
		t += s.TokenText()
		t += string(s.readType(false))
	}
	return Type(strings.TrimPrefix(t, "bash."))
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
		ret = append(ret, TypedName{name + "." + f[i], Type(f[i+1])})
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

func (s *state) readFuncCall(name string) *shExpression {
	var args []string
	if p := strings.LastIndex(name, "."); p >= 0 {
		ns := name[:p]
		name = name[p+1:]
		if s.vars[ns] != "" {
			name = s.vars[ns].MemberName(name)
			for _, field := range s.fields(s.vars[ns], ns) {
				args = append(args, `"`+varValue(strings.ReplaceAll(field.Name, ".", "__"))+`"`)
			}
		} else if s.imports[ns] != "" {
			name = path.Base(s.imports[ns]) + "." + name
		}
	}
	for s.lastToken != scanner.EOF && s.lastToken != ')' {
		e := s.readExpression("", true)
		for i, t := range e.retTypes {
			if i == 0 {
				for _, field := range s.fields(t, "") {
					if field.Name != "" {
						args = append(args, `"$`+strings.ReplaceAll(varName(e.AsValue())+field.Name, ".", "__")+`"`)
					} else {
						args = append(args, e.AsValue())
					}
				}
			} else if i != 0 && RetVarName(e.retTypes, i) != "" {
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
	retVar := ""
	if len(f.retTypes) > 0 && f.retTypes[0] == "_ARG1" && len(args) > 0 {
		retVar = varName(args[0])
	}
	return &shExpression{exp: exp, retTypes: f.retTypes, retVar: retVar, stdout: f.stdout}
}

func (s *state) readExpression(typeHint Type, ignoreNewLine bool) *shExpression {
	exp := ""
	l := s.Line
	s.Scan()
	if s.lastToken == '=' {
		s.Scan()
	}
	if s.lastToken == '[' {
		t := s.readType(true)
		s.Scan() // {
		for s.lastToken != scanner.EOF && s.lastToken != '}' {
			exp += " " + s.readExpression(t[2:], true).AsValue()
		}
		return &shExpression{exp: exp, retTypes: []Type{t}}
	}
	tokens := 0
	var funcRet *shExpression
	var singleVar = true
	var isString = typeHint == "string"
	for ; s.lastToken != scanner.EOF && (ignoreNewLine || s.Line == l); s.Scan() {
		tok := s.lastToken
		if tok == ')' || typeHint != "" && tok == ':' || tok == ',' || tok == ';' || tok == ']' || tok == '{' || tok == '}' {
			break
		} else if tok == '(' {
			funcRet = s.readExpression("", true)
			if !isString && funcRet.typ == "INT_EXP" {
				exp += "(" + funcRet.exp + ")"
			} else {
				exp += funcRet.AsValue()
			}
			continue
		} else if isString && tok == '+' || tok == ':' {
			continue
		}
		singleVar = singleVar && tok == scanner.Ident
		isString = isString || tok == scanner.String
		t := s.TokenText()
		if (t == "=" || t == "!") && s.Peek() == '=' {
			s.Scan()
			t = " " + t + "= "
			typeHint = "bool"
		}

		if tok == scanner.String {
			t = escapeShellString(t)
		}
		l = s.Line
		if tok == scanner.Ident {
			t = s.readName()
			if s.lastToken != '(' && s.lastToken != '[' {
				s.skipNextScan = true
			}
			isString = isString || s.vars[t] == "string"
			ot := t
			t = strings.ReplaceAll(t, ".", "__")
			if s.vars[ot].IsArray() {
				t += "[@]"
			}
			if t == "true" {
				singleVar = false
				t = "1"
			} else if t == "false" || t == "nil" {
				singleVar = false
				t = "0"
			} else if s.lastToken == '[' {
				var idx []*shExpression
				for s.lastToken != scanner.EOF && s.lastToken != ']' {
					idx = append(idx, s.readExpression("int", true))
				}
				if len(idx) == 1 {
					t = ot + "[" + idx[0].AsValue() + "]"
				} else if len(idx) >= 2 {
					t += ":" + idx[0].AsValue() + ":$(( " + idx[1].AsValue() + " - " + idx[0].AsValue() + " ))"
				}
			}
			if s.lastToken == '(' || s.funcs[ot].exp != "" {
				funcRet = s.readFuncCall(ot)
				t = funcRet.AsValue()
				singleVar = false
				if len(funcRet.retTypes) > 0 && funcRet.retTypes[0] == "string" {
					isString = true
				}
				l = s.Line
			} else if isString {
				t = "\"" + varValue(t) + "\""
			}
		}
		exp += t
		tokens++
	}
	s.skipNextScan = s.skipNextScan || s.Line != l
	if funcRet != nil && exp == funcRet.AsValue() {
		return funcRet
	}
	if tokens == 1 && !isString && singleVar {
		typeHint = s.vars[exp]
		exp = varValue(exp)
	}
	e := &shExpression{exp: exp, retTypes: []Type{"any"}}
	if isString && typeHint == "bool" {
		e.typ = "STR_CMP"
		e.retTypes = []Type{typeHint}
	} else if tokens > 1 && !isString {
		e.typ = "INT_EXP"
		e.retTypes = []Type{"int"}
	} else if typeHint != "" {
		e.retTypes = []Type{typeHint}
	} else if isString {
		e.retTypes = []Type{"string"}
	}
	return e
}

func (s *state) procAssign(names []string, decrare, readonly bool) {
	var typ Type
	if len(names) == 0 {
		s.Scan()
		name := s.TokenText()
		typ = s.readType(false)
		names = append(names, name)
	}
	e := s.readExpression(typ, false)
	v := e.AsValue()
	primaryIndex := -1
	statusIndex := -1
	for i, name := range names {
		vn := RetVarName(e.retTypes, i)

		if vn == "?" {
			statusIndex = i
		} else if e.retVar != name && vn == "" {
			primaryIndex = i
		}
		if typ != "" {
			s.vars[name] = typ
		} else if decrare || s.vars[name] == "" {
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
	if s.vars[names[0]].IsArray() {
		v = "(" + v + ")"
	}
	writeAssign := func(i int) {
		local := decrare && s.funcName != ""
		for _, field := range s.fields(s.vars[names[i]], "") {
			if field.Name != "" {
				s.vars[names[i]+field.Name] = field.Type
			}
			name := strings.ReplaceAll(names[i]+field.Name, ".", "__")
			vn := RetVarName(e.retTypes, i)
			if local && name != "_" {
				s.WriteString("local ")
				if readonly {
					s.WriteString("-r ")
				}
			}
			if i == primaryIndex && v != "" {
				if local && statusIndex >= 0 {
					s.Writeln(name) // to avoid 'local' modify status code
				}
				if field.Name != "" {
					s.Writeln(name + `="$` + varName(v) + strings.ReplaceAll(field.Name, ".", "__") + `"`)
				} else {
					s.Writeln(name + "=" + v)
				}
			} else if vn != "" && name != e.retVar && name != "_" {
				s.Writeln(name + "=\"$" + vn + field.Name + "\"")
			} else if local && name != "_" {
				s.Writeln(name)
			}
		}
	}
	if primaryIndex >= 0 {
		writeAssign(primaryIndex)
	} else {
		s.Writeln(e.AsExec())
	}
	if statusIndex >= 0 {
		writeAssign(statusIndex)
	}
	for i := range names {
		if i != primaryIndex && i != statusIndex {
			writeAssign(i)
		}
	}
}

func (s *state) procReturn() {
	var status *shExpression
	for i, t := range s.funcs[s.funcName].retTypes {
		e := s.readExpression("", false)
		if t == "StatusCode" {
			status = e
		} else if RetVarName(s.funcs[s.funcName].retTypes, i) != "" {
			s.WriteString(RetVarName(s.funcs[s.funcName].retTypes, i) + "=" + e.AsValue() + ";")
		} else {
			s.WriteString("echo " + e.AsValue() + ";")
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
	var retTypes []Type
	for s.lastToken != scanner.EOF && s.lastToken != ')' && s.lastToken != '{' {
		retTypes = append(retTypes, s.readType(s.lastToken != '(' && s.lastToken != ','))
		s.Scan() // , or ')' or '{'
	}
	for ; s.lastToken != '{' && s.lastToken != scanner.EOF; s.Scan() {
	}

	s.Writeln("function " + name + "() {")
	s.cl = append(s.cl, "}")
	for _, arg := range args {
		for _, field := range s.fields(s.vars[arg], arg) {
			if field.Type.IsArray() {
				s.Writeln("local " + strings.ReplaceAll(field.Name, ".", "__") + `=("$@")`)
			} else {
				s.Writeln("local " + strings.ReplaceAll(field.Name, ".", "__") + `="$1"; shift`)
			}
		}
	}
	f := shFunc{exp: name, retTypes: retTypes, stdout: len(retTypes) > 0 && retTypes[0] != "StatusCode" && retTypes[0] != "TempVarString"}
	s.funcs[name] = f
	if n, found := strings.CutPrefix(name, "GOTOSH_FUNC_"); found {
		s.funcs[strings.ReplaceAll(n, "_", ".")] = f
	}
}

func (s *state) procFor() {
	f := []*shExpression{{}, {}, {}}

	n := 0
	for ; s.lastToken != scanner.EOF && s.lastToken != '{' && n < 3; n++ {
		f[n] = s.readExpression("", true)
	}

	if s.useExFor {
		s.Writeln("for (( " + f[0].AsExec() + "; " + f[1].AsExec() + "; " + f[2].AsExec() + " )); do")
		s.cl = append(s.cl, "done")
		return
	}

	condIdx := 0
	if n > 1 {
		s.Writeln(f[0].AsExec())
		condIdx = 1
	}
	cond := "true"
	if f[condIdx].AsValue() != "" {
		cond = "[ " + f[condIdx].AsValue() + " -ne 0 ]"
	}
	s.Writeln("while " + cond + "; do :")
	end := "done"
	if f[2].AsExec() != "" {
		end = "let \"" + f[2].AsExec() + "\"; done" // TODO continue...
	}
	s.cl = append(s.cl, end)
}

func (s *state) procIf() {
	s.Writeln("if [ " + s.readExpression("bool", true).AsValue() + " -ne 0 ]; then :")
	s.cl = append(s.cl, "fi")
}

func (s *state) procElse() {
	s.bufLine = "" // cancel fi
	if s.Scan() == scanner.Ident && s.TokenText() == "if" {
		s.Writeln("elif [ " + s.readExpression("bool", true).AsValue() + " -ne 0 ]; then :")
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
		s.Writeln(s.readFuncCall(names[0]).AsExec())
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
				s.Writeln(s.readExpression("", false).AsExec() + " &")
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
