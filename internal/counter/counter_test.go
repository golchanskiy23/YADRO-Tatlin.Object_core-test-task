package counter

import (
	"os"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"name-frequency-counter/internal/parser"
	"name-frequency-counter/internal/queue"
)

func TestCountInvariant(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		lines []string
	}{
		{
			name:  "все строки непустые",
			lines: []string{"Миша", "Коля", "Миша", "Марина"},
		},
		{
			name:  "только пробельные строки",
			lines: []string{"", "   ", "\t", "  \t  "},
		},
		{
			name:  "смесь имён и пробельных строк",
			lines: []string{"Alice", "", "Bob", "   ", "Alice", "\t"},
		},
		{
			name:  "пустой ввод",
			lines: []string{},
		},
		{
			name:  "одно имя много раз",
			lines: []string{"X", "X", "X", "X", "X"},
		},
		{
			name:  "имя с двоеточием",
			lines: []string{"a:b", "a:b", "c:d"},
		},
		{
			name:  "UTF-8 имена",
			lines: []string{"Ян", "Ян", "Ён", "Ён", "Ён"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runInvariantCheck(t, tc.lines)
		})
	}

	t.Run("property: случайные строки", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			lines := rapid.SliceOf(
				rapid.OneOf(
					rapid.StringMatching(`\S+`),
					rapid.Just(""),
					rapid.Just("   "),
					rapid.Just("\t"),
				),
			).Draw(rt, "lines")

			runInvariantCheck(t, lines)
		})
	})
}

func runInvariantCheck(t *testing.T, lines []string) {
	t.Helper()

	q := queue.NewMaxPriorityQueue()
	sm := NewSafeMap(q)

	expected := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		expected++
		sm.Increment(trimmed)
	}

	got := 0
	for q.Len() > 0 {
		item, err := q.Pop()
		if err != nil {
			t.Fatalf("unexpected error popping from queue: %v", err)
		}
		got += item.Count
	}

	if got != expected {
		t.Fatalf("count invariant violated: sum of counts = %d, non-empty lines = %d", got, expected)
	}
}

func TestWorkerPoolRun(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		content  string
		wantSums map[string]int
	}{
		{
			name:    "базовый pipeline",
			content: "Миша\nКоля\nМиша\nМарина",
			wantSums: map[string]int{
				"Миша":   2,
				"Коля":   1,
				"Марина": 1,
			},
		},
		{
			name:     "пустой файл",
			content:  "",
			wantSums: map[string]int{},
		},
		{
			name:     "только пробельные строки",
			content:  "   \n\t\n",
			wantSums: map[string]int{},
		},
		{
			name:    "одно имя",
			content: "Alice",
			wantSums: map[string]int{
				"Alice": 1,
			},
		},
		{
			name:    "имя с двоеточием",
			content: "a:b\na:b",
			wantSums: map[string]int{
				"a:b": 2,
			},
		},
		{
			name:    "UTF-8",
			content: "Ян\nЁн\nЯн",
			wantSums: map[string]int{
				"Ян": 2,
				"Ён": 1,
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := writeTempFile(t, tc.content)
			f.Close()

			p := parser.NewParser(f.Name())
			resp := p.Split(2)
			if resp.Err != nil {
				t.Fatalf("Split error: %v", resp.Err)
			}
			if resp.File != nil {
				defer resp.File.Close()
			}

			q := queue.NewMaxPriorityQueue()
			sm := NewSafeMap(q)
			wp := NewWorkerPool(resp.File, resp.Chunk, sm)
			wp.Run()

			got := collectCounts(t, q)

			if len(got) != len(tc.wantSums) {
				t.Fatalf("got %d unique names, want %d; got=%v", len(got), len(tc.wantSums), got)
			}
			for name, wantCount := range tc.wantSums {
				if got[name] != wantCount {
					t.Errorf("name %q: got count %d, want %d", name, got[name], wantCount)
				}
			}
		})
	}
}

func TestWorkerPoolRunInvariant(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		names := rapid.SliceOfN(rapid.StringMatching(`\S+`), 1, 20).Draw(rt, "names")

		content := strings.Join(names, "\n")
		f := writeTempFile(t, content)
		f.Close()

		p := parser.NewParser(f.Name())
		resp := p.Split(2)
		if resp.Err != nil {
			t.Fatalf("Split error: %v", resp.Err)
		}
		if resp.File != nil {
			defer resp.File.Close()
		}

		q := queue.NewMaxPriorityQueue()
		sm := NewSafeMap(q)
		wp := NewWorkerPool(resp.File, resp.Chunk, sm)
		wp.Run()

		expected := 0
		for _, line := range strings.Split(content, "\n") {
			if strings.TrimSpace(line) != "" {
				expected++
			}
		}

		got := 0
		for q.Len() > 0 {
			item, err := q.Pop()
			if err != nil {
				t.Fatalf("pop error: %v", err)
			}
			got += item.Count
		}

		if got != expected {
			t.Fatalf("invariant violated: sum of counts = %d, want %d", got, expected)
		}
	})
}

func writeTempFile(t *testing.T, content string) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "counter_test_*.txt")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatalf("seek temp file: %v", err)
	}
	return f
}

func collectCounts(t *testing.T, q *queue.MaxPriorityQueue) map[string]int {
	t.Helper()
	result := make(map[string]int)
	for q.Len() > 0 {
		item, err := q.Pop()
		if err != nil {
			t.Fatalf("pop error: %v", err)
		}
		result[item.Name] = item.Count
	}
	return result
}
