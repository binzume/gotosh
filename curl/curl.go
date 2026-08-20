package curl

import (
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/binzume/gotosh/shell"
)

func newRequest(method, url string, body io.Reader, headers []string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, shell.StatusCode(1)
	}
	for _, header := range headers {
		name, value, ok := strings.Cut(header, ":")
		if !ok || strings.TrimSpace(name) == "" {
			return nil, shell.StatusCode(1)
		}
		req.Header.Add(strings.TrimSpace(name), strings.TrimSpace(value))
	}
	return req, nil
}

func do(req *http.Request) ([]byte, error) {
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err := client.Do(req)
	if err != nil {
		return nil, shell.StatusCode(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, shell.StatusCode(1)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, shell.StatusCode(22)
	}
	return body, nil
}

// Do performs an HTTP request with a string body and optional Header: value arguments.
func Do(method, url, body string, headers ...string) (string, error) {
	req, err := newRequest(method, url, strings.NewReader(body), headers)
	if err != nil {
		return "", err
	}
	result, err := do(req)
	return string(result), err
}

// Get performs an HTTP GET with optional Header: value arguments.
func Get(url string, headers ...string) (string, error) {
	return Do(http.MethodGet, url, "", headers...)
}

// Post performs an HTTP POST with a string body and optional Header: value arguments.
func Post(url, body string, headers ...string) (string, error) {
	req, err := newRequest(http.MethodPost, url, strings.NewReader(body), headers)
	if err != nil {
		return "", err
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	result, err := do(req)
	return string(result), err
}

// Download saves an HTTP GET response to path.
// It returns shell.StatusCode as error: 22 for HTTP 4xx/5xx and 1 for other errors.
func Download(url, path string, headers ...string) error {
	req, err := newRequest(http.MethodGet, url, nil, headers)
	if err != nil {
		return err
	}
	body, err := do(req)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, body, 0o666); err != nil {
		return shell.StatusCode(1)
	}
	return nil
}
