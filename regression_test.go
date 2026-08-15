package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

var regressionExamples = []string{
	"hello_world",
	"fizz_buzz",
	"for_loop",
	"string_sample",
	"file_io",
	"math_sample",
	"map_sample",
	"lambda_sample",
	"misc",
}

const regressionTimeout = 30 * time.Second

func TestRegression(t *testing.T) {
	input, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range regressionExamples {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), regressionTimeout)
			defer cancel()

			script := runCommand(t, ctx, input, "go", "run", ".", filepath.Join("examples", name+".go"))
			want := runCommand(t, ctx, input, "go", "run", filepath.Join("examples", name+".go"), "aa", "bb", "123", "456")
			got := runShell(t, ctx, script)
			if !bytes.Equal(want, got) {
				t.Errorf("output mismatch (-want +got):\nwant:\n%s\ngot:\n%s", want, got)
			}
		})
	}
}

func runCommand(t *testing.T, ctx context.Context, input []byte, name string, args ...string) []byte {
	t.Helper()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = bytes.NewReader(input)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", strings.Join(cmd.Args, " "), err, stderr.String())
	}
	return out
}

func runShell(t *testing.T, ctx context.Context, script []byte) []byte {
	t.Helper()
	args := []string{"-u", "-s", "--", "aa", "bb", "123", "456"}
	command := "bash"
	if runtime.GOOS == "windows" {
		command = "wsl"
		args = append([]string{"bash"}, args...)
	}
	if _, err := exec.LookPath(command); err != nil {
		t.Skipf("%s is required to run regression tests: %v", command, err)
	}

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stdin = bytes.NewReader(script)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", strings.Join(cmd.Args, " "), err, out)
	}
	return out
}
