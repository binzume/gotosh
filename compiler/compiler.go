package compiler

import (
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"strings"
	"text/scanner"
)

const RET_PREFIX = "GOTOSH_RET_"

func trimQuote(s string) string {
	if len(s) >= 2 && s[0] == '\'' {
		return strings.ReplaceAll(s[1:len(s)-1], "\\'", "'")
	}
	return strings.Trim(s, "\"`") // TODO: unescape
}

func varName(s string) string {
	return strings.ReplaceAll(strings.TrimSuffix(strings.Trim(trimQuote(s), "${!} "), "[@]"), ".", "__")
}

func varValue(name string) string {
	if strings.ContainsAny(name, "#@[:]!") {
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

func (t Type) ElementType() Type {
	if strings.HasPrefix(string(t), TYPE_ARRAY) || strings.HasPrefix(string(t), TYPE_MAP) {
		p := strings.IndexRune(string(t), ']')
		if p > 0 {
			t = Type(string(t)[p+1:])
		}
	}
	return t
}

type TypedName struct {
	Name string
	Type Type
}

var specialReturnTypes = map[Type]Type{"StatusCode": "int", "TempVarString": "string", "TempVarInt": "int"}

var asValueFunc = map[string]func(*shExpression) string{
	"FLOAT_EXPR": func(e *shExpression) string { return `$(echo "` + e.expr + `" | bc -l)` },
	"INT_EXPR":   func(e *shExpression) string { return "$(( " + e.expr + " ))" },
	"STR_CMP":    func(e *shExpression) string { return "$([[ " + e.expr + " ]] && echo 1 || echo 0)" },
}

type shExpression struct {
	expr       string
	typ        string
	stdout     bool
	argTypes   []Type
	retTypes   []Type
	primaryIdx int
	lhs        []string
	declare    bool
	values     []string // for array, slice, struct
	applyFunc  func(f *shExpression, arg []string)
	applyFunc2 func(f *shExpression, arg []*shExpression)
	template   bool
	funcUsed   bool
}

func (f *shExpression) AsValue() string {
	expr := f.expr
	if fn, ok := asValueFunc[f.typ]; ok {
		expr = fn(f)
	} else if len(f.retTypes) > 0 && f.primaryIdx < 0 {
		expr = "$(" + expr + " >&2; echo \"$" + f.RetVarName(0) + "\")"
	} else if f.stdout && len(f.retTypes) > 0 && (f.retTypes[0] == "int" || strings.HasPrefix(string(f.retTypes[0]), "[]")) {
		expr = "$(" + expr + ")"
	} else if f.stdout {
		expr = "\"$(" + expr + ")\""
	}
	return expr
}

func (f *shExpression) Values() []string {
	if f.values != nil {
		return f.values
	}
	return []string{f.AsValue()}
}

func (f *shExpression) RetVarName(i int) string {
	if len(f.retTypes) > i && f.retTypes[i] == "StatusCode" {
		return "?"
	}
	return RET_PREFIX + fmt.Sprint(i)
}

func (f *shExpression) AsExec() string {
	if f.stdout {
		return f.expr + " >/dev/null"
	} else if f.typ != "" {
		return ": " + f.AsValue()
	}
	return f.expr
}

type loopInfo struct {
	level        int
	continueProc *shExpression
}

type state struct {
	scanner.Scanner
	imports      map[string]string
	funcs        map[string]shExpression
	runtimeDefs  map[string]runtimeDefinition
	vars         map[string]TypedName
	types        map[Type]Type
	cl           []string
	loopInfo     []loopInfo
	lastToken    rune
	funcName     string
	packageName  string
	w            io.Writer
	bufLine      string
	middleofline bool
	skipNextScan bool
	anonFuncID   int
}

func newState() *state {
	var s state
	s.w = os.Stdout
	s.vars = map[string]TypedName{}
	s.types = map[Type]Type{"*os.File": "int", "*exec.Cmd": "string", "bool": "int"} // Use fd as *os.File
	InitBuiltInFuncs(&s)
	return &s
}

const TYPE_PTR string = "*"
const TYPE_ARRAY string = "[]"
const TYPE_MAP string = "map["

func (s *state) IsType(t Type, prefix string) bool {
	return strings.HasPrefix(string(s.resolveType(t)), prefix)
}

func (s *state) Scan() rune {
	if s.skipNextScan {
		s.skipNextScan = false
	} else {
		s.lastToken = s.Scanner.Scan()
	}
	return s.lastToken
}

func (s *state) PeekToken() rune {
	tok := s.Scan()
	s.skipNextScan = true
	return tok
}

func (s *state) ScanIdent() string {
	s.Scan()
	return s.TokenText()
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
	for len(s.loopInfo) > 0 && s.loopInfo[len(s.loopInfo)-1].level >= len(s.cl)-1 {
		s.writeExpr(s.loopInfo[len(s.loopInfo)-1].continueProc, "")
		s.loopInfo = s.loopInfo[:len(s.loopInfo)-1]
	}
	t := s.cl[len(s.cl)-1]
	s.cl = s.cl[:len(s.cl)-1]
	s.bufLine = t + "\n" // for "else"
}

func (s *state) parseImportPkg() {
	if s.lastToken == scanner.Ident {
		name := s.TokenText()
		s.imports[name] = trimQuote(s.ScanIdent())
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

func (s *state) readFuncArgs(args []string, types []Type) ([]string, []Type) {
	for tok := s.Scan(); tok != scanner.EOF && tok != ')'; tok = s.Scan() {
		if tok == '(' || tok == ',' {
			if s.PeekToken() != ')' {
				types = append(types, s.readType(false)) // type or variable
			}
		} else {
			tt := types[len(args):]
			types = types[:len(args)]
			t := s.readType(true)
			for _, n := range tt {
				args = append(args, string(n))
				types = append(types, t)
			}
		}
	}
	for len(args) < len(types) {
		args = append(args, "_")
	}
	return args, types
}

func joinTypes(types []Type) string {
	t := ""
	for _, tt := range types {
		t += string(tt) + ", "
	}
	return strings.TrimSuffix(t, ", ")
}

func funcType(args, ret []Type) Type {
	t := "func(" + joinTypes(args) + ")"
	if len(ret) > 1 {
		t += "(" + joinTypes(ret) + ")"
	} else if len(ret) == 1 {
		t += string(ret[0])
	}
	return Type(t)
}

func (s *state) readType(scaned bool) Type {
	if scaned {
		s.skipNextScan = true
	}
	t := ""
	if tok := s.Scan(); tok == scanner.Ident {
		t = s.TokenText()
		if t == "map" {
			s.Scan() // [
			t += "[" + string(s.readType(false)) + "]"
			s.Scan() // ]
			t += string(s.readType(false))
		} else if t == "func" {
			_, argTypes := s.readFuncArgs(nil, nil)
			line := s.Position.Line
			var retTypes []Type
			if s.PeekToken() == '(' {
				_, retTypes = s.readFuncArgs(nil, nil)
			} else if s.Position.Line == line {
				if typ := s.readType(false); typ != "" {
					retTypes = []Type{typ}
				}
			}
			return funcType(argTypes, retTypes)
		} else if t == "struct" {
			tok := s.Scan() // {
			n := 0
			for ; tok != '}' && tok != scanner.EOF; tok = s.Scan() {
				if tok == ';' || tok == scanner.RawString || tok == scanner.String {
					continue
				} else if n > 0 && n%2 == 0 && tok != ',' {
					ft := s.readType(true)
					t = strings.ReplaceAll(t, ":,:", ":"+string(ft)+":") + string(ft) + ":"
				} else {
					t += s.TokenText() + ":"
				}
				n++
			}
			t += s.TokenText() // }
		} else if _, ok := s.imports[t]; ok {
			s.Scan() // .
			t += "." + s.ScanIdent()
		} else if _, ok := s.types[Type(s.packageName+"."+t)]; ok {
			t = s.packageName + "." + t
		}
	} else if tok == '*' {
		t = s.TokenText()
		t += string(s.readType(false))
	} else if tok == '[' {
		s.readExpression("int", "]", false) // ignore array size
		t += "[]" + string(s.readType(false))
	} else if tok == '.' {
		s.Scan() // ...
		s.Scan()
		t = "..." + string(s.readType(false))
	} else {
		s.skipNextScan = true
	}
	return Type(strings.TrimPrefix(t, "shell."))
}
func (s *state) setType(name string, t Type) TypedName {
	if special, ok := specialReturnTypes[t]; ok {
		t = special
	}
	s.vars[name] = TypedName{name, t}
	f := strings.Split(string(s.resolveType(Type(strings.TrimPrefix(string(t), "*")))), ":")
	for i := 1; i < len(f)-2; i += 2 {
		s.setType(name+"."+f[i], Type(f[i+1]))
	}
	return s.vars[name]
}

func (s *state) fields(t Type, name string) []TypedName {
	f := strings.Split(string(s.resolveType(t)), ":")
	if len(f) == 1 {
		return []TypedName{{name, t}}
	}
	var ret []TypedName
	for i := 1; i < len(f)-2; i += 2 {
		ret = append(ret, s.fields(Type(f[i+1]), name+"."+f[i])...)
	}
	return ret
}

func (s *state) resolveType(t Type) Type {
	for s.types[t] != "" {
		t = s.types[t]
	}
	return t
}

func (s *state) readFuncCall(name string, invoke bool) *shExpression {
	var args []*shExpression
	if v, ok := s.vars[name]; ok && s.IsType(v.Type, "func(") {
		name = "$" + name // TODO: parse retTypes
	} else if p := strings.LastIndex(name, "."); p >= 0 {
		ns := name[:p]
		if v, ok := s.vars[ns]; ok {
			name = strings.TrimPrefix(string(v.Type), "*") + "." + name[p+1:]
			var values []string
			for _, field := range s.fields(v.Type, ns) {
				values = append(values, `"$`+varName(field.Name)+`"`)
			}
			args = []*shExpression{{expr: `"` + varValue(varName(ns)) + `"`, values: values, retTypes: []Type{v.Type}}}
		} else if pkg, ok := s.imports[ns]; ok {
			name = path.Base(pkg) + "." + name[p+1:]
		}
	}
	if invoke {
		s.Scan()
		for s.lastToken != scanner.EOF && s.lastToken != ')' {
			args = append(args, s.readExpression("", ",)", false))
		}
	}

	expr := strings.ReplaceAll(name, ".", "__")
	f, ok := s.funcs[name]
	if ok {
		f.funcUsed = true
		s.funcs[name] = f
		if f.typ != "VALUE" && !invoke {
			return &shExpression{expr: expr, retTypes: []Type{funcType(f.argTypes, f.retTypes)}}
		}
		expr = f.expr
	} else {
		f.retTypes = []Type{""}
	}
	e := &shExpression{expr: expr, typ: f.typ, retTypes: f.retTypes, primaryIdx: f.primaryIdx, stdout: f.stdout}

	if f.applyFunc2 != nil {
		f.applyFunc2(e, args)
		return e
	}

	var values []string
	for _, e := range args {
		for i, t := range e.retTypes {
			if len(f.argTypes) > len(values) && s.IsType(f.argTypes[len(values)], TYPE_PTR) && !s.IsType(t, TYPE_PTR) && e.expr != "" {
				values = append(values, "\""+varName(e.expr)+"\"")
			} else if i == e.primaryIdx || i == 0 {
				values = append(values, e.Values()...)
			} else if e.primaryIdx != i {
				values = append(values, `"`+varValue(e.RetVarName(i))+`"`) // FIXME
			}
		}
	}

	if f.applyFunc != nil {
		f.applyFunc(e, values)
	} else if f.template {
		e.expr = strings.ReplaceAll(e.expr, "{0R}", RET_PREFIX+"0")
		e.expr = strings.ReplaceAll(e.expr, "{1R}", RET_PREFIX+"1")
		for i, a := range args {
			value := a.AsValue()
			e.expr = strings.ReplaceAll(e.expr, fmt.Sprintf("{%d}", i), value)
			e.expr = strings.ReplaceAll(e.expr, fmt.Sprintf("{*%d}", i), varName(value))
			if a.typ == "FLOAT_EXPR" {
				value = a.expr
			}
			e.expr = strings.ReplaceAll(e.expr, fmt.Sprintf("{%dF}", i), value)
		}
	} else if len(values) > 0 {
		e.expr = strings.TrimSpace(e.expr + " " + strings.Join(values, " "))
	}
	return e
}

func (s *state) readValues() (values []string) {
	end := ')'
	if s.Scan() == '{' {
		end = '}'
	}
	for s.lastToken != scanner.EOF && s.lastToken != end {
		values = append(values, s.readExpression("", string(end)+":", false).Values()...)
	}
	return
}

func (s *state) readExpression(typeHint Type, endToks string, allowAssign bool) *shExpression {
	expr := ""
	l := s.Line
	tokens := 0
	declare := false
	var lastExpr *shExpression
	var lastVar string
	var expressionType Type = "int"
	var lhs, lhs_candidate, values []string
	var lastTok rune
	for tok := s.Scan(); tok != scanner.EOF && (endToks != "" || strings.ContainsRune(".=*/%,:", lastTok) || s.Line == l); tok = s.Scan() {
		t := s.TokenText()
		l = s.Line
		if tok == '}' && !strings.ContainsRune(endToks, tok) {
			s.skipNextScan = true
			break
		} else if strings.ContainsRune(endToks, tok) || (!allowAssign && tok == ',') || tok == ';' {
			break
		} else if tok == '(' {
			lastExpr = s.readExpression("", ")", false)
			if expressionType != "string" && (lastExpr.typ == "INT_EXPR" || lastExpr.typ == "FLOAT_EXPR") {
				t = "(" + lastExpr.expr + ")"
			} else {
				t = lastExpr.AsValue()
			}
		} else if tok == scanner.Int {
			t = strings.Replace(strings.Replace(t, "0o", "8#", 1), "0b", "2#", 1)
		} else if tok == scanner.Float {
			expressionType = "float32"
		} else if tok == scanner.String {
			expressionType = "string"
			t = escapeShellString(t)
		} else if tok == scanner.RawString {
			expressionType = "string"
			t = "'" + strings.ReplaceAll(strings.Trim(t, "`"), "'", "\\'") + "'"
		} else if tok == scanner.Ident && t == "func" {
			lastExpr = s.procAnonFunc()
			t = lastExpr.AsValue() + " "
			expressionType = lastExpr.retTypes[0]
		} else if tok == scanner.Ident && t == "range" {
			t = "#RANGE#"
		} else if tok == '[' || tok == scanner.Ident && (t == "struct" || t == "map") { // type
			typeHint = s.readType(true)
			values = s.readValues()
			t = ""
		} else if tok == scanner.Ident || ((tok == '*' || tok == '&') && strings.ContainsRune("=+-*/([\x00", lastTok)) {
			derefPtr := tok == '*'
			refPtr := tok == '&'
			tok = scanner.Ident
			if derefPtr || refPtr {
				s.Scan()
			}
			t = s.TokenText()
			for tok := s.Scan(); tok == '.'; tok = s.Scan() {
				if s.Scan() == '.' {
					s.Scan() // ...
					s.Scan()
					break
				}
				t += "." + s.TokenText()
			}
			s.skipNextScan = true
			if s.vars[t].Type == "" && (s.vars[s.packageName+"."+t].Type != "" || s.types[Type(s.packageName+"."+t)] != "") {
				t = s.packageName + "." + t
			}
			if vt, ok := s.vars[t]; ok {
				expressionType = vt.Type
				if derefPtr {
					expressionType = Type(strings.TrimPrefix(string(expressionType), "*"))
				}
			}
			ot := t
			lt := t
			t = varName(t)
			if s.IsType(s.vars[ot].Type, TYPE_ARRAY) {
				t += "[@]"
			}
			lastVar = t
			if s.lastToken == '[' {
				s.Scan()
				var idx []*shExpression
				for s.lastToken != scanner.EOF && s.lastToken != ']' {
					idx = append(idx, s.readExpression("int", ":]", false))
				}
				if len(idx) == 1 && expressionType != "string" {
					t = ot + "[" + idx[0].AsValue() + "]:-"
					expressionType = expressionType.ElementType()
				} else if len(idx) == 1 {
					t += ":" + idx[0].AsValue() + ":1"
				} else if len(idx) >= 2 {
					t += ":" + idx[0].AsValue() + ":$(( " + idx[1].AsValue() + " - " + idx[0].AsValue() + " ))"
				}
				lt = strings.TrimSuffix(t, ":-")
			}

			if _, ok := s.types[Type(ot)]; ok {
				if tok := s.PeekToken(); tok == '{' || tok == '(' {
					values = s.readValues()
					typeHint = Type(ot)
				}
				t = ""
			} else if _, ok := s.vars[ot]; !ok || s.lastToken == '(' {
				lastExpr = s.readFuncCall(ot, s.lastToken == '(')
				t = lastExpr.AsValue()
				if len(lastExpr.retTypes) > 0 && lastExpr.retTypes[0] != "" {
					expressionType = lastExpr.retTypes[0]
				}
			} else if expressionType == "float32" || expressionType == "float64" {
				t = " " + varValue(t) + " "
			} else if expressionType == "string" || s.IsType(expressionType, TYPE_ARRAY) {
				t = "\"" + varValue(t) + "\""
			}
			if refPtr {
				t = "\"" + varName(t) + "\""
				lt = t
				expressionType = Type("*" + string(expressionType))
			}
			if allowAssign && lhs == nil {
				lhs_candidate = append(lhs_candidate, lt)
			}
		} else if strings.Contains("=!<>", t) && s.Peek() == '=' && lastTok != '<' && lastTok != '>' {
			s.Scan()
			t = " " + t + "= "
			typeHint = "bool"
		} else if tok == ':' && s.Peek() == '=' {
			declare = true
			t = ""
		} else if allowAssign && strings.Contains("+-*/%<>", t) && s.Peek() == '=' && len(lhs_candidate) > 0 {
			s.Scan()
			lhs = lhs_candidate
			if expressionType == "string" && t == "+" {
				t = ""
			} else if expressionType != "float32" && expressionType != "float64" {
				lhs = []string{}
				t += "="
			}
		} else if allowAssign && tok == '=' {
			lhs = lhs_candidate
			t = ""
			expr = ""
			tokens = -1
		} else if tok == '.' || tok == '+' && expressionType == "string" || tok == '=' && expr == "" {
			t = "" // skip
		}
		expr += t
		tokens++
		lastTok = tok
		if !s.skipNextScan {
			l = s.Line
		}
	}
	if typeHint == "" {
		typeHint = expressionType
	}
	s.skipNextScan = s.skipNextScan || s.Line != l
	e := &shExpression{expr: strings.TrimSpace(expr), retTypes: []Type{typeHint}, declare: declare, lhs: lhs, values: values}
	if lastExpr != nil && (expr == lastExpr.expr || expr == lastExpr.AsValue()) {
		lastExpr.lhs = e.lhs
		lastExpr.declare = e.declare
		return lastExpr
	} else if lastVar != "" && expr == lastVar && !s.IsType(typeHint, TYPE_MAP) {
		if s.IsType(typeHint, TYPE_PTR) {
			expr = "!" + expr // bash: !, zsh: (!)
		}
		e.expr = varValue(expr)
		if fields := s.fields(e.retTypes[0], ""); len(fields) == 0 || fields[0].Name != "" {
			e.values = []string{}
			for _, f := range fields {
				e.values = append(e.values, `"`+varValue(varName(expr+f.Name))+`"`)
			}
		}
	} else if expressionType == "string" && typeHint == "bool" {
		e.typ = "STR_CMP"
	} else if tokens > 1 && (expressionType == "float32" || expressionType == "float64") {
		e.typ = "FLOAT_EXPR"
	} else if tokens > 1 && s.resolveType(expressionType) == "int" && !s.IsType(expressionType, TYPE_ARRAY) {
		e.typ = "INT_EXPR"
	}
	return e
}

func (s *state) writeExpr(e *shExpression, typ Type) {
	statusIndex := -1
	for i, name := range e.lhs {
		if name != "_" && e.RetVarName(i) == "?" {
			statusIndex = i
		}
	}
	writeAssign := func(i int, v, vn string) {
		if typ != "" {
			s.setType(e.lhs[i], typ)
		} else if e.declare && len(e.retTypes) > i {
			s.setType(e.lhs[i], e.retTypes[i])
		}
		local := e.declare && s.funcName != ""
		for vi, field := range s.fields(s.vars[e.lhs[i]].Type, "") {
			name := varName(e.lhs[i] + field.Name)
			if s.IsType(s.vars[e.lhs[i]].Type, TYPE_MAP) && v == "" {
				s.WriteString("typeset -A ")
			} else if e.declare && s.IsType(s.vars[e.lhs[i]].Type, TYPE_PTR) ||
				len(e.retTypes) > i && s.IsType(e.retTypes[i], TYPE_PTR) ||
				s.IsType(s.vars[e.lhs[i]].Type, TYPE_MAP) {
				s.WriteString("typeset -n ") // Need to re-declare for updating pointer as well.
			} else if local {
				s.WriteString("local ")
			}
			if vn != "" && len(e.retTypes) > i {
				s.Writeln(name + "=\"$" + varName(vn+field.Name) + "\"")
			} else if local || v != "" || len(e.values) > vi {
				tv := v
				if s.IsType(field.Type, TYPE_ARRAY) || (s.IsType(field.Type, TYPE_MAP) && e.values != nil) {
					tv = "(" + strings.Join(e.Values(), " ") + ")"
				} else if len(e.values) > vi {
					tv = e.values[vi]
				} else if tv == "" && field.Type == "int" {
					tv = "0"
				}
				if local && statusIndex >= 0 {
					s.Writeln(name + "=") // to avoid 'local' modify status code
				}
				s.Writeln(name + "=" + tv)
			}
		}
	}
	if v := e.AsValue(); e.primaryIdx >= 0 && len(e.lhs) > e.primaryIdx {
		writeAssign(e.primaryIdx, v, "")
	} else if v != "" {
		s.Writeln(e.AsExec())
	}
	if statusIndex >= 0 {
		writeAssign(statusIndex, "", "?")
	}
	for i, name := range e.lhs {
		if i != e.primaryIdx && i != statusIndex && name != "_" {
			writeAssign(i, "", e.RetVarName(i))
		}
	}
}

func (s *state) procVar(names []string) {
	prefix := ""
	if s.funcName == "" && s.packageName != "main" {
		prefix = s.packageName + "."
	}
	for ; len(names) == 0 || s.lastToken == ','; s.Scan() {
		names = append(names, prefix+s.ScanIdent())
	}
	var typ = s.readType(true)
	e := &shExpression{}
	if typ == "" || s.lastToken == '=' || s.PeekToken() == '=' {
		e = s.readExpression(typ, "", false)
	}
	e.lhs = names
	e.declare = true
	s.writeExpr(e, typ)
}

func (s *state) procReturn() {
	f := s.funcs[s.funcName]
	var status *shExpression
	for i, t := range f.retTypes {
		e := s.readExpression("", "", false)
		values := e.Values()
		if i == 0 && len(e.retTypes) == len(f.retTypes) && (e.primaryIdx < 0 || e.stdout) {
			s.Writeln(e.expr + "; return $?")
			return
		} else if t == "StatusCode" {
			status = e
		} else if i == f.primaryIdx {
			s.WriteString("echo " + strings.Join(values, " ") + "; ")
		} else if fields := s.fields(t, f.RetVarName(i)); len(values) >= len(fields) {
			for vi, field := range fields {
				s.WriteString(varName(field.Name) + "=" + values[vi] + "; ")
			}
		}
		if s.lastToken != ',' {
			break
		}
	}
	if status != nil {
		s.Writeln("return " + status.AsValue())
	} else {
		s.Writeln("return")
	}
}

func (s *state) compileFunc(name, shname string, args []string, argTypes []Type) shExpression {
	previousFuncName := s.funcName
	previousVars := maps.Clone(s.vars)
	s.funcName = name
	for i, n := range args {
		argTypes[i] = Type(strings.Replace(string(argTypes[i]), "...", "[]", 1))
		s.setType(n, argTypes[i])
	}

	f := shExpression{expr: shname, primaryIdx: -1, argTypes: argTypes}
	if s.PeekToken() == '(' {
		_, f.retTypes = s.readFuncArgs(nil, nil)
	} else if typ := s.readType(false); typ != "" {
		f.retTypes = []Type{typ}
	}
	if len(f.retTypes) == 1 || len(f.retTypes) == 2 && (f.retTypes[0] == "StatusCode" || f.retTypes[1] == "StatusCode") {
		for i, t := range f.retTypes {
			if _, ok := specialReturnTypes[t]; !ok && len(s.fields(t, "")) == 1 && !s.IsType(t, TYPE_PTR) {
				f.primaryIdx = i
				f.stdout = true
			}
		}
	}
	s.Scan() // {
	s.Writeln(f.expr + "() {")
	s.cl = append(s.cl, "}")
	for i, arg := range args {
		if s.IsType(argTypes[i], TYPE_PTR) || s.IsType(argTypes[i], TYPE_MAP) {
			for _, field := range s.fields(Type(strings.TrimPrefix(string(argTypes[i]), "*")), "") {
				s.Writeln("[ \"$1\" != '" + arg + "' ] && typeset -n " + arg + varName(field.Name) + "=\"$1\"" + varName(field.Name))
			}
			s.Writeln("shift")
			continue
		}
		for _, field := range s.fields(argTypes[i], arg) {
			if !s.IsType(field.Type, TYPE_ARRAY) {
				s.Writeln("local " + varName(field.Name) + `="$1"; shift`)
			} else if field.Name != "_" {
				s.Writeln("local " + varName(field.Name) + `=("$@")`)
			}
		}
	}
	s.funcs[name] = f
	s.compile(len(s.cl) - 1)
	s.vars = previousVars
	s.funcName = previousFuncName
	return f
}

func (s *state) procFunc() {
	var args []string
	var argTypes []Type
	tok := s.PeekToken()
	name := s.TokenText()
	if tok == '(' {
		args, argTypes = s.readFuncArgs(nil, nil)
		name = s.ScanIdent()
		if len(args) > 0 {
			name = strings.TrimPrefix(string(argTypes[0]), "*") + "." + name
		}
	}
	args, argTypes = s.readFuncArgs(args, argTypes)
	shname := name
	if s.packageName != "main" {
		shname = s.packageName + "." + shname
	}
	f := s.compileFunc(name, strings.ReplaceAll(shname, ".", "__"), args, argTypes)
	s.funcs[s.packageName+"."+name] = f
	if n, found := strings.CutPrefix(name, "GOTOSH_FUNC_"); found {
		s.funcs[strings.ReplaceAll(n, "_", ".")] = f
	}
}

func (s *state) procAnonFunc() *shExpression {
	name := fmt.Sprintf("GOTOSH_ANON_%d", s.anonFuncID)
	s.anonFuncID++
	args, argTypes := s.readFuncArgs(nil, nil)
	f := s.compileFunc(name, name, args, argTypes)
	return &shExpression{expr: f.expr, retTypes: []Type{funcType(f.argTypes, f.retTypes)}}
}

func (s *state) procFor() {
	e := s.readExpression("", "{", true)
	if s.lastToken == ';' {
		s.writeExpr(e, "")
		e = s.readExpression("", "{", false)
	}

	continueExpr := &shExpression{}
	if expr := strings.TrimPrefix(e.expr, "#RANGE#"); expr != e.expr {
		var k, v = "_", "_"
		if len(e.lhs) > 0 && e.lhs[0] != "_" {
			k = e.lhs[0]
			s.writeExpr(&shExpression{lhs: []string{k}, expr: "0", declare: e.declare}, "int")
			continueExpr = &shExpression{typ: "INT_EXPR", expr: k + "+=1"}
		}
		if len(e.lhs) > 1 && e.lhs[1] != "_" {
			v = e.lhs[1]
			s.writeExpr(&shExpression{lhs: []string{v}, expr: "", declare: e.declare}, e.retTypes[0].ElementType())
		}
		if s.IsType(e.retTypes[0], TYPE_MAP) {
			s.Writeln("for " + k + ` in ` + "${!" + expr + "[@]} ; do :")
			if v != "_" {
				s.Writeln(v + `=` + "${" + expr + "[${" + k + "}]}")
			}
			continueExpr = &shExpression{}
		} else {
			s.Writeln("for " + v + ` in ` + expr + strings.Join(e.values, " ") + "; do :")
		}
	} else {
		cond := "true"
		if e.AsValue() != "" {
			cond = "[ " + e.AsValue() + " -ne 0 ]"
		}
		s.Writeln("while " + cond + "; do :")
		if s.lastToken == ';' {
			continueExpr = s.readExpression("", "{", false)
		}
	}
	s.loopInfo = append(s.loopInfo, loopInfo{len(s.cl), continueExpr})
	s.cl = append(s.cl, "done")
}

func (s *state) procIf() {
	e := s.readExpression("", "{", true)
	if s.lastToken == ';' {
		s.writeExpr(e, "")
		e = s.readExpression("bool", "{", false)
	}
	s.Writeln("if [ " + e.AsValue() + " -ne 0 ]; then :")
	s.cl = append(s.cl, "fi")
}

func (s *state) procElse() {
	s.bufLine = "" // cancel fi
	if s.Scan() == scanner.Ident && s.TokenText() == "if" {
		s.Writeln("elif [ " + s.readExpression("bool", "{", false).AsValue() + " -ne 0 ]; then :")
	} else {
		s.Writeln("else")
	}
	s.cl = append(s.cl, "fi")
}

func (s *state) compile(endDepth int) {
	for tok := s.ScanWC(); tok != scanner.EOF; tok = s.ScanWC() {
		if tok == '}' && len(s.cl) > 0 {
			s.EndBlock()
			if len(s.cl) == endDepth {
				break
			}
		} else if tok == '{' {
			s.cl = append(s.cl, "")
		} else if tok == scanner.Comment {
			for _, c := range strings.Split(strings.Trim(s.TokenText(), "/* "), "\n") {
				s.Writeln("# " + c)
			}
		} else if tok == scanner.Ident {
			t := s.TokenText()
			switch {
			case t == "package" && len(s.cl) == 0:
				s.packageName = s.ScanIdent()
			case t == "import" && len(s.cl) == 0:
				s.parseImport()
			case t == "func" && len(s.cl) == 0:
				s.procFunc()
			case t == "type":
				name := s.ScanIdent()
				s.types[Type(s.packageName+"."+name)] = s.readType(s.Scan() != '=')
			case t == "var", t == "const":
				s.procVar(nil)
			case len(s.cl) == 0:
				fmt.Fprintf(os.Stderr, "Unknown token %s: %s\n", s.Position, s.TokenText())
			case t == "for":
				s.procFor()
			case t == "if":
				s.procIf()
			case t == "else":
				s.procElse()
			case t == "break":
				s.Writeln("break")
			case t == "continue":
				if len(s.loopInfo) > 0 {
					s.writeExpr(s.loopInfo[len(s.loopInfo)-1].continueProc, "")
				}
				s.Writeln("continue")
			case t == "return":
				s.procReturn()
			case t == "go":
				s.Writeln(s.readExpression("", "", false).AsExec() + " &")
			case t == "defer":
				s.Writeln("# defer " + s.readExpression("", "", false).AsExec())
			default:
				s.skipNextScan = true
				s.writeExpr(s.readExpression("", "", true), "")
			}
		} else if tok == '*' || tok == '&' {
			s.skipNextScan = true
			s.writeExpr(s.readExpression("", "", true), "")
		} else {
			fmt.Fprintf(os.Stderr, "Unknown token %s: %s\n", s.Position, s.TokenText())
		}
	}
	s.FlushLine()
}

func (s *state) Compile(r io.Reader, srcName string) error {
	if err := s.loadRuntimeDefinitions(); err != nil {
		return err
	}
	s.Init(r)
	s.Filename = srcName
	s.imports = map[string]string{}
	s.compile(-1)
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
	if f, ok := s.funcs["main.main"]; ok {
		s.emitUsedRuntime()
		s.Writeln(f.expr + " \"${@}\"")
	}
	return nil
}
