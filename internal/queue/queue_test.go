package queue

import (
	"testing"

	"pgregory.net/rapid"
)

func TestQueueOrder(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		n := rapid.IntRange(1, 50).Draw(rt, "n")
		names := rapid.SliceOfN(
			rapid.StringMatching(`[a-zA-Zа-яА-ЯёЁ]{1,20}`),
			n, n,
		).Draw(rt, "names")
		counts := rapid.SliceOfN(rapid.IntRange(1, 100), n, n).Draw(rt, "counts")

		q := NewMaxPriorityQueue()
		for i := 0; i < n; i++ {
			q.Push(&Item{Name: names[i], Count: counts[i]})
		}

		var prev *Item
		for q.Len() > 0 {
			cur, err := q.Pop()
			if err != nil {
				rt.Fatalf("Pop() error: %v", err)
			}
			if prev != nil {
				if cur.Count > prev.Count {
					rt.Fatalf("order violation: got Count=%d after Count=%d", cur.Count, prev.Count)
				}
				if cur.Count == prev.Count && cur.Name < prev.Name {
					rt.Fatalf("order violation at equal count: got Name=%q after Name=%q", cur.Name, prev.Name)
				}
			}
			prev = cur
		}
	})
}

func TestIncrementAndFixOrder(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		n := rapid.IntRange(1, 30).Draw(rt, "n")
		names := rapid.SliceOfNDistinct(
			rapid.StringMatching(`[a-zA-Zа-яА-ЯёЁ]{1,15}`),
			n, n, func(s string) string { return s },
		).Draw(rt, "names")

		q := NewMaxPriorityQueue()
		items := make([]*Item, n)
		for i, name := range names {
			items[i] = &Item{Name: name, Count: 1}
			q.Push(items[i])
		}

		increments := rapid.SliceOfN(rapid.IntRange(0, 10), n, n).Draw(rt, "increments")
		for i, inc := range increments {
			for k := 0; k < inc; k++ {
				if err := q.IncrementAndFix(items[i]); err != nil {
					rt.Fatalf("IncrementAndFix error: %v", err)
				}
			}
		}

		for i, item := range items {
			want := 1 + increments[i]
			if item.Count != want {
				rt.Fatalf("item %q: Count=%d, want %d", item.Name, item.Count, want)
			}
		}

		var prev *Item
		for q.Len() > 0 {
			cur, err := q.Pop()
			if err != nil {
				rt.Fatalf("Pop() error: %v", err)
			}
			if prev != nil {
				if cur.Count > prev.Count {
					rt.Fatalf("order violation after IncrementAndFix: Count=%d after Count=%d", cur.Count, prev.Count)
				}
				if cur.Count == prev.Count && cur.Name < prev.Name {
					rt.Fatalf("order violation at equal count: Name=%q after Name=%q", cur.Name, prev.Name)
				}
			}
			prev = cur
		}
	})
}
