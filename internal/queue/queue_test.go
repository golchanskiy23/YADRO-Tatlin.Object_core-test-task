package queue

import (
	"sync"
	"testing"

	"pgregory.net/rapid"
)

func TestQueueOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 50).Draw(t, "n")
		names := rapid.SliceOfN(
			rapid.StringMatching(`[a-zA-Zа-яА-ЯёЁ]{1,20}`),
			n, n,
		).Draw(t, "names")
		counts := rapid.SliceOfN(rapid.IntRange(1, 100), n, n).Draw(t, "counts")

		q := NewMaxPriorityQueue()
		for i := 0; i < n; i++ {
			q.Push(&Item{Name: names[i], Count: counts[i]})
		}

		var prev *Item
		for q.Len() > 0 {
			cur := q.Pop()
			if prev != nil {
				if cur.Count > prev.Count {
					t.Fatalf("order violation: got Count=%d after Count=%d", cur.Count, prev.Count)
				}
				if cur.Count == prev.Count && cur.Name < prev.Name {
					t.Fatalf("order violation at equal count: got Name=%q after Name=%q", cur.Name, prev.Name)
				}
			}
			prev = cur
		}
	})
}

func TestQueueConcurrency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numGoroutines := rapid.IntRange(2, 8).Draw(t, "numGoroutines")
		itemsPerGoroutine := rapid.IntRange(1, 10).Draw(t, "itemsPerGoroutine")
		n := numGoroutines * itemsPerGoroutine

		names := rapid.SliceOfN(
			rapid.StringMatching(`[a-zA-Zа-яА-ЯёЁ]{1,15}`),
			n, n,
		).Draw(t, "names")
		counts := rapid.SliceOfN(rapid.IntRange(1, 50), n, n).Draw(t, "counts")

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
						q.Fix(item, 1)
					}
				} else {
					for range ownedItems {
						_ = q.Pop()
					}
				}
			}(g, goroutineItems[g])
		}
		wg.Wait()

		remaining := q.Len()
		if remaining < 0 {
			t.Fatalf("queue Len() is negative after concurrent ops: %d", remaining)
		}

		var prev *Item
		for q.Len() > 0 {
			cur := q.Pop()
			if cur == nil {
				t.Fatal("Pop() returned nil on non-empty queue")
			}
			if prev != nil {
				if cur.Count > prev.Count {
					t.Fatalf("ordering violated after concurrent ops: Count=%d after Count=%d", cur.Count, prev.Count)
				}
				if cur.Count == prev.Count && cur.Name < prev.Name {
					t.Fatalf("ordering violated after concurrent ops: Name=%q after Name=%q at equal count", cur.Name, prev.Name)
				}
			}
			prev = cur
		}
	})
}
