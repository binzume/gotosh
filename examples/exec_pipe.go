package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/binzume/gotosh/shell"
)

func produce(_ *os.File, out *os.File, _ ...string) shell.StatusCode {
	fmt.Fprintln(out, "alpha")
	fmt.Fprintln(out, "beta")
	fmt.Fprintln(out, "gamma")
	return 0
}

func upper(in, out *os.File, _ ...string) shell.StatusCode {
	for {
		line, status := shell.ReadLine(in)
		if status != 0 {
			return 0
		}
		fmt.Fprintln(out, strings.ToUpper(line))
	}
}

func main() {
	shell.ExecPipe(
		shell.Bind(produce),
		shell.Bind(upper),
	)
}
