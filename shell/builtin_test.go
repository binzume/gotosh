package shell

import (
	"bufio"
	"io"
	"os"
	"strings"
	"testing"
)

func TestExecPipe(t *testing.T) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
		_ = r.Close()
		_ = w.Close()
	})

	status := ExecPipe(
		Bind(func(_ *os.File, out *os.File, _ ...string) StatusCode {
			_, _ = out.WriteString("hello\n")
			return 0
		}),
		Bind(func(in, out *os.File, _ ...string) StatusCode {
			scanner := bufio.NewScanner(in)
			for scanner.Scan() {
				_, _ = out.WriteString(strings.ToUpper(scanner.Text()) + "!\n")
			}
			if scanner.Err() != nil {
				return 1
			}
			return 0
		}),
	)
	if status != 0 {
		t.Fatalf("ExecPipe status = %d, want 0", status)
	}

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "HELLO!\n" {
		t.Fatalf("ExecPipe output = %q, want %q", got, "HELLO!\n")
	}
}
