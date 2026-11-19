package cmd

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/pecigonzalo/loro/lib"
	"github.com/spf13/cobra"
)

// streamsCmd represents the streams command
var streamsCmd = &cobra.Command{
	Use:   "streams",
	Short: "List available stream",
	RunE:  streams,
}

func init() {
	listCmd.AddCommand(streamsCmd)
}

func streams(cmd *cobra.Command, args []string) error {
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

	streams, err := logReader.ListStreams(ctx)
	if err != nil {
		return err
	}

	sort.Slice(streams, func(i, j int) bool {
		return aws.ToInt64(streams[i].LastIngestionTime) > aws.ToInt64(streams[j].LastIngestionTime)
	})

	rows := make([]string, 0, len(streams))
	for _, stream := range streams {
		rows = append(rows, fmt.Sprintf("%s\t%s\t%s",
			aws.ToString(stream.LogStreamName),
			lib.ParseAWSTimestamp(stream.LastIngestionTime).Local().Format(lib.ShortTimeFormat),
			lib.ParseAWSTimestamp(stream.CreationTime).Local().Format(lib.ShortTimeFormat),
		))
	}

	return printTable(cmd.OutOrStdout(), "Stream\tLast Event\tCreation", rows)
}
