package printer

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"name-frequency-counter/internal/queue"
)

func assertFormat(t *testing.T, name string, count int) {
	t.Helper()

	item := &queue.Item{Name: name, Count: count}
	result := Format(item)

	expected := fmt.Sprintf("%s:%d", name, count)
	if result != expected {
		t.Fatalf("Format(%q, %d) = %q, want %q", name, count, result, expected)
	}

	colonIdx := strings.LastIndex(result, ":")
	if colonIdx < 0 {
		t.Fatalf("Format result %q contains no colon", result)
	}
	if result[:colonIdx] != name {
		t.Fatalf("Format result %q: part before last colon is %q, want %q", result, result[:colonIdx], name)
	}
	if result[colonIdx+1:] != fmt.Sprintf("%d", count) {
		t.Fatalf("Format result %q: part after last colon is %q, want %d", result, result[colonIdx+1:], count)
	}
	if strings.TrimSpace(result) != result {
		t.Fatalf("Format result %q has leading or trailing spaces", result)
	}
}

func TestPrinterFormatTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		count int
		want  string
	}{
		{name: "Миша", count: 2, want: "Миша:2"},
		{name: "Коля", count: 1, want: "Коля:1"},
		{name: "Марина", count: 1, want: "Марина:1"},
		{name: "Alice", count: 0, want: "Alice:0"},
		{name: "Bob", count: 1_000_000, want: "Bob:1000000"},
		{name: "name:with:colons", count: 3, want: "name:with:colons:3"},
		{name: "X", count: 42, want: "X:42"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			item := &queue.Item{Name: tc.name, Count: tc.count}
			got := Format(item)
			if got != tc.want {
				t.Fatalf("Format(%q, %d) = %q, want %q", tc.name, tc.count, got, tc.want)
			}
		})
	}
}

func TestPrinterFormat(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		name := rapid.StringMatching(`[^ \t\n\r\f\v]{1,50}`).Draw(rt, "name")
		count := rapid.IntRange(0, 1_000_000).Draw(rt, "count")

		assertFormat(t, name, count)
	})
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		n := rapid.IntRange(1, 20).Draw(rt, "n")

		type pair struct {
			name  string
			count int
		}

		pairs := make([]pair, n)
		for i := range pairs {
			name := rapid.StringMatching(`[^ \t\n\r\f\v]{1,50}`).Draw(rt, fmt.Sprintf("name_%d", i))
			count := rapid.IntRange(1, 1_000_000).Draw(rt, fmt.Sprintf("count_%d", i))
			pairs[i] = pair{name: name, count: count}
		}

		for _, p := range pairs {
			item := &queue.Item{Name: p.name, Count: p.count}
			formatted := Format(item)

			lastColon := strings.LastIndexByte(formatted, ':')
			if lastColon < 0 {
				rt.Fatalf("formatted output %q contains no colon", formatted)
			}

			parsedName := formatted[:lastColon]
			parsedCountStr := formatted[lastColon+1:]

			parsedCount, err := strconv.Atoi(parsedCountStr)
			if err != nil {
				rt.Fatalf("could not parse count from %q: %v", parsedCountStr, err)
			}

			if parsedName != p.name {
				rt.Fatalf("round-trip name mismatch: got %q, want %q (formatted: %q)", parsedName, p.name, formatted)
			}
			if parsedCount != p.count {
				rt.Fatalf("round-trip count mismatch: got %d, want %d (formatted: %q)", parsedCount, p.count, formatted)
			}
		}
	})
}
