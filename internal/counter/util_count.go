package counter

import (
	"fmt"
	"io"
	"runtime"

	"github.com/golchanskiy23/name-frequency-counter/internal/parser"
	"github.com/golchanskiy23/name-frequency-counter/internal/printer"
	"github.com/golchanskiy23/name-frequency-counter/internal/queue"
)

func RunCountWithWriter(out io.Writer, filePath string, top int) error {
	if top < 0 {
		return fmt.Errorf("count: --top must be a non-negative integer, got %d", top)
	}

	if top == 0 {
		return nil
	}

	p := parser.NewParser(filePath)
	resp := p.Split(runtime.GOMAXPROCS(0))
	if resp.Err != nil {
		return resp.Err
	}

	if len(resp.Chunk) == 0 {
		return nil
	}

	f := resp.File
	defer f.Close()

	q := queue.NewMaxPriorityQueue()
	sm := NewSafeMap(q)
	wp := NewWorkerPool(f, resp.Chunk, sm)
	wp.Run()

	for printed := 0; q.Len() > 0 && printed < top; printed++ {
		item, err := q.Pop()
		if err != nil {
			return fmt.Errorf("count: pop: %w", err)
		}
		fmt.Fprintln(out, printer.Format(item))
	}

	return nil
}
