package integration

import (
	"bytes"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/golchanskiy23/name-frequency-counter/internal/counter"
)

func executeCount(t *testing.T, filePath string, top int) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	err := counter.RunCountWithWriter(&buf, filePath, top)
	return buf.String(), err
}

func executeCountAll(t *testing.T, filePath string) (string, error) {
	t.Helper()
	return executeCount(t, filePath, math.MaxInt)
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "cli_test_*.txt")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}
	return f.Name()
}

func parseOutput(t *testing.T, output string) map[string]int {
	t.Helper()
	result := make(map[string]int)
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		if line == "" {
			continue
		}
		idx := strings.LastIndex(line, ":")
		if idx < 0 {
			t.Fatalf("parseOutput: unexpected line format: %q", line)
		}
		name := line[:idx]
		count, err := strconv.Atoi(line[idx+1:])
		if err != nil {
			t.Fatalf("parseOutput: cannot parse count in line %q: %v", line, err)
		}
		result[name] = count
	}
	return result
}

func TestCLIFullPipeline(t *testing.T) {
	path := writeTempFile(t, "Миша\nМиша\nКоля\nМарина")
	got, err := executeCountAll(t, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]int{"Миша": 2, "Коля": 1, "Марина": 1}
	if gotMap := parseOutput(t, got); !mapsEqual(gotMap, want) {
		t.Fatalf("output mismatch:\ngot:  %v\nwant: %v", gotMap, want)
	}
}

func mapsEqual(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func TestCLIEmptyFile(t *testing.T) {
	path := writeTempFile(t, "")
	got, err := executeCountAll(t, path)
	if err != nil {
		t.Fatalf("unexpected error for empty file: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty output for empty file, got: %q", got)
	}
}

func TestCLIWhitespaceOnlyFile(t *testing.T) {
	path := writeTempFile(t, "   \n\t\n  \t  \n")
	got, err := executeCountAll(t, path)
	if err != nil {
		t.Fatalf("unexpected error for whitespace-only file: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty output for whitespace-only file, got: %q", got)
	}
}

func TestCLIFileNotFound(t *testing.T) {
	_, err := executeCountAll(t, "/nonexistent/path/file.txt")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestCLIFileNoPermission(t *testing.T) {
	path := writeTempFile(t, "Миша\nКоля\n")
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(path, 0o644) }) //nolint:errcheck // best-effort cleanup in test
	if _, err := executeCountAll(t, path); err == nil {
		t.Fatal("expected error for unreadable file, got nil")
	}
}

func TestCLITopZero(t *testing.T) {
	path := writeTempFile(t, "Миша\nКоля\n")
	got, err := executeCount(t, path, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty output for --top 0, got: %q", got)
	}
}

func TestCLITopNegative(t *testing.T) {
	path := writeTempFile(t, "Миша\nКоля\n")
	_, err := executeCount(t, path, -1)
	if err == nil {
		t.Fatal("expected error for --top -1, got nil")
	}
}

func TestCLITopFlag(t *testing.T) {
	path := writeTempFile(t, "Миша\nМиша\nМиша\nКоля\nКоля\nМарина")
	got, err := executeCount(t, path, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]int{"Миша": 3, "Коля": 2}
	if gotMap := parseOutput(t, got); !mapsEqual(gotMap, want) {
		t.Fatalf("--top 2 mismatch:\ngot:  %v\nwant: %v", gotMap, want)
	}
}

func TestCLIIdempotent(t *testing.T) {
	path := writeTempFile(t, "Миша\nМиша\nКоля\nМарина")
	first, err := executeCountAll(t, path)
	if err != nil {
		t.Fatalf("first run error: %v", err)
	}
	second, err := executeCountAll(t, path)
	if err != nil {
		t.Fatalf("second run error: %v", err)
	}
	if first != second {
		t.Fatalf("idempotency violation:\nfirst:  %q\nsecond: %q", first, second)
	}
}
