package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
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

	rows := make([]string, 0, len(groups))
	for _, group := range groups {
		rows = append(rows, fmt.Sprintf("%s\t%s",
			aws.ToString(group.LogGroupName),
			lib.ParseAWSTimestamp(group.CreationTime).Local().Format(lib.ShortTimeFormat),
		))
	}

	return printTable("Group\tCreation", rows)
}
