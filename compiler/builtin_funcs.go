package compiler

import (
	"strconv"
	"strings"
)

// TODO: export types to modify from outside
var InitBuiltInFuncs = func(s *state) {
	s.funcs = map[string]shExpression{
		"nil":                 {expr: "0", retTypes: []Type{""}},
		"true":                {expr: "1", retTypes: []Type{"bool"}},
		"false":               {expr: "0", retTypes: []Type{"bool"}},
		"shell.Sleep":         {expr: "sleep"},
		"shell.Exit":          {expr: "exit"},
		"shell.Export":        {expr: "export"},
		"shell.Exec":          {retTypes: []Type{"string", "StatusCode"}, stdout: true},
		"shell.Read":          {expr: `IFS= read -r -s _tmp0`, retTypes: []Type{"string", "StatusCode"}, primaryIdx: -1},
		"shell.ReadLine":      {expr: `IFS= read -r -s _tmp0 <&{0}`, retTypes: []Type{"string", "StatusCode"}, primaryIdx: -1},
		"shell.SubStr":        {expr: "\"${{*0}:{1}:{2}}\"", retTypes: []Type{"string"}},
		"shell.Arg":           {expr: `eval echo \${{0}}`, retTypes: []Type{"string"}, stdout: true},
		"shell.Args":          {expr: `"$@"`, retTypes: []Type{"[]string"}},
		"shell.SetArgs":       {expr: `set -- `},
		"shell.NArgs":         {expr: `$(( $# + 1 ))`, retTypes: []Type{"int"}},
		"shell.UnixTimeMs":    {expr: `printf '%.0f' $( echo "${EPOCHREALTIME:-$(date +%s)} * 1000" | bc )`, retTypes: []Type{"int"}, stdout: true},
		"shell.Do":            {retTypes: []Type{"StatusCode"}, applyFunc: func(e *shExpression, arg []string) { e.expr = trimQuote(arg[0]) }, primaryIdx: -1},
		"shell.IsShellScript": {expr: "1", retTypes: []Type{"bool"}},

		"shell.SetFloatPrecision": {applyFunc: func(e *shExpression, arg []string) {
			if p, err := strconv.Atoi(arg[0]); err == nil && p >= 0 {
				asValueFunc["FLOAT_EXPR"] = func(e *shExpression) string {
					return `$(echo "scale=` + strconv.Itoa(p) + `;` + e.expr + `" | BC_LINE_LENGTH=` + strconv.Itoa(p+10) + ` bc -l)`
				}
			} else {
				asValueFunc["FLOAT_EXPR"] = func(e *shExpression) string { return `$(echo "` + e.expr + `" | bc -l)` }
			}
		}, retTypes: []Type{"struct{:}"}, primaryIdx: -1},
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
		"strings.ReplaceAll": {expr: "\"${{*0}//{1}/{2}}\"", retTypes: []Type{"string"}},
		"strings.ToUpper":    {expr: "echo {0}|tr '[:lower:]' '[:upper:]'", retTypes: []Type{"string"}, stdout: true},
		"strings.ToLower":    {expr: "echo {0}|tr '[:upper:]' '[:lower:]'", retTypes: []Type{"string"}, stdout: true},
		"strings.TrimSpace":  {expr: "echo {0}| sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//'", retTypes: []Type{"string"}, stdout: true},
		"strings.TrimPrefix": {expr: "\"${{*0}#{1}}\"", retTypes: []Type{"string"}},
		"strings.TrimSuffix": {expr: "\"${{*0}%{1}}\"", retTypes: []Type{"string"}},
		"strings.Split": {retTypes: []Type{"[]string"}, stdout: true, applyFunc: func(e *shExpression, arg []string) {
			e.expr = "IFS=" + arg[1] + " _tmp0=(" + trimQuote(arg[0]) + ") ;echo \"${_tmp0[@]}\""
		}},
		"strings.Join":     {expr: "IFS={1}; echo \"${{*0}[*]}\"", retTypes: []Type{"string"}, stdout: true},
		"strings.Contains": {expr: "case {0} in (*{1}*) echo 1;; (*) echo 0;; esac", retTypes: []Type{"bool"}, stdout: true},
		"strings.IndexAny": {expr: "expr '(' index {0} {1} ')' - 1", retTypes: []Type{"int"}, stdout: true},
		// os
		"os.Stdin":    {expr: "0", retTypes: []Type{"*os.File"}},         // variable
		"os.Stdout":   {expr: "1", retTypes: []Type{"*os.File"}},         // variable
		"os.Stderr":   {expr: "1", retTypes: []Type{"*os.File"}},         // variable
		"os.Args":     {expr: `"$0" "$@"`, retTypes: []Type{"[]string"}}, // variable
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
		"os.Pipe": {expr: `_tmp=$(mktemp -d) && mkfifo $_tmp/f && _tmp0=$(( GOTOSH_fd=${GOTOSH_fd:-2}+1 )) && _tmp1=$(( ++GOTOSH_fd ))` +
			` && eval "exec $_tmp1<>\"$_tmp/f\" $_tmp0<\"$_tmp/f\"" && rm -rf $_tmp`,
			retTypes: []Type{"*os.File", "*os.File", "StatusCode"}, primaryIdx: -1},
		"os.Open":              {expr: `_tmp0=$(( GOTOSH_fd=${GOTOSH_fd:-2}+1 )); eval "exec $_tmp0<'{0}'"`, retTypes: []Type{"*os.File", "StatusCode"}, primaryIdx: -1},
		"os.Create":            {expr: `_tmp0=$(( GOTOSH_fd=${GOTOSH_fd:-2}+1 )); eval "exec $_tmp0>'{0}'"`, retTypes: []Type{"*os.File", "StatusCode"}, primaryIdx: -1},
		"os.Mkdir":             {expr: "mkdir {0}", retTypes: []Type{"StatusCode"}},
		"os.MkdirAll":          {expr: "mkdir -p {0}", retTypes: []Type{"StatusCode"}},
		"os.Remove":            {expr: "rm -f", retTypes: []Type{"StatusCode"}},
		"os.RemoveAll":         {expr: "rm -rf", retTypes: []Type{"StatusCode"}},
		"os.Rename":            {expr: "mv", retTypes: []Type{"StatusCode"}},
		"os.File__WriteString": {expr: `echo -n {1} >&{0}`},
		"os.File__Close":       {expr: `eval "exec {0}<&- {0}>&-"`},
		"os.File__Fd":          {expr: `{0}`, retTypes: []Type{"int"}},
		"exec.Command":         {expr: "echo -n ", retTypes: []Type{"*exec.Cmd"}, stdout: true}, // TODO escape command string...
		"exec.Cmd__Output":     {expr: "bash -c", retTypes: []Type{"string", "StatusCode"}, stdout: true},
		"reflect.TypeOf":       {retTypes: []Type{"string"}, applyFunc: func(e *shExpression, arg []string) { e.expr = `"` + string(s.vars[varName(arg[0])]) + `"` }},
		"runtime.Compiler":     {expr: "'gotosh'", retTypes: []Type{"string"}},               // constant
		"runtime.GOARCH":       {expr: "uname -m", retTypes: []Type{"string"}, stdout: true}, // constant
		"runtime.GOOS":         {expr: "uname -o", retTypes: []Type{"string"}, stdout: true}, // constant
		// math (using bc)
		"math.Pi":   {expr: "3.141592653589793", retTypes: []Type{"float64"}}, // constant
		"math.E":    {expr: "2.718281828459045", retTypes: []Type{"float64"}}, // constant
		"math.Sqrt": {typ: "FLOAT_EXPR", expr: "sqrt({f0})", retTypes: []Type{"float64"}},
		"math.Pow":  {typ: "FLOAT_EXPR", expr: "e(l({f0})*{f1})", retTypes: []Type{"float64"}},
		"math.Exp":  {typ: "FLOAT_EXPR", expr: "e({f0})", retTypes: []Type{"float64"}},
		"math.Log":  {typ: "FLOAT_EXPR", expr: "l({f0})", retTypes: []Type{"float64"}},
		"math.Sin":  {typ: "FLOAT_EXPR", expr: "s({f0})", retTypes: []Type{"float64"}},
		"math.Cos":  {typ: "FLOAT_EXPR", expr: "c({f0})", retTypes: []Type{"float64"}},
		"math.Tan":  {typ: "FLOAT_EXPR", expr: "x={f0}; s(x)/c(x)", retTypes: []Type{"float64"}},
		"math.Atan": {typ: "FLOAT_EXPR", expr: "a({f0})", retTypes: []Type{"float64"}},
		"math.Sinh": {typ: "FLOAT_EXPR", expr: "x={f0}; ((e(x)-e(-x))/2)", retTypes: []Type{"float64"}},
		"math.Cosh": {typ: "FLOAT_EXPR", expr: "x={f0}; ((e(x)+e(-x))/2)", retTypes: []Type{"float64"}},
		"math.Tanh": {typ: "FLOAT_EXPR", expr: "x={f0}; ((e(x)-e(-x))/(e(x)+e(-x)))", retTypes: []Type{"float64"}},
		// TODO: cast
		"int":              {expr: "printf '%.0f' {0}", retTypes: []Type{"int"}, stdout: true},
		"byte":             {retTypes: []Type{"int"}},
		"float32":          {retTypes: []Type{"float64"}},
		"float64":          {retTypes: []Type{"float64"}},
		"string":           {retTypes: []Type{"string"}},
		"strconv.Atoi":     {retTypes: []Type{"int", "StatusCode"}},
		"strconv.Itoa":     {retTypes: []Type{"string"}},
		"shell.StatusCode": {retTypes: []Type{"int"}},
		// slice
		"len":    {retTypes: []Type{"int"}, applyFunc: func(e *shExpression, arg []string) { e.expr = "${#" + strings.Trim(trimQuote(arg[0]), "${}") + "}" }},
		"append": {retTypes: []Type{"[]any"}},
	}
}
