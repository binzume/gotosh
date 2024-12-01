package bash

import (
	"os"
	"os/exec"
	"time"
)

type StdoutInt = int
type StdoutString = string
type StatusCode = byte
type TempVarInt = int
type TempVarString = string

var IFS = " \t\n"

func Exec(cmd string) (StdoutString, StatusCode) {
	out, err := exec.Command("ls", "-la").Output()
	if err != nil {
		return string(out), 1
	}
	return string(out), 0
}

func Read() (StdoutString, StatusCode) {
	line := make([]byte, 0, 100)
	for {
		b := make([]byte, 1)
		n, err := os.Stdin.Read(b)
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

func Export(s ...string) {
}

func SubStr(s string, pos, len int) string {
	return s[pos : pos+len]
}

// TODO: coreutil.Sleep
func Sleep(t float32) {
	time.Sleep(time.Duration(t*1000) * time.Millisecond)
}

func UnixTimeMs() int {
	return int(time.Now().UnixMilli())
}
