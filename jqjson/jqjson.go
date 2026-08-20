package jqjson

import (
	"os/exec"
	"strings"

	"github.com/binzume/gotosh/shell"
)

func run(input string, args ...string) (string, error) {
	cmd := exec.Command("jq", args...)
	cmd.Stdin = strings.NewReader(input)
	output, err := cmd.CombinedOutput()
	result := strings.TrimRight(string(output), "\r\n")
	if err == nil {
		return result, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		code := exitErr.ExitCode()
		if code > 0 && code <= 255 {
			return result, shell.StatusCode(code)
		}
	}
	return result, shell.StatusCode(1)
}

// Filter evaluates a jq filter against input. options are passed to jq before filter.
func Filter(input, filter string, options ...string) (string, error) {
	args := append([]string{}, options...)
	args = append(args, filter)
	return run(input, args...)
}

// Build evaluates a jq filter without an input value.
func Build(filter string, options ...string) (string, error) {
	args := append([]string{"-n"}, options...)
	args = append(args, filter)
	return run("", args...)
}

// Type returns the jq type of the input value.
func Type(input string) (string, error) {
	return Filter(input, "type", "-r")
}

// Get evaluates path against input and returns string values without JSON quotes.
func Get(input, path string) (string, error) {
	return Filter(input, path, "-r")
}

// Append appends a JSON value to the array at path in input.
func Append(input, path, value string) (string, error) {
	return Filter(input, path+" += [$value]", "--argjson", "value", value)
}

// AppendString appends a string value to the array at path in input.
func AppendString(input, path, value string) (string, error) {
	return Filter(input, path+" += [$value]", "--arg", "value", value)
}
