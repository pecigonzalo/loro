package lib

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	lru "github.com/hashicorp/golang-lru/v2"
)

const (
	// MaxEventsPerCall is the maximum number events from a filter call
	MaxEventsPerCall = 10000
)

var (
	// MaxStreams is the maximum number of streams you can give to a filter call
	MaxStreams = 100
)

// CloudwatchLogsReader is responsible for fetching logs for a particular log
// group
type CloudwatchLogsReader struct {
	logGroupName string
	svc          *cloudwatchlogs.Client
	eventCache   *lru.Cache[string, any]
	start        time.Time
	end          time.Time
	error        error
	streamPrefix string
}

// SetMaxStreams sets the maximum number of streams for describe/filter calls
func SetMaxStreams(max int) {
	MaxStreams = max
}

// NewCloudwatchLogsReader takes a group and optionally a stream prefix, start and
// end time, and returns a reader for any logs that match those parameters.
func NewCloudwatchLogsReader(group string, streamPrefix string, start time.Time, end time.Time) (*CloudwatchLogsReader, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}

	// Extend default retry count to 10
	cfg.RetryMaxAttempts = 10

	svc := cloudwatchlogs.NewFromConfig(cfg)

	cache, err := lru.New[string, any](MaxEventsPerCall)
	if err != nil {
		return nil, err
	}

	reader := &CloudwatchLogsReader{
		logGroupName: group,
		svc:          svc,
		eventCache:   cache,
		start:        start,
		end:          end,
		streamPrefix: streamPrefix,
	}

	return reader, nil
}

// ListGroups returns a list of possible groups given a group name
func (c *CloudwatchLogsReader) ListGroups(ctx context.Context) ([]types.LogGroup, error) {
	return getLogGroups(ctx, c.svc, c.logGroupName)
}

func getLogGroups(ctx context.Context, svc *cloudwatchlogs.Client, name string) ([]types.LogGroup, error) {
	describeLogGroupsInput := &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(name),
	}

	paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(svc, describeLogGroupsInput)

	// Paginate through all of the log groups
	groups := []types.LogGroup{}
	for paginator.HasMorePages() {
		if page, err := paginator.NextPage(ctx); err != nil {
			return groups, err
		} else {
			groups = append(groups, page.LogGroups...)
		}
	}

	return groups, nil
}

// GetGroup returns a selected group given a group name
func (c *CloudwatchLogsReader) GetGroup(ctx context.Context) (types.LogGroup, error) {
	return getLogGroup(ctx, c.svc, c.logGroupName)
}

func getLogGroup(ctx context.Context, svc *cloudwatchlogs.Client, name string) (types.LogGroup, error) {
	groups, err := getLogGroups(ctx, svc, name)
	if err != nil {
		return types.LogGroup{}, err
	}

	if len(groups) == 0 {
		return types.LogGroup{}, fmt.Errorf("could not find log group '%s'", name)
	}

	if *groups[0].LogGroupName != name {
		// Didn't find exact match, offer some alternatives based on prefix
		errMsg := fmt.Sprintf("Could not find log group '%s'.\n\nDid you mean:\n\n", name)
		for ix, group := range groups {
			if ix > 4 {
				break
			}
			errMsg += fmt.Sprintf("%s\n", *group.LogGroupName)
		}
		return types.LogGroup{}, errors.New(errMsg)
	}

	return groups[0], nil
}

// ListStreams returns any log streams that match the params given in the
// reader's constructor.  Will return at most `MaxStreams` streams
func (c *CloudwatchLogsReader) ListStreams(ctx context.Context) ([]types.LogStream, error) {
	_, err := getLogGroup(ctx, c.svc, c.logGroupName)
	if err != nil {
		return nil, err
	}
	return c.getLogStreams(ctx)
}

func (c *CloudwatchLogsReader) getLogStreams(ctx context.Context) ([]types.LogStream, error) {
	params := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: aws.String(c.logGroupName),
	}

	sortByTime := false
	if c.streamPrefix != "" {
		// If we are looking for a specific stream, search by prefix
		params.LogStreamNamePrefix = aws.String(c.streamPrefix)
	} else {
		// If not, just give us the most recently active
		params.OrderBy = types.OrderByLastEventTime
		params.Descending = aws.Bool(true)
		sortByTime = true
	}

	startTimestamp := c.start.Unix() * 1e3
	endTimestamp := time.Now().Unix() * 1e3
	if !c.end.IsZero() {
		endTimestamp = c.end.Unix() * 1e3
	}

	paginator := cloudwatchlogs.NewDescribeLogStreamsPaginator(c.svc, params)

	// Paginate through all of the log streams
	streams := []types.LogStream{}
	for paginator.HasMorePages() {
		if page, err := paginator.NextPage(ctx); err != nil {
			return streams, err
		} else {
			for _, s := range page.LogStreams {
				if len(streams) >= MaxStreams {
					return streams, nil
				}
				if s.LastIngestionTime == nil {
					// treat nil timestamps as 0
					s.LastIngestionTime = aws.Int64(0)
				}
				// if we are sorting by time, we can do some shortcuts to end
				// paging early if we are no longer in our time window
				if sortByTime {
					if s.CreationTime != nil && *s.CreationTime > endTimestamp {
						continue
					}
					if *s.LastIngestionTime < startTimestamp {
						return streams, nil
					}
					streams = append(streams, s)
				} else {
					// else we are searching by prefix, so we need to do a full
					if s.CreationTime != nil && *s.CreationTime < endTimestamp &&
						*s.LastIngestionTime > startTimestamp {
						streams = append(streams, s)
					}
				}
			}
		}
	}

	sort.Slice(streams[:], func(i, j int) bool { return *streams[i].LastIngestionTime > *streams[j].LastIngestionTime })
	if len(streams) == 0 {
		if c.streamPrefix != "" {
			return nil, fmt.Errorf("no log streams found matching task prefix '%s' in your time window.  Consider adjusting your time window with --since and/or --until", c.streamPrefix)
		}

		return nil, errors.New("no log streams found in your time window.  Consider adjusting your time window with --since and/or --until")

	}
	return streams, nil
}

// StreamEvents returns a channel where you can read events matching the params
// given in the readers constructor.  The channel will be closed once
// all events are read or an error occurs.  You can check for errors
// after the channel is closed by calling Error()
func (c *CloudwatchLogsReader) StreamEvents(ctx context.Context, follow bool) <-chan Event {
	eventChan := make(chan Event)
	go c.pumpEvents(ctx, eventChan, follow)

	return eventChan
}

func (c *CloudwatchLogsReader) pumpEvents(ctx context.Context, eventChan chan<- Event, follow bool) {

	startTime := c.start.Unix() * 1e3
	params := &cloudwatchlogs.FilterLogEventsInput{
		Interleaved:  aws.Bool(true),
		LogGroupName: aws.String(c.logGroupName),
		StartTime:    aws.Int64(startTime),
	}

	if !follow && c.end.IsZero() {
		c.end = time.Now()
	}

	if !c.end.IsZero() {
		endTime := c.end.Unix() * 1e3
		params.EndTime = aws.Int64(endTime)
	}

	if c.streamPrefix != "" {
		streams, err := c.getLogStreams(ctx)
		if err != nil {
			c.error = err
			close(eventChan)
			return
		}
		params.LogStreamNames = streamsToNames(streams)
	}

	for {
		paginator := cloudwatchlogs.NewFilterLogEventsPaginator(c.svc, params)
		for paginator.HasMorePages() {
			if page, err := paginator.NextPage(ctx); err != nil {
				c.error = err
				close(eventChan)
				return
			} else {
				for _, event := range page.Events {
					if _, ok := c.eventCache.Peek(*event.EventId); !ok {
						eventChan <- NewEvent(event, c.logGroupName)
						c.eventCache.Add(*event.EventId, nil)
					}
				}
			}
		}

		// If we are not following the logs, we are done
		if !follow {
			close(eventChan)
			return
		}

		// If we are following the logs, we need to wait for new events to appear
		time.Sleep(100 * time.Millisecond)
	}
}

// Error returns an error if one occurred while streaming events.
func (c *CloudwatchLogsReader) Error() error {
	return c.error
}

func streamsToNames(streams []types.LogStream) []string {
	names := make([]string, 0, len(streams))
	for _, s := range streams {
		names = append(names, *s.LogStreamName)
	}
	return names
}
