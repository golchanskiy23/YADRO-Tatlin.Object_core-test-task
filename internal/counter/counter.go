package counter

import (
	"os"
	"strings"
	"sync"
	"unicode"

	"name-frequency-counter/internal/parser"
	"name-frequency-counter/internal/queue"
)

type SafeMap struct {
	mu    sync.Mutex
	items map[string]*queue.Item
	q     *queue.MaxPriorityQueue
}

type WorkerPool struct {
	file    *os.File
	chunks  []parser.Chunk
	safeMap *SafeMap
}

func NewSafeMap(q *queue.MaxPriorityQueue) *SafeMap {
	return &SafeMap{
		items: make(map[string]*queue.Item),
		q:     q,
	}
}

func (m *SafeMap) Increment(name string) {
	m.mu.Lock()
	item, exists := m.items[name]
	if !exists {
		item = &queue.Item{Name: name, Count: 1}
		m.items[name] = item
		m.mu.Unlock()
		m.q.Push(item)
		return
	}
	item.Count++
	m.mu.Unlock()
	m.q.Fix(item) //nolint:errcheck
}

func NewWorkerPool(f *os.File, chunks []parser.Chunk, sm *SafeMap) *WorkerPool {
	return &WorkerPool{
		file:    f,
		chunks:  chunks,
		safeMap: sm,
	}
}

func isValidName(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && r != ' ' {
			return false
		}
	}
	return true
}

func (wp *WorkerPool) Run() {
	var wg sync.WaitGroup
	for _, chunk := range wp.chunks {
		wg.Add(1)
		go func(c parser.Chunk) {
			defer wg.Done()
			data, err := parser.ReadChunk(wp.file, c)
			if err != nil {
				return
			}
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if !isValidName(trimmed) {
					continue
				}
				wp.safeMap.Increment(trimmed)
			}
		}(chunk)
	}
	wg.Wait()
}