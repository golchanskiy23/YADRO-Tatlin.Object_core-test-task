package parser

import (
	"os"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

func TestParallelEqualsSequential(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{"property_parallel_equals_sequential"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rapid.Check(t, func(rt *rapid.T) {
				names := rapid.SliceOfN(
					rapid.StringMatching(`[a-zA-Zа-яА-ЯёЁ]{1,20}`),
					1, 50,
				).Draw(rt, "names")

				numBlanks := rapid.IntRange(0, 10).Draw(rt, "numBlanks")

				lines := make([]string, len(names))
				copy(lines, names)

				for i := 0; i < numBlanks; i++ {
					blank := rapid.StringMatching(`[ \t]*`).Draw(rt, "blank")
					pos := rapid.IntRange(0, len(lines)).Draw(rt, "blankPos")
					lines = append(lines, "")
					copy(lines[pos+1:], lines[pos:])
					lines[pos] = blank
				}

				content := strings.Join(lines, "\n")

				f, err := os.CreateTemp("", "parser_test_*.txt")
				if err != nil {
					rt.Fatalf("failed to create temp file: %v", err)
				}
				defer os.Remove(f.Name())

				if _, err := f.WriteString(content); err != nil {
					f.Close()
					rt.Fatalf("failed to write temp file: %v", err)
				}
				if err := f.Close(); err != nil {
					rt.Fatalf("failed to close temp file: %v", err)
				}

				n := rapid.IntRange(1, 8).Draw(rt, "n")

				p := NewParser(f.Name())
				resp := p.Split(n)
				if resp.Err != nil {
					rt.Fatalf("Split error: %v", resp.Err)
				}
				defer func() {
					if resp.File != nil {
						resp.File.Close()
					}
				}()

				parallelMultiset := make(map[string]int)
				for _, chunk := range resp.Chunk {
					data, err := ReadChunk(resp.File, chunk)
					if err != nil {
						rt.Fatalf("ReadChunk error: %v", err)
					}
					for _, line := range strings.Split(string(data), "\n") {
						if trimmed := strings.TrimSpace(line); trimmed != "" {
							parallelMultiset[trimmed]++
						}
					}
				}

				seqMultiset := make(map[string]int)
				for _, line := range strings.Split(content, "\n") {
					if trimmed := strings.TrimSpace(line); trimmed != "" {
						seqMultiset[trimmed]++
					}
				}

				if len(parallelMultiset) != len(seqMultiset) {
					rt.Fatalf("multiset size mismatch: parallel=%d, sequential=%d\nparallel=%v\nsequential=%v",
						len(parallelMultiset), len(seqMultiset), parallelMultiset, seqMultiset)
				}
				for name, seqCount := range seqMultiset {
					parCount, ok := parallelMultiset[name]
					if !ok {
						rt.Fatalf("name %q present in sequential but missing in parallel", name)
					}
					if parCount != seqCount {
						rt.Fatalf("count mismatch for %q: parallel=%d, sequential=%d", name, parCount, seqCount)
					}
				}
			})
		})
	}
}
