package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/golchanskiy23/name-frequency-counter/internal/counter"
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
	countCmd.Flags().IntVar(&topN, "top", 0, "Output only the top N entries (0 = output nothing)")
	countCmd.Flags().StringVar(&outputPath, "output", "", "Write output to file instead of stdout")
	rootCmd.AddCommand(countCmd)
}

func runCount(cmd *cobra.Command, args []string) error {
	var out io.Writer = os.Stdout
	if outputPath != "" {
		outFile, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("count: open output file %q: %w", outputPath, err)
		}
		defer outFile.Close()
		out = outFile
	}
	if err := counter.RunCountWithWriter(out, args[0], topN); err != nil {
		return fmt.Errorf("count: %w", err)
	}
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
