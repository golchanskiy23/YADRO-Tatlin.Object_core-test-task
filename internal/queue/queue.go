package queue

import (
	"container/heap"
	"fmt"
)

type ErrEmptyQueue struct{}

func (e *ErrEmptyQueue) Error() string {
	return "queue: pop from empty queue"
}

type ErrInvalidIndex struct {
	Name  string
	Index int
}

func (e *ErrInvalidIndex) Error() string {
	return fmt.Sprintf("queue: fix item %q: invalid index %d", e.Name, e.Index)
}

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
	heap maxHeap
}

func NewMaxPriorityQueue() *MaxPriorityQueue {
	q := &MaxPriorityQueue{}
	heap.Init(&q.heap)
	return q
}

func (q *MaxPriorityQueue) Push(item *Item) {
	heap.Push(&q.heap, item)
}

func (q *MaxPriorityQueue) IncrementAndFix(item *Item) error {
	if item.index < 0 || item.index >= len(q.heap) {
		return &ErrInvalidIndex{Name: item.Name, Index: item.index}
	}
	item.Count++
	heap.Fix(&q.heap, item.index)
	return nil
}

func (q *MaxPriorityQueue) Pop() (*Item, error) {
	if len(q.heap) == 0 {
		return nil, &ErrEmptyQueue{}
	}
	return heap.Pop(&q.heap).(*Item), nil
}

func (q *MaxPriorityQueue) Len() int {
	return q.heap.Len()
}
