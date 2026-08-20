package shell

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TempVarInt = int
type TempVarString = string

type StatusCode byte

// Stage is one streaming stage in an ExecPipe. The first two arguments are
// the input and output file descriptors. Additional arguments are stage-specific
// values supplied with Bind.
type Stage func(in, out *os.File, args ...string) StatusCode

// StageCall is the Go runtime representation of a bound stage. The
// transpiler handles Bind and ExecPipe specially and does not emit this value into
// the generated shell script.
type StageCall struct {
	fn   Stage
	args []string
}

func Bind(fn Stage, args ...string) StageCall {
	return StageCall{fn: fn, args: args}
}

func (s StatusCode) Error() string {
	return strconv.Itoa(int(s))
}

var IsShellScript = false

var currentArgs []string

func Exec(name string, args ...string) (string, StatusCode) {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return string(out), 1
	}
	return strings.TrimSuffix(string(out), "\n"), 0
}

func ReadLine(r io.Reader) (string, StatusCode) {
	line := make([]byte, 0, 100)
	for {
		b := make([]byte, 1)
		n, err := r.Read(b)
		if n > 0 {
			if b[0] == '\n' {
				break
			}
			line = append(line, b[0])
		}
		if err != nil {
			return string(line), 1
		}
	}
	return string(line), 0
}

func Read() (string, StatusCode) {
	return ReadLine(os.Stdin)
}

func Files(pattern string) []string {
	r, _ := filepath.Glob(pattern)
	return r
}

func Export(s ...string) {
	// do nothing in Go
}

func SubStr(s string, pos, len int) string {
	return s[pos : pos+len]
}

func Arg(n int) string {
	if n > 0 && n-1 < len(Args()) {
		return Args()[n-1]
	} else if len(os.Args) > 0 {
		return os.Args[0]
	}
	return ""
}

func NArgs() int {
	return len(os.Args)
}

func Args() []string {
	if currentArgs == nil {
		currentArgs = os.Args[1:]
	}
	return currentArgs
}

func SetArgs(args ...string) {
	currentArgs = args
}

// ExecPipe connects all stages with OS pipes and runs them concurrently. The
// status code follows normal shell pipeline semantics: the last stage's code
// is returned.
func ExecPipe(stages ...StageCall) StatusCode {
	if len(stages) == 0 {
		return 0
	}

	inputs := make([]*os.File, len(stages))
	outputs := make([]*os.File, len(stages))
	statuses := make([]StatusCode, len(stages))
	inputs[0] = os.Stdin
	outputs[len(stages)-1] = os.Stdout

	var created []*os.File
	for i := 0; i < len(stages)-1; i++ {
		r, w, err := os.Pipe()
		if err != nil {
			for _, f := range created {
				_ = f.Close()
			}
			return 1
		}
		inputs[i+1] = r
		outputs[i] = w
		created = append(created, r, w)
	}

	var wg sync.WaitGroup
	wg.Add(len(stages))
	for i, stage := range stages {
		go func(i int, stage StageCall) {
			defer wg.Done()
			if inputs[i] != os.Stdin {
				defer inputs[i].Close()
			}
			if outputs[i] != os.Stdout {
				defer outputs[i].Close()
			}
			if stage.fn == nil {
				statuses[i] = 1
				return
			}
			statuses[i] = stage.fn(inputs[i], outputs[i], stage.args...)
		}(i, stage)
	}

	wg.Wait()
	return statuses[len(statuses)-1]
}

func Do(rawScript string) StatusCode {
	// Do nothing in Go
	return 1
}

func SetFloatPrecision(a int) struct{} {
	// Do nothing in Go
	return struct{}{}
}

// TODO: coreutil.Sleep
func Sleep(t float32) {
	time.Sleep(time.Duration(t*1000) * time.Millisecond)
}

func UnixTimeMs() int {
	return int(time.Now().UnixMilli())
}
