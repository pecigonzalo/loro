package cmd

import (
	"fmt"
	"io"
	"text/tabwriter"
)

// printTable writes a tab-aligned table to w. The header and each row are
// tab-separated column values.
func printTable(w io.Writer, header string, rows []string) error {
	tw := tabwriter.NewWriter(w, 0, 8, 2, '\t', 0)
	// tabwriter buffers, so write errors surface from Flush
	_, _ = fmt.Fprintln(tw, header)
	for _, row := range rows {
		_, _ = fmt.Fprintln(tw, row)
	}

	return tw.Flush()
}
