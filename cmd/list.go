package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// printTable writes a tab-aligned table with the given header and rows to stdout.
func printTable(header string, rows []string) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)
	// tabwriter buffers, so write errors surface from Flush
	_, _ = fmt.Fprintln(w, header)
	for _, row := range rows {
		_, _ = fmt.Fprintln(w, row)
	}

	return w.Flush()
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use: "list",
}

func init() {
	rootCmd.AddCommand(listCmd)
}
