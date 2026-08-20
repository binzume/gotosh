package compiler

import (
	"fmt"
	"strconv"
	"strings"
)

// TODO: export types to modify from outside
var InitBuiltInFuncs = func(s *state) {
	s.funcs = map[string]shExpression{
		"nil":   {expr: "0", typ: "VALUE", retTypes: []Type{""}},
		"true":  {expr: "1", typ: "VALUE", retTypes: []Type{"bool"}},
		"false": {expr: "0", typ: "VALUE", retTypes: []Type{"bool"}},
		"shell.Bind": {retTypes: []Type{"StageCall"}, primaryIdx: 0, applyFunc: func(e *shExpression, arg []string) {
			if len(arg) == 0 {
				e.expr = ""
				return
			}
			e.expr = strings.Join(append([]string{arg[0], "0", "1"}, arg[1:]...), " ")
		}},
		"shell.ExecPipe": {retTypes: []Type{"StatusCode"}, primaryIdx: -1, applyFunc: func(e *shExpression, arg []string) {
			commands := make([]string, 0, len(arg))
			for _, command := range arg {
				if command != "" {
					commands = append(commands, command)
				}
			}
			e.expr = strings.Join(commands, " | ")
		}},
		"shell.Sleep":         {expr: "sleep"},
		"shell.Exit":          {expr: "exit"},
		"shell.Export":        {expr: "export"},
		"shell.Exec":          {retTypes: []Type{"string", "StatusCode"}, stdout: true},
		"shell.Read":          {expr: `IFS= read -r -s {0R}`, retTypes: []Type{"string", "StatusCode"}, primaryIdx: -1, template: true},
		"shell.ReadLine":      {expr: `IFS= read -r -s {0R} <&{0}`, retTypes: []Type{"string", "StatusCode"}, primaryIdx: -1, template: true},
		"shell.SubStr":        {expr: "\"${{*0}:{1}:{2}}\"", retTypes: []Type{"string"}, template: true},
		"shell.Arg":           {expr: `eval echo \${{0}}`, retTypes: []Type{"string"}, stdout: true, template: true},
		"shell.Args":          {expr: `"$@"`, retTypes: []Type{"[]string"}},
		"shell.SetArgs":       {expr: `set -- `},
		"shell.NArgs":         {expr: `$(( $# + 1 ))`, retTypes: []Type{"int"}},
		"shell.UnixTimeMs":    {expr: `printf '%.0f' $( echo "${EPOCHREALTIME:-$(date +%s)} * 1000" | bc )`, retTypes: []Type{"int"}, stdout: true},
		"shell.Do":            {retTypes: []Type{"StatusCode"}, applyFunc: func(e *shExpression, arg []string) { e.expr = strings.TrimSpace(trimQuote(arg[0])) }, primaryIdx: -1},
		"shell.IsShellScript": {expr: "1", typ: "VALUE", retTypes: []Type{"bool"}},

		"shell.SetFloatPrecision": {applyFunc: func(e *shExpression, arg []string) {
			if p, err := strconv.Atoi(arg[0]); err == nil && p >= 0 {
				asValueFunc["FLOAT_EXPR"] = func(e *shExpression) string {
					return `$(echo "scale=` + strconv.Itoa(p) + `;` + e.expr + `" | BC_LINE_LENGTH=` + strconv.Itoa(p+10) + ` bc -l)`
				}
			} else {
				asValueFunc["FLOAT_EXPR"] = func(e *shExpression) string { return `$(echo "` + e.expr + `" | bc -l)` }
			}
		}, retTypes: []Type{"struct{:}"}, primaryIdx: -1},
		"shell.Files": {applyFunc: func(e *shExpression, arg []string) { e.expr = trimQuote(arg[0]) }, retTypes: []Type{"[]string"}},
		// fmt
		"fmt.Print":   {expr: "echo -n"},
		"fmt.Println": {expr: "echo"},
		"fmt.Printf":  {expr: "printf"},
		"fmt.Sprint":  {expr: "echo -n", retTypes: []Type{"string"}, stdout: true},
		"fmt.Sprintln": {retTypes: []Type{"string"}, applyFunc: func(e *shExpression, arg []string) {
			e.expr = "$(echo " + strings.Join(arg, " ") + ")$'\\n'"
		}},
		"fmt.Sprintf":  {expr: "printf", retTypes: []Type{"string"}, stdout: true},
		"fmt.Fprint":   {applyFunc: func(e *shExpression, arg []string) { e.expr = "echo -n " + strings.Join(arg[1:], " ") + " >&" + arg[0] }},
		"fmt.Fprintln": {applyFunc: func(e *shExpression, arg []string) { e.expr = "echo " + strings.Join(arg[1:], " ") + " >&" + arg[0] }},
		"fmt.Fprintf":  {applyFunc: func(e *shExpression, arg []string) { e.expr = "printf " + strings.Join(arg[1:], " ") + " >&" + arg[0] }},
		// strings
		"strings.Split": {retTypes: []Type{"[]string"}, stdout: true, applyFunc: func(e *shExpression, arg []string) {
			e.expr = "IFS=" + arg[1] + " _tmp0=(" + trimQuote(arg[0]) + ") ;echo \"${_tmp0[@]}\""
		}},
		"strings.Join": {expr: "IFS={1} {0R}=\"${{*0}[*]}\"", retTypes: []Type{"string"}, primaryIdx: -1, template: true},
		// os
		"os.Stdin":    {expr: "0", typ: "VALUE", retTypes: []Type{"*os.File"}},
		"os.Stdout":   {expr: "1", typ: "VALUE", retTypes: []Type{"*os.File"}},
		"os.Stderr":   {expr: "1", typ: "VALUE", retTypes: []Type{"*os.File"}},
		"os.Args":     {expr: `"$0" "$@"`, typ: "VALUE", retTypes: []Type{"[]string"}},
		"os.Exit":     {expr: "exit"},
		"os.Getwd":    {expr: "pwd", retTypes: []Type{"string", "StatusCode"}, stdout: true},
		"os.Chdir":    {expr: "cd", retTypes: []Type{"StatusCode"}, stdout: true},
		"os.Getpid":   {expr: "$$", retTypes: []Type{"int"}},
		"os.Getppid":  {expr: "$PPID", retTypes: []Type{"int"}},
		"os.Getuid":   {expr: "${UID:--1}", retTypes: []Type{"int"}},
		"os.Geteuid":  {expr: "${EUID:-${UID:--1}}", retTypes: []Type{"int"}},
		"os.Getgid":   {expr: "${GID:--1}", retTypes: []Type{"int"}},
		"os.Getegid":  {expr: "${EGID:-${GID:--1}}", retTypes: []Type{"int"}},
		"os.Hostname": {expr: "uname -n", retTypes: []Type{"string", "StatusCode"}, stdout: true},
		"os.Getenv": {applyFunc: func(e *shExpression, arg []string) {
			e.expr = "\"${" + trimQuote(arg[0]) + "}\""
		}, retTypes: []Type{"string"}},
		"os.Setenv": {applyFunc: func(e *shExpression, arg []string) {
			e.expr = "export " + trimQuote(arg[0]) + "=" + arg[1]
		}},
		"os.Pipe": {expr: `_tmp=$(mktemp -d) && mkfifo $_tmp/f && {0R}=$(( GOTOSH_fd=${GOTOSH_fd:-2}+1 )) && {1R}=$(( ++GOTOSH_fd ))` +
			` && eval "exec ${1R}<>\"$_tmp/f\" ${0R}<\"$_tmp/f\"" && rm -rf $_tmp`,
			retTypes: []Type{"*os.File", "*os.File", "StatusCode"}, primaryIdx: -1, template: true},
		"os.Open":             {expr: `{0R}=$(( GOTOSH_fd=${GOTOSH_fd:-2}+1 )); eval "exec ${0R}<'{0}'"`, retTypes: []Type{"*os.File", "StatusCode"}, primaryIdx: -1, template: true},
		"os.Create":           {expr: `{0R}=$(( GOTOSH_fd=${GOTOSH_fd:-2}+1 )); eval "exec ${0R}>'{0}'"`, retTypes: []Type{"*os.File", "StatusCode"}, primaryIdx: -1, template: true},
		"os.Stat":             {expr: `[ -e {0} ] && echo {0}`, retTypes: []Type{"fs.FileInfo", "StatusCode"}, stdout: true, template: true},
		"fs.FileInfo.Name":    {expr: `basename {0}`, retTypes: []Type{"string"}, stdout: true, template: true},
		"fs.FileInfo.Size":    {expr: `stat -c %s {0} 2>/dev/null || stat -f %z {0}`, retTypes: []Type{"int"}, stdout: true, template: true},
		"fs.FileInfo.Mode":    {typ: "INT_EXPR", expr: `8#$(stat -c %a {0} 2>/dev/null || stat -f %p {0})`, retTypes: []Type{"int"}, template: true},
		"fs.FileInfo.IsDir":   {expr: `[ -d {0} ] && echo 1 || echo 0`, retTypes: []Type{"bool"}, stdout: true, template: true},
		"os.Mkdir":            {expr: "mkdir {0}", retTypes: []Type{"StatusCode"}, template: true},
		"os.MkdirAll":         {expr: "mkdir -p {0}", retTypes: []Type{"StatusCode"}, template: true},
		"os.Remove":           {expr: "rm -f", retTypes: []Type{"StatusCode"}},
		"os.RemoveAll":        {expr: "rm -rf", retTypes: []Type{"StatusCode"}},
		"os.Rename":           {expr: "mv", retTypes: []Type{"StatusCode"}},
		"os.File.WriteString": {expr: `echo -n {1} >&{0}`, template: true},
		"os.File.Close":       {expr: `eval "exec {0}<&- {0}>&-"`, template: true},
		"os.File.Fd":          {expr: `{0}`, retTypes: []Type{"int"}, template: true},
		"exec.Command":        {expr: "echo -n ", retTypes: []Type{"*exec.Cmd"}, stdout: true}, // TODO escape command string...
		"exec.Cmd.Output":     {expr: "bash -c", retTypes: []Type{"string", "StatusCode"}, stdout: true},
		"reflect.TypeOf":      {retTypes: []Type{"string"}, applyFunc: func(e *shExpression, arg []string) { e.expr = `"` + string(s.vars[varName(arg[0])].Type) + `"` }},
		"runtime.Compiler":    {expr: "'gotosh'", typ: "VALUE", retTypes: []Type{"string"}},               // constant
		"runtime.GOARCH":      {expr: "uname -m", typ: "VALUE", retTypes: []Type{"string"}, stdout: true}, // constant
		"runtime.GOOS":        {expr: "uname -o", typ: "VALUE", retTypes: []Type{"string"}, stdout: true}, // constant
		// math (using bc)
		"math.Pi":   {expr: "3.141592653589793", typ: "VALUE", retTypes: []Type{"float64"}}, // constant
		"math.E":    {expr: "2.718281828459045", typ: "VALUE", retTypes: []Type{"float64"}}, // constant
		"math.Sqrt": {typ: "FLOAT_EXPR", expr: "sqrt({0F})", retTypes: []Type{"float64"}, template: true},
		"math.Pow":  {typ: "FLOAT_EXPR", expr: "e(l({0F})*{1F})", retTypes: []Type{"float64"}, template: true},
		"math.Exp":  {typ: "FLOAT_EXPR", expr: "e({0F})", retTypes: []Type{"float64"}, template: true},
		"math.Log":  {typ: "FLOAT_EXPR", expr: "l({0F})", retTypes: []Type{"float64"}, template: true},
		"math.Sin":  {typ: "FLOAT_EXPR", expr: "s({0F})", retTypes: []Type{"float64"}, template: true},
		"math.Cos":  {typ: "FLOAT_EXPR", expr: "c({0F})", retTypes: []Type{"float64"}, template: true},
		"math.Tan":  {typ: "FLOAT_EXPR", expr: "x={0F}; s(x)/c(x)", retTypes: []Type{"float64"}, template: true},
		"math.Atan": {typ: "FLOAT_EXPR", expr: "a({0F})", retTypes: []Type{"float64"}, template: true},
		"math.Sinh": {typ: "FLOAT_EXPR", expr: "x={0F}; ((e(x)-e(-x))/2)", retTypes: []Type{"float64"}, template: true},
		"math.Cosh": {typ: "FLOAT_EXPR", expr: "x={0F}; ((e(x)+e(-x))/2)", retTypes: []Type{"float64"}, template: true},
		"math.Tanh": {typ: "FLOAT_EXPR", expr: "x={0F}; ((e(x)-e(-x))/(e(x)+e(-x)))", retTypes: []Type{"float64"}, template: true},
		// TODO: cast
		"int":              {expr: "printf '%.0f' {0}", retTypes: []Type{"int"}, stdout: true, template: true},
		"byte":             {retTypes: []Type{"int"}},
		"float32":          {retTypes: []Type{"float32"}},
		"float64":          {retTypes: []Type{"float64"}},
		"string":           {retTypes: []Type{"string"}},
		"strconv.Atoi":     {retTypes: []Type{"int", "StatusCode"}},
		"strconv.Itoa":     {retTypes: []Type{"string"}},
		"shell.StatusCode": {retTypes: []Type{"int"}},
		// slice
		"len": {retTypes: []Type{"int"}, applyFunc2: func(e *shExpression, args []*shExpression) {
			if len(args) > 0 && len(args[0].retTypes) > 0 {
				if args[0].expr == "" && s.IsType(args[0].retTypes[0], TYPE_MAP) {
					e.expr = fmt.Sprint(len(args[0].values) / 2)
				} else if args[0].expr == "" && s.IsType(args[0].retTypes[0], TYPE_ARRAY) {
					e.expr = fmt.Sprint(len(args[0].values))
				} else if s.IsType(args[0].retTypes[0], TYPE_MAP) {
					e.expr = "${#" + varName(args[0].expr) + "[@]}"
				} else {
					e.expr = "${#" + strings.Trim(trimQuote(args[0].expr), "${}@") + "}"
				}
			}
		}},
		"append": {retTypes: []Type{"[]any"}},
	}
}
