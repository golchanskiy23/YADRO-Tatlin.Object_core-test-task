package printer

import (
	"fmt"

	"name-frequency-counter/internal/queue"
)

func Format(item *queue.Item) string {
	return fmt.Sprintf("%s:%d", item.Name, item.Count)
}
