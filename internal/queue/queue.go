package queue

import (
	"container/heap"
	"sync"
)

type Item struct {
	Name  string
	Count int
	index int
}

type maxHeap []*Item

func (h maxHeap) Len() int { return len(h) }

func (h maxHeap) Less(i, j int) bool {
	if h[i].Count != h[j].Count {
		return h[i].Count > h[j].Count
	}
	return h[i].Name < h[j].Name
}

func (h maxHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *maxHeap) Push(x any) {
	item := x.(*Item)
	item.index = len(*h)
	*h = append(*h, item)
}

func (h *maxHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[:n-1]
	return item
}

type MaxPriorityQueue struct {
	mu   sync.Mutex
	heap maxHeap
}

func NewMaxPriorityQueue() *MaxPriorityQueue {
	q := &MaxPriorityQueue{}
	heap.Init(&q.heap)
	return q
}

func (q *MaxPriorityQueue) Push(item *Item) {
	q.mu.Lock()
	defer q.mu.Unlock()
	heap.Push(&q.heap, item)
}

// Fix increments item.Count by delta and restores heap ordering.
// Both the count update and the heap fix happen under the same lock,
// preventing data races between writers and concurrent heap operations.
func (q *MaxPriorityQueue) Fix(item *Item, delta int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	item.Count += delta
	heap.Fix(&q.heap, item.index)
}

func (q *MaxPriorityQueue) Pop() *Item {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.heap) == 0 {
		return nil
	}
	return heap.Pop(&q.heap).(*Item)
}

func (q *MaxPriorityQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.heap.Len()
}
