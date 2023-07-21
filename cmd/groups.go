package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/pecigonzalo/loro/lib"
	"github.com/spf13/cobra"
)

// groupsCmd represents the groups command
var groupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "List streams for a group",
	Args:  cobra.MaximumNArgs(1),

	RunE: groups,
}

func init() {
	listCmd.AddCommand(groupsCmd)
}

func groups(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	group := "/"

	if len(args) > 0 {
		group = args[0]
	}

	start, err := lib.GetTime(since, time.Now())
	if err != nil {
		return fmt.Errorf("failed to parse time '%s'", since)
	}

	end, err := lib.GetTime(until, time.Now())
	if err != nil {
		return fmt.Errorf("failed to parse time '%s'", until)
	}

	logReader, err := lib.NewCloudwatchLogsReader(group, prefix, start, end)
	if err != nil {
		return err
	}

	groups, err := logReader.ListGroups(ctx)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)
	fmt.Fprintln(w, "Group\tCreation")

	for _, group := range groups {
		fmt.Fprintf(w, "%s\t%s\n",
			*group.LogGroupName,
			lib.ParseAWSTimestamp(group.CreationTime).Local().Format(lib.ShortTimeFormat),
		)
	}
	w.Flush()

	return nil
}
