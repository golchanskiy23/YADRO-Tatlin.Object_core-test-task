package main

import (
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"name-frequency-counter/internal/counter"
	"name-frequency-counter/internal/parser"
	"name-frequency-counter/internal/printer"
	"name-frequency-counter/internal/queue"
)

var (
	topN       int
	outputPath string
)

var rootCmd = &cobra.Command{
	Use:   "name-frequency-counter",
	Short: "Count name frequencies from a text file",
}

var countCmd = &cobra.Command{
	Use:   "count <file>",
	Short: "Count name frequencies from a text file",
	Args:  cobra.ExactArgs(1),
	RunE:  runCount,
}

func init() {
	countCmd.Flags().IntVar(&topN, "top", 0, "Output only the top N entries (0 = all)")
	countCmd.Flags().StringVar(&outputPath, "output", "", "Write output to file instead of stdout")
	rootCmd.AddCommand(countCmd)
}

func runCount(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	p := parser.NewParser(filePath)
	resp := p.Split(runtime.GOMAXPROCS(0))
	if resp.Err != nil {
		return resp.Err
	}

	// Empty file — print nothing and exit 0.
	if len(resp.Chunk) == 0 {
		return nil
	}

	f := resp.File
	defer f.Close()

	q := queue.NewMaxPriorityQueue()
	sm := counter.NewSafeMap(q)
	wp := counter.NewWorkerPool(f, resp.Chunk, sm)
	wp.Run()

	// Determine output writer: stdout by default, file if --output is set.
	var out io.Writer = os.Stdout
	if outputPath != "" {
		outFile, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("count: open output file %q: %w", outputPath, err)
		}
		defer outFile.Close()
		out = outFile
	}

	// Drain the priority queue.
	limit := topN
	for q.Len() > 0 {
		item, err := q.Pop()
		if err != nil {
			return fmt.Errorf("count: pop: %w", err)
		}
		fmt.Fprintln(out, printer.Format(item))
		if limit > 0 {
			limit--
			if limit == 0 {
				break
			}
		}
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
