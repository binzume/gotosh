package main

import (
	"bytes"
	"context"
	"flag"
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
	"exec_pipe",
	"math_sample",
	"lambda_sample",
	"misc",
	// bash only
	"pointer_sample",
	"map_sample",
}

const regressionTimeout = 30 * time.Second

var regressionShell = flag.String("shell", defaultRegressionShell(), "shell command for regression tests")

func defaultRegressionShell() string {
	if shell := os.Getenv("TEST_SHELL"); shell != "" {
		return shell
	}
	if runtime.GOOS == "windows" {
		return "wsl bash"
	}
	return "bash"
}

func TestRegression(t *testing.T) {
	input, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range regressionExamples {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), regressionTimeout)
			defer cancel()

			script, transpileStderr := runCommand(t, ctx, input, "go", "run", ".", filepath.Join("examples", name+".go"))
			if transpileStderr != "" {
				t.Errorf("transpiler wrote to stderr: %q", transpileStderr)
			}
			want, _ := runCommand(t, ctx, input, "go", "run", filepath.Join("examples", name+".go"), "aa", "bb", "123", "456")
			got := runShell(t, ctx, script)
			if !bytes.Equal(want, got) {
				t.Errorf("output mismatch (-want +got):\nwant:\n%s\ngot:\n%s", want, got)
			}
		})
	}
}

func runCommand(t *testing.T, ctx context.Context, input []byte, name string, args ...string) ([]byte, string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = bytes.NewReader(input)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", strings.Join(cmd.Args, " "), err, stderr.String())
	}
	return out, stderr.String()
}

func runShell(t *testing.T, ctx context.Context, script []byte) []byte {
	t.Helper()
	parts := strings.Fields(*regressionShell)
	if len(parts) == 0 {
		t.Fatal("-shell must specify a command")
	}
	args := []string{"-u", "-s", "--", "aa", "bb", "123", "456"}
	command := parts[0]
	args = append(parts[1:], args...)
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
