package cmd

import (
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use: "list",
}

func init() {
	rootCmd.AddCommand(listCmd)
}
