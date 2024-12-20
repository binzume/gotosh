package compiler

import (
	"strings"
)

// TODO: export types to modify from outside
var InitBuiltInFuncs = func(s *state) {
	s.funcs = map[string]shFunc{
		"shell.Sleep":         {exp: "sleep"},
		"shell.Exit":          {exp: "exit"},
		"shell.Export":        {exp: "export"},
		"shell.Exec":          {retTypes: []Type{"string", "StatusCode"}, stdout: true},
		"shell.Read":          {exp: `IFS= read -r -s _tmp0`, retTypes: []Type{"string", "StatusCode"}, primaryIdx: -1},
		"shell.ReadLine":      {exp: `IFS= read -r -s _tmp0 <&{0}`, retTypes: []Type{"string", "StatusCode"}, primaryIdx: -1},
		"shell.SubStr":        {exp: "\"${{*0}:{1}:{2}}\"", retTypes: []Type{"string"}},
		"shell.Arg":           {exp: `eval echo \${{0}}`, retTypes: []Type{"string"}, stdout: true},
		"shell.NArgs":         {exp: `$(( $# + 1 ))`, retTypes: []Type{"int"}},
		"shell.UnixTimeMs":    {exp: `printf '%.0f' $( echo "${EPOCHREALTIME:-$(date +%s)} * 1000" | bc )`, retTypes: []Type{"int"}, stdout: true},
		"shell.Do":            {retTypes: []Type{"StatusCode"}, convFunc: func(arg []string) string { return trimQuote(arg[0]) }, primaryIdx: -1},
		"shell.IsShellScript": {exp: "1", retTypes: []Type{"bool"}},
		// fmt
		"fmt.Print":   {exp: "echo -n"},
		"fmt.Println": {exp: "echo"},
		"fmt.Printf":  {exp: "printf"},
		"fmt.Sprint":  {exp: "echo -n", retTypes: []Type{"string"}, stdout: true},
		"fmt.Sprintln": {retTypes: []Type{"string"}, convFunc: func(arg []string) string {
			return "$(echo " + strings.Join(arg, " ") + ")$'\\n'"
		}},
		"fmt.Sprintf":  {exp: "printf", retTypes: []Type{"string"}, stdout: true},
		"fmt.Fprint":   {convFunc: func(arg []string) string { return "echo -n " + strings.Join(arg[1:], " ") + " >&" + arg[0] }},
		"fmt.Fprintln": {convFunc: func(arg []string) string { return "echo " + strings.Join(arg[1:], " ") + " >&" + arg[0] }},
		"fmt.Fprintf":  {convFunc: func(arg []string) string { return "printf " + strings.Join(arg[1:], " ") + " >&" + arg[0] }},
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
		"strings.Contains": {exp: "case {0} in (*{1}*) echo 1;; (*) echo 0;; esac", retTypes: []Type{"bool"}, stdout: true},
		"strings.IndexAny": {exp: "expr '(' index {0} {1} ')' - 1", retTypes: []Type{"int"}, stdout: true},
		// os
		"os.Stdin":    {exp: "0", retTypes: []Type{"*os.File"}},         // variable
		"os.Stdout":   {exp: "1", retTypes: []Type{"*os.File"}},         // variable
		"os.Stderr":   {exp: "1", retTypes: []Type{"*os.File"}},         // variable
		"os.Args":     {exp: `"$0" "$@"`, retTypes: []Type{"[]string"}}, // variable
		"os.Exit":     {exp: "exit"},
		"os.Getwd":    {exp: "pwd", retTypes: []Type{"string", "StatusCode"}, stdout: true},
		"os.Chdir":    {exp: "cd", retTypes: []Type{"StatusCode"}, stdout: true},
		"os.Getpid":   {exp: "$$", retTypes: []Type{"int"}},
		"os.Getppid":  {exp: "$PPID", retTypes: []Type{"int"}},
		"os.Getuid":   {exp: "${UID:--1}", retTypes: []Type{"int"}},
		"os.Geteuid":  {exp: "${EUID:-${UID:--1}}", retTypes: []Type{"int"}},
		"os.Getgid":   {exp: "${GID:--1}", retTypes: []Type{"int"}},
		"os.Getegid":  {exp: "${EGID:-${GID:--1}}", retTypes: []Type{"int"}},
		"os.Hostname": {exp: "uname -n", retTypes: []Type{"string", "StatusCode"}, stdout: true},
		"os.Getenv": {convFunc: func(arg []string) string {
			return "\"${" + trimQuote(arg[0]) + "}\""
		}, retTypes: []Type{"string"}},
		"os.Setenv": {convFunc: func(arg []string) string {
			return "export " + trimQuote(arg[0]) + "=" + arg[1]
		}},
		"os.Pipe": {exp: `_tmp=$(mktemp -d) && mkfifo $_tmp/f && _tmp0=$(( ++GOTOSH_fd + 2 )) && _tmp1=$(( ++GOTOSH_fd + 2 )) && eval "exec $_tmp1<>\"$_tmp/f\" $_tmp0<\"$_tmp/f\"" && rm -rf $_tmp`,
			retTypes: []Type{"*os.File", "*os.File", "StatusCode"}, primaryIdx: -1},
		"os.Open":              {exp: `_tmp0=$(( ++GOTOSH_fd + 2 )) ; eval "exec $_tmp0<"{0}`, retTypes: []Type{"*os.File", "StatusCode"}, primaryIdx: -1},
		"os.Create":            {exp: `_tmp0=$(( ++GOTOSH_fd + 2 )) ; eval "exec $_tmp0>"{0}`, retTypes: []Type{"*os.File", "StatusCode"}, primaryIdx: -1},
		"os.Mkdir":             {exp: "mkdir {0}", retTypes: []Type{"StatusCode"}},
		"os.MkdirAll":          {exp: "mkdir -p {0}", retTypes: []Type{"StatusCode"}},
		"os.Remove":            {exp: "rm -f", retTypes: []Type{"StatusCode"}},
		"os.RemoveAll":         {exp: "rm -rf", retTypes: []Type{"StatusCode"}},
		"os.Rename":            {exp: "mv", retTypes: []Type{"StatusCode"}},
		"os.File__WriteString": {exp: `echo -n {1} >&{0}`},
		"os.File__Close":       {exp: `eval "exec {0}<&- {0}>&-"`},
		"os.File__Fd":          {exp: `{0}`, retTypes: []Type{"int"}},
		"exec.Command":         {exp: "echo -n ", retTypes: []Type{"*exec.Cmd"}, stdout: true}, // TODO escape command string...
		"exec.Cmd__Output":     {exp: "bash -c", retTypes: []Type{"string", "StatusCode"}, stdout: true},
		"reflect.TypeOf":       {retTypes: []Type{"string"}, convFunc: func(arg []string) string { return `"` + string(s.vars[varName(arg[0])]) + `"` }},
		"runtime.Compiler":     {exp: "'gotosh'", retTypes: []Type{"string"}},               // constant
		"runtime.GOARCH":       {exp: "uname -m", retTypes: []Type{"string"}, stdout: true}, // constant
		"runtime.GOOS":         {exp: "uname -o", retTypes: []Type{"string"}, stdout: true}, // constant
		// math (using bc)
		"math.Pi":   {exp: "3.141592653589793", retTypes: []Type{"float64"}}, // constant
		"math.E":    {exp: "2.718281828459045", retTypes: []Type{"float64"}}, // constant
		"math.Sqrt": {typ: "FLOAT_EXP", exp: "sqrt({f0})", retTypes: []Type{"float64"}},
		"math.Pow":  {typ: "FLOAT_EXP", exp: "e(l({f0})*{f1})", retTypes: []Type{"float64"}},
		"math.Exp":  {typ: "FLOAT_EXP", exp: "e({f0})", retTypes: []Type{"float64"}},
		"math.Log":  {typ: "FLOAT_EXP", exp: "l({f0})", retTypes: []Type{"float64"}},
		"math.Sin":  {typ: "FLOAT_EXP", exp: "s({f0})", retTypes: []Type{"float64"}},
		"math.Cos":  {typ: "FLOAT_EXP", exp: "c({f0})", retTypes: []Type{"float64"}},
		"math.Tan":  {typ: "FLOAT_EXP", exp: "x={f0}; s(x)/c(x)", retTypes: []Type{"float64"}},
		"math.Atan": {typ: "FLOAT_EXP", exp: "a({f0})", retTypes: []Type{"float64"}},
		"math.Sinh": {typ: "FLOAT_EXP", exp: "x={f0}; ((e(x)-e(-x))/2)", retTypes: []Type{"float64"}},
		"math.Cosh": {typ: "FLOAT_EXP", exp: "x={f0}; ((e(x)+e(-x))/2)", retTypes: []Type{"float64"}},
		"math.Tanh": {typ: "FLOAT_EXP", exp: "x={f0}; ((e(x)-e(-x))/(e(x)+e(-x)))", retTypes: []Type{"float64"}},
		// TODO: cast
		"int":              {exp: "printf '%.0f' {0}", retTypes: []Type{"int"}, stdout: true},
		"byte":             {retTypes: []Type{"int"}},
		"float32":          {retTypes: []Type{"float32"}},
		"string":           {retTypes: []Type{"string"}},
		"strconv.Atoi":     {retTypes: []Type{"int", "StatusCode"}},
		"strconv.Itoa":     {retTypes: []Type{"string"}},
		"shell.StatusCode": {retTypes: []Type{"int"}},
		// slice
		"len": {retTypes: []Type{"int"}, convFunc: func(arg []string) string { return "${#" + strings.Trim(trimQuote(arg[0]), "${}") + "}" }},
		"append": {typ: "RET_ARG1", retTypes: []Type{"[]any"}, primaryIdx: -1,
			convFunc: func(arg []string) string {
				return varName(arg[0]) + "+=(" + strings.Join(arg[1:], " ") + ")"
			}},
	}
}
