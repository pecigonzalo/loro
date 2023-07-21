package cmd

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"text/template"
	"time"

	"github.com/pecigonzalo/loro/lib"
	"github.com/segmentio/events/v2"
	"github.com/spf13/cobra"
)

const (
	defaultFormatString = `[ {{ uniquecolor (print .Stream) }} ] {{ .TimeShort }} - {{ .Event.message }}`
	rawFormatString     = `{{ .PrettyPrint }}`
)

var templateFuncMap = template.FuncMap{
	"red":         lib.Red,
	"green":       lib.Green,
	"yellow":      lib.Yellow,
	"blue":        lib.Blue,
	"magenta":     lib.Magenta,
	"cyan":        lib.Cyan,
	"white":       lib.White,
	"uniquecolor": lib.Unique,
}

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get logs from a group or stream",
	RunE:  get,
}

var (
	follow        bool
	prefix        string
	eventTemplate string
	raw           bool
)

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().StringVarP(&prefix, "prefix", "p", "", "Stream Name or prefix")
	getCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log streams")
	getCmd.Flags().StringVarP(&eventTemplate, "format", "o", defaultFormatString, "Format template for displaying log events")
	getCmd.Flags().StringVarP(&since, "since", "s", "1h", "Fetch logs since timestamp (e.g. 2013-01-02T13:23:37), relative (e.g. 42m for 42 minutes), or all for all logs")
	getCmd.Flags().StringVarP(&until, "until", "u", "now", "Fetch logs until timestamp (e.g. 2013-01-02T13:23:37) or relative (e.g. 42m for 42 minutes)")
	getCmd.Flags().IntVarP(&maxStreams, "max-streams", "m", 10, "Maximum number of streams to fetch from (for prefix search)")
	getCmd.Flags().BoolVarP(&raw, "raw", "r", false, "Raw JSON output")
}

func get(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	group := "/"

	if len(args) > 0 {
		group = args[0]
	}

	start, err := lib.GetTime(since, time.Now())
	if err != nil {
		return fmt.Errorf("failed to parse time '%s'", since)
	}

	var end time.Time
	if cmd.Flags().Lookup("until").Changed {
		if cmd.Flags().Lookup("follow").Changed {
			return fmt.Errorf("can't set both --until and --follow")
		}
		end, err = lib.GetTime(until, time.Now())
		if err != nil {
			return fmt.Errorf("failed to parse time '%s'", until)
		}
	}

	lib.SetMaxStreams(100)

	logReader, err := lib.NewCloudwatchLogsReader(group, prefix, start, end)
	if err != nil {
		return err
	}

	// Try and fetch the group to verify it exists
	_, err = logReader.GetGroup(ctx)
	if err != nil {
		return err
	}

	if raw {
		eventTemplate = rawFormatString
	}

	output, err := template.New("event").Funcs(templateFuncMap).Parse(eventTemplate)
	if err != nil {
		return err
	}

	ctx, cancel := events.WithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	eventChan := logReader.StreamEvents(ctx, follow)

	ticker := time.After(7 * time.Second)
ReadLoop:
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				break ReadLoop
			}

			err = output.Execute(os.Stdout, event)
			if err != nil {
				fmt.Fprint(os.Stdout, err.Error())
				return err
			}

			fmt.Fprintf(os.Stdout, "\n")
			// reset slow log warning timer
			ticker = time.After(7 * time.Second)
		case <-ticker:
			if !follow {
				fmt.Fprintf(os.Stdout, "logs are taking a while to load... possibly try a smaller time window")
			}
		}
	}

	if err := logReader.Error(); err != nil {
		if err == context.Canceled {
			return nil
		}

		return err
	}

	return nil

}
