package bash

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

type StatusCode = byte
type TempVarInt = int
type TempVarString = string

var IFS = " \t\n"

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

func Export(s ...string) {
	// do nothing in Go
}

func SubStr(s string, pos, len int) string {
	return s[pos : pos+len]
}

func Arg(n int) string {
	if n >= len(os.Args) {
		return ""
	}
	return os.Args[n]
}

func NArgs() int {
	return len(os.Args)
}

// TODO: coreutil.Sleep
func Sleep(t float32) {
	time.Sleep(time.Duration(t*1000) * time.Millisecond)
}

func UnixTimeMs() int {
	return int(time.Now().UnixMilli())
}
