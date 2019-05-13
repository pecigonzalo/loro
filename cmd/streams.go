package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/pecigonzalo/loro/lib"
	log "github.com/sirupsen/logrus"
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
	group := "/"

	if len(args) > 0 {
		group = args[0]
	}

	start, err := lib.GetTime(since, time.Now())
	if err != nil {
		log.Errorf("Failed to parse time '%s'", since)
		return err
	}

	end, err := lib.GetTime(until, time.Now())
	if err != nil {
		log.Errorf("Failed to parse time '%s'", since)
		return err
	}

	logReader, err := lib.NewCloudwatchLogsReader(group, prefix, start, end)
	if err != nil {
		return err
	}

	streams, err := logReader.ListStreams()
	if err != nil {
		return err
	}

	sort.Slice(streams, func(i, j int) bool {
		return *streams[i].LastIngestionTime > *streams[j].LastIngestionTime
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)
	fmt.Fprintln(w, "Stream\tLast Event\tCreation")

	for _, stream := range streams {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			*stream.LogStreamName,
			lib.ParseAWSTimestamp(stream.LastIngestionTime).Local().Format(lib.ShortTimeFormat),
			lib.ParseAWSTimestamp(stream.CreationTime).Local().Format(lib.ShortTimeFormat),
		)
	}
	w.Flush()

	return nil
}
