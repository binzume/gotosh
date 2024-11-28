package bash

import (
	"fmt"
	"os"
	"os/exec"
)

type StdoutInt = int
type StdoutString = string
type StatusCode = byte
type TempVarInt = int
type TempVarString = string

var IFS = " \t\n"

func Echo(params ...any) {
	fmt.Println(params...)
}

func EchoN(params ...any) {
	fmt.Print(params...)
}

func Printf(format string, a ...any) {
	fmt.Printf(format, a...)
}

func Sprintf(format string, a ...any) StdoutString {
	return fmt.Sprintf(format, a...)
}

func Exit(code int) {
	os.Exit(code)
}

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

func Cd(d string) StatusCode {
	if os.Chdir(d) != nil {
		return 1
	}
	return 0
}

func Pwd() StdoutString {
	pwd, _ := os.Getwd()
	return pwd
}

func Export(s ...string) {
}

// TODO: coreutil.Sleep
func Sleep(t float32) {
}
