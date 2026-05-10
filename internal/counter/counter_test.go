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

func TestWhitespaceIgnored(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		base   []string
		extras []string
	}{
		{
			name:   "пустые строки не меняют результат",
			base:   []string{"Миша", "Коля", "Миша"},
			extras: []string{"", "", ""},
		},
		{
			name:   "строки из пробелов не меняют результат",
			base:   []string{"Alice", "Bob"},
			extras: []string{"   ", "  ", " "},
		},
		{
			name:   "табуляции не меняют результат",
			base:   []string{"X", "X", "Y"},
			extras: []string{"\t", "\t\t"},
		},
		{
			name:   "смесь пробелов и табуляций",
			base:   []string{"Ян", "Ён", "Ян"},
			extras: []string{" \t ", "\t ", "  \t"},
		},
		{
			name:   "только одно имя с пробельными строками",
			base:   []string{"Solo"},
			extras: []string{"", "   ", "\t"},
		},
		{
			name:   "имя с двоеточием, пробельные строки игнорируются",
			base:   []string{"a:b", "a:b", "c:d"},
			extras: []string{"", "   "},
		},
		{
			name:   "пробельные строки в начале и конце",
			base:   []string{"Марина", "Марина"},
			extras: []string{"   ", "\t", ""},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			q1 := queue.NewMaxPriorityQueue()
			sm1 := NewSafeMap(q1)
			for _, name := range tc.base {
				sm1.Increment(name)
			}
			baseResult := collectCounts(t, q1)

			mixed := append(tc.base, tc.extras...)
			q2 := queue.NewMaxPriorityQueue()
			sm2 := NewSafeMap(q2)
			for _, line := range mixed {
				trimmed := strings.TrimSpace(line)
				if trimmed == "" {
					continue
				}
				sm2.Increment(trimmed)
			}
			mixedResult := collectCounts(t, q2)

			if len(baseResult) != len(mixedResult) {
				t.Fatalf("result size mismatch: base=%d, mixed=%d; base=%v, mixed=%v",
					len(baseResult), len(mixedResult), baseResult, mixedResult)
			}
			for name, count := range baseResult {
				if mixedResult[name] != count {
					t.Errorf("count mismatch for %q: base=%d, mixed=%d", name, count, mixedResult[name])
				}
			}
		})
	}

	t.Run("property: случайные имена и пробельные строки", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			names := rapid.SliceOfN(rapid.StringMatching(`[!-~А-яЁё]+`), 1, 20).Draw(rt, "names")

			whitespaceLines := rapid.SliceOfN(
				rapid.StringMatching(`[ \t\r]+`),
				0, 10,
			).Draw(rt, "whitespaceLines")

			q1 := queue.NewMaxPriorityQueue()
			sm1 := NewSafeMap(q1)
			for _, name := range names {
				sm1.Increment(name)
			}
			baseResult := collectCounts(t, q1)

			mixed := make([]string, 0, len(names)+len(whitespaceLines))
			mixed = append(mixed, names...)
			mixed = append(mixed, whitespaceLines...)

			q2 := queue.NewMaxPriorityQueue()
			sm2 := NewSafeMap(q2)
			for _, line := range mixed {
				trimmed := strings.TrimSpace(line)
				if trimmed == "" {
					continue
				}
				sm2.Increment(trimmed)
			}
			mixedResult := collectCounts(t, q2)

			if len(baseResult) != len(mixedResult) {
				rt.Fatalf("result size mismatch: base=%d, mixed=%d; base=%v, mixed=%v",
					len(baseResult), len(mixedResult), baseResult, mixedResult)
			}
			for name, count := range baseResult {
				if mixedResult[name] != count {
					rt.Fatalf("count mismatch for %q: base=%d, mixed=%d", name, count, mixedResult[name])
				}
			}
		})
	})
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

// Feature: name-frequency-counter, Property 3: Регистр имён сохраняется без изменений
// Validates: Requirements 3.3
func TestCasePreserved(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
	}{
		{name: "верхний регистр", input: "ALICE"},
		{name: "смешанный регистр", input: "AlIcE"},
		{name: "нижний регистр", input: "alice"},
		{name: "кириллица верхний", input: "МИША"},
		{name: "кириллица смешанный", input: "МиШа"},
		{name: "только цифры и буквы", input: "User123"},
		{name: "имя с двоеточием и регистром", input: "A:B"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			q := queue.NewMaxPriorityQueue()
			sm := NewSafeMap(q)
			sm.Increment(tc.input)

			item, err := q.Pop()
			if err != nil {
				t.Fatalf("unexpected error popping from queue: %v", err)
			}
			if item.Name != tc.input {
				t.Fatalf("case not preserved: got %q, want %q", item.Name, tc.input)
			}
		})
	}

	t.Run("property: регистр сохраняется для любого имени", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			// Generate names that contain at least one non-whitespace character
			name := rapid.StringMatching(`\S+`).Draw(rt, "name")

			q := queue.NewMaxPriorityQueue()
			sm := NewSafeMap(q)
			sm.Increment(name)

			item, err := q.Pop()
			if err != nil {
				rt.Fatalf("unexpected error popping from queue: %v", err)
			}
			if item.Name != name {
				rt.Fatalf("case not preserved: got %q, want %q", item.Name, name)
			}
		})
	})
}
