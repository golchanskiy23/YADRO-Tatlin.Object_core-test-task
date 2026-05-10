package queue

import (
	"sync"
	"testing"

	"pgregory.net/rapid"
)

func TestQueueOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{"property_queue_order"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
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
		})
	}
}

func TestQueueConcurrency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{"property_queue_concurrency"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rapid.Check(t, func(rt *rapid.T) {
				numGoroutines := rapid.IntRange(2, 8).Draw(rt, "numGoroutines")
				itemsPerGoroutine := rapid.IntRange(1, 10).Draw(rt, "itemsPerGoroutine")
				n := numGoroutines * itemsPerGoroutine

				names := rapid.SliceOfN(
					rapid.StringMatching(`[a-zA-Zа-яА-ЯёЁ]{1,15}`),
					n, n,
				).Draw(rt, "names")
				counts := rapid.SliceOfN(rapid.IntRange(1, 50), n, n).Draw(rt, "counts")

				q := NewMaxPriorityQueue()
				goroutineItems := make([][]*Item, numGoroutines)
				for g := 0; g < numGoroutines; g++ {
					goroutineItems[g] = make([]*Item, itemsPerGoroutine)
					for k := 0; k < itemsPerGoroutine; k++ {
						idx := g*itemsPerGoroutine + k
						item := &Item{Name: names[idx], Count: counts[idx]}
						goroutineItems[g][k] = item
						q.Push(item)
					}
				}

				var wg sync.WaitGroup
				for g := 0; g < numGoroutines; g++ {
					wg.Add(1)
					go func(id int, ownedItems []*Item) {
						defer wg.Done()
						if id%2 == 0 {
							for _, item := range ownedItems {
								_ = q.Fix(item)
							}
						} else {
							for range ownedItems {
								_, _ = q.Pop()
							}
						}
					}(g, goroutineItems[g])
				}
				wg.Wait()

				var prev *Item
				for q.Len() > 0 {
					cur, err := q.Pop()
					if err != nil {
						rt.Fatalf("Pop() error after concurrent ops: %v", err)
					}
					if prev != nil {
						if cur.Count > prev.Count {
							rt.Fatalf("ordering violated after concurrent ops: Count=%d after Count=%d", cur.Count, prev.Count)
						}
						if cur.Count == prev.Count && cur.Name < prev.Name {
							rt.Fatalf("ordering violated after concurrent ops: Name=%q after Name=%q at equal count", cur.Name, prev.Name)
						}
					}
					prev = cur
				}
			})
		})
	}
}
