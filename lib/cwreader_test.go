package lib

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	lru "github.com/hashicorp/golang-lru/v2"
)

// fakeCW serves canned pages. A page's NextToken is its index in the slice,
// so page N+1 is requested with token "N+1".
type fakeCW struct {
	mu sync.Mutex

	groupPages  [][]types.LogGroup
	streamPages [][]types.LogStream
	eventPages  [][]types.FilteredLogEvent
	err         error // returned by every call when set

	streamInputs []*cloudwatchlogs.DescribeLogStreamsInput
	eventInputs  []*cloudwatchlogs.FilterLogEventsInput
}

// page returns the page selected by token (nil means the first page) and the
// token for the following page, or nil after the last one.
func page[T any](pages [][]T, token *string) ([]T, *string) {
	i := 0
	if token != nil {
		i, _ = strconv.Atoi(*token)
	}
	var next *string
	if i+1 < len(pages) {
		next = aws.String(strconv.Itoa(i + 1))
	}
	return pages[i], next
}

func (f *fakeCW) DescribeLogGroups(ctx context.Context, in *cloudwatchlogs.DescribeLogGroupsInput, _ ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	groups, next := page(f.groupPages, in.NextToken)
	return &cloudwatchlogs.DescribeLogGroupsOutput{LogGroups: groups, NextToken: next}, nil
}

func (f *fakeCW) DescribeLogStreams(ctx context.Context, in *cloudwatchlogs.DescribeLogStreamsInput, _ ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogStreamsOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.streamInputs = append(f.streamInputs, in)
	if f.err != nil {
		return nil, f.err
	}
	streams, next := page(f.streamPages, in.NextToken)
	return &cloudwatchlogs.DescribeLogStreamsOutput{LogStreams: streams, NextToken: next}, nil
}

func (f *fakeCW) FilterLogEvents(ctx context.Context, in *cloudwatchlogs.FilterLogEventsInput, _ ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.FilterLogEventsOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *in
	f.eventInputs = append(f.eventInputs, &cp)
	if f.err != nil {
		return nil, f.err
	}
	events, next := page(f.eventPages, in.NextToken)
	return &cloudwatchlogs.FilterLogEventsOutput{Events: events, NextToken: next}, nil
}

func newTestReader(t *testing.T, svc cloudWatchLogsAPI, group, prefix string, start, end time.Time) *CloudwatchLogsReader {
	t.Helper()
	cache, err := lru.New[string, any](100)
	if err != nil {
		t.Fatal(err)
	}
	return &CloudwatchLogsReader{
		logGroupName: group,
		svc:          svc,
		eventCache:   cache,
		start:        start,
		end:          end,
		streamPrefix: prefix,
	}
}

func lg(name string) types.LogGroup { return types.LogGroup{LogGroupName: aws.String(name)} }

func ls(name string, created, lastIngest int64) types.LogStream {
	s := types.LogStream{LogStreamName: aws.String(name), CreationTime: aws.Int64(created)}
	if lastIngest >= 0 {
		s.LastIngestionTime = aws.Int64(lastIngest)
	}
	return s
}

func ev(id, stream, msg string) types.FilteredLogEvent {
	e := filteredEvent(msg)
	e.EventId = aws.String(id)
	e.LogStreamName = aws.String(stream)
	return e
}

func TestListGroupsPaginates(t *testing.T) {
	svc := &fakeCW{groupPages: [][]types.LogGroup{{lg("a"), lg("b")}, {lg("c")}}}
	r := newTestReader(t, svc, "", "", time.Time{}, time.Time{})

	groups, err := r.ListGroups(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 3 || *groups[2].LogGroupName != "c" {
		t.Errorf("groups = %v", groups)
	}
}

func TestListGroupsError(t *testing.T) {
	boom := errors.New("boom")
	r := newTestReader(t, &fakeCW{err: boom}, "", "", time.Time{}, time.Time{})

	if _, err := r.ListGroups(context.Background()); !errors.Is(err, boom) {
		t.Errorf("err = %v, want %v", err, boom)
	}
}

func TestGetGroup(t *testing.T) {
	tests := []struct {
		name       string
		pages      [][]types.LogGroup
		lookup     string
		wantErr    []string // substrings; nil means success
		notInError []string
	}{
		{name: "exact match", pages: [][]types.LogGroup{{lg("app"), lg("app-2")}}, lookup: "app"},
		{name: "not found", pages: [][]types.LogGroup{{}}, lookup: "app", wantErr: []string{"could not find log group 'app'"}},
		{
			name:    "no exact match suggests alternatives",
			pages:   [][]types.LogGroup{{lg("app-1"), lg("app-2")}},
			lookup:  "app",
			wantErr: []string{"Did you mean", "app-1", "app-2"},
		},
		{
			name:       "suggestions are capped at five",
			pages:      [][]types.LogGroup{{lg("x1"), lg("x2"), lg("x3"), lg("x4"), lg("x5"), lg("x6"), lg("x7")}},
			lookup:     "x",
			wantErr:    []string{"x5"},
			notInError: []string{"x6", "x7"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTestReader(t, &fakeCW{groupPages: tt.pages}, tt.lookup, "", time.Time{}, time.Time{})
			g, err := r.GetGroup(context.Background())

			if tt.wantErr == nil {
				if err != nil {
					t.Fatal(err)
				}
				if *g.LogGroupName != tt.lookup {
					t.Errorf("group = %q", *g.LogGroupName)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			for _, s := range tt.wantErr {
				if !strings.Contains(err.Error(), s) {
					t.Errorf("error %q missing %q", err, s)
				}
			}
			for _, s := range tt.notInError {
				if strings.Contains(err.Error(), s) {
					t.Errorf("error %q should not contain %q", err, s)
				}
			}
		})
	}
}

// Window used by the stream tests: 1000s .. 2000s, in milliseconds below.
var (
	winStart = time.Unix(1000, 0)
	winEnd   = time.Unix(2000, 0)
)

func TestListStreamsByTime(t *testing.T) {
	svc := &fakeCW{
		groupPages: [][]types.LogGroup{{lg("g")}},
		streamPages: [][]types.LogStream{
			{
				ls("future", 2_500_000, 3_000_000), // created after the window: skipped
				ls("recent", 500_000, 1_900_000),
				ls("older", 500_000, 1_200_000),
			},
			{
				ls("stale", 100_000, 500_000), // before the window: stop paging here
				ls("after-stale", 100_000, 1_500_000),
			},
		},
	}
	r := newTestReader(t, svc, "g", "", winStart, winEnd)

	streams, err := r.ListStreams(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := streamsToNames(streams), []string{"recent", "older"}; !slices.Equal(got, want) {
		t.Errorf("streams = %v, want %v", got, want)
	}

	in := svc.streamInputs[0]
	if in.OrderBy != types.OrderByLastEventTime || !aws.ToBool(in.Descending) || in.LogStreamNamePrefix != nil {
		t.Errorf("expected most-recent-first ordering without prefix, got %+v", in)
	}
}

func TestListStreamsByPrefix(t *testing.T) {
	svc := &fakeCW{
		groupPages: [][]types.LogGroup{{lg("g")}},
		streamPages: [][]types.LogStream{{
			ls("web-old", 100_000, 500_000), // ingested before the window
			ls("web-in", 100_000, 1_500_000),
			ls("web-later", 100_000, 1_800_000),
			ls("web-future", 2_500_000, 3_000_000), // created after the window
		}},
	}
	r := newTestReader(t, svc, "g", "web", winStart, winEnd)

	streams, err := r.ListStreams(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// results are sorted by last ingestion time, newest first
	if got, want := streamsToNames(streams), []string{"web-later", "web-in"}; !slices.Equal(got, want) {
		t.Errorf("streams = %v, want %v", got, want)
	}
	if in := svc.streamInputs[0]; aws.ToString(in.LogStreamNamePrefix) != "web" || in.OrderBy != "" {
		t.Errorf("prefix search should send the prefix and no ordering, got %+v", in)
	}
}

func TestListStreamsNilIngestionTimeTreatedAsZero(t *testing.T) {
	svc := &fakeCW{
		groupPages:  [][]types.LogGroup{{lg("g")}},
		streamPages: [][]types.LogStream{{ls("with-events", 100_000, 1_500_000), ls("empty", 100_000, -1)}},
	}
	// Zero start so the empty (time 0) stream is still inside the window.
	r := newTestReader(t, svc, "g", "", time.Unix(0, 0), winEnd)

	streams, err := r.ListStreams(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := streamsToNames(streams), []string{"with-events", "empty"}; !slices.Equal(got, want) {
		t.Errorf("streams = %v, want %v", got, want)
	}
}

func TestListStreamsRespectsMaxStreams(t *testing.T) {
	prev := MaxStreams
	SetMaxStreams(2)
	t.Cleanup(func() { SetMaxStreams(prev) })

	svc := &fakeCW{
		groupPages: [][]types.LogGroup{{lg("g")}},
		streamPages: [][]types.LogStream{{
			ls("s1", 100_000, 1_900_000), ls("s2", 100_000, 1_800_000), ls("s3", 100_000, 1_700_000),
		}},
	}
	r := newTestReader(t, svc, "g", "", winStart, winEnd)

	streams, err := r.ListStreams(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 2 {
		t.Errorf("got %d streams, want 2", len(streams))
	}
}

func TestListStreamsNoneFound(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		want   string
	}{
		{"no prefix", "", "no log streams found in your time window"},
		{"with prefix", "web", "matching task prefix 'web'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeCW{groupPages: [][]types.LogGroup{{lg("g")}}, streamPages: [][]types.LogStream{{}}}
			r := newTestReader(t, svc, "g", tt.prefix, winStart, winEnd)

			_, err := r.ListStreams(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestListStreamsGroupMissing(t *testing.T) {
	svc := &fakeCW{groupPages: [][]types.LogGroup{{}}}
	r := newTestReader(t, svc, "g", "", winStart, winEnd)

	if _, err := r.ListStreams(context.Background()); err == nil {
		t.Fatal("expected an error for a missing group")
	}
	if len(svc.streamInputs) != 0 {
		t.Error("streams should not be queried when the group does not exist")
	}
}

func drain(ch <-chan Event) []Event {
	var out []Event
	for e := range ch {
		out = append(out, e)
	}
	return out
}

func TestStreamEventsPaginatesAndDeduplicates(t *testing.T) {
	svc := &fakeCW{eventPages: [][]types.FilteredLogEvent{
		{ev("1", "s", "one"), ev("2", "s", "two")},
		{ev("2", "s", "two again"), ev("3", "s", "three")}, // id 2 already seen
	}}
	r := newTestReader(t, svc, "g", "", winStart, time.Time{})

	events := drain(r.StreamEvents(context.Background(), false))

	if r.Error() != nil {
		t.Fatalf("unexpected error: %v", r.Error())
	}
	var ids []string
	for _, e := range events {
		ids = append(ids, e.ID)
	}
	if want := []string{"1", "2", "3"}; !slices.Equal(ids, want) {
		t.Errorf("event ids = %v, want %v", ids, want)
	}
	if events[0].Group != "g" || events[0].Event["message"] != "one" {
		t.Errorf("first event = %+v", events[0])
	}
}

func TestStreamEventsRequestParams(t *testing.T) {
	t.Run("not following sets an end time", func(t *testing.T) {
		svc := &fakeCW{eventPages: [][]types.FilteredLogEvent{{}}}
		r := newTestReader(t, svc, "g", "", winStart, time.Time{})

		drain(r.StreamEvents(context.Background(), false))

		in := svc.eventInputs[0]
		if aws.ToInt64(in.StartTime) != 1_000_000 || in.EndTime == nil {
			t.Errorf("StartTime=%v EndTime=%v", in.StartTime, in.EndTime)
		}
		if aws.ToString(in.LogGroupName) != "g" {
			t.Errorf("LogGroupName = %v", in.LogGroupName)
		}
	})

	t.Run("explicit end time is used", func(t *testing.T) {
		svc := &fakeCW{eventPages: [][]types.FilteredLogEvent{{}}}
		r := newTestReader(t, svc, "g", "", winStart, winEnd)

		drain(r.StreamEvents(context.Background(), false))

		if got := aws.ToInt64(svc.eventInputs[0].EndTime); got != 2_000_000 {
			t.Errorf("EndTime = %d, want 2000000", got)
		}
	})
}

func TestStreamEventsResolvesStreamPrefix(t *testing.T) {
	svc := &fakeCW{
		streamPages: [][]types.LogStream{{ls("web-1", 100_000, 1_500_000), ls("web-2", 100_000, 1_600_000)}},
		eventPages:  [][]types.FilteredLogEvent{{ev("1", "web-2", "hi")}},
	}
	r := newTestReader(t, svc, "g", "web", winStart, winEnd)

	events := drain(r.StreamEvents(context.Background(), false))
	if len(events) != 1 || r.Error() != nil {
		t.Fatalf("events=%d err=%v", len(events), r.Error())
	}
	if got, want := svc.eventInputs[0].LogStreamNames, []string{"web-2", "web-1"}; !slices.Equal(got, want) {
		t.Errorf("LogStreamNames = %v, want %v", got, want)
	}
}

func TestStreamEventsPrefixWithNoStreamsReportsError(t *testing.T) {
	svc := &fakeCW{streamPages: [][]types.LogStream{{}}}
	r := newTestReader(t, svc, "g", "web", winStart, winEnd)

	events := drain(r.StreamEvents(context.Background(), false))
	if len(events) != 0 {
		t.Errorf("expected no events, got %d", len(events))
	}
	if r.Error() == nil || !strings.Contains(r.Error().Error(), "matching task prefix") {
		t.Errorf("Error() = %v", r.Error())
	}
	if len(svc.eventInputs) != 0 {
		t.Error("events should not be queried when no streams matched")
	}
}

func TestStreamEventsAPIErrorIsReported(t *testing.T) {
	boom := errors.New("throttled")
	r := newTestReader(t, &fakeCW{err: boom}, "g", "", winStart, time.Time{})

	drain(r.StreamEvents(context.Background(), false))

	if !errors.Is(r.Error(), boom) {
		t.Errorf("Error() = %v, want %v", r.Error(), boom)
	}
}

func TestStreamEventsFollowStopsOnCancel(t *testing.T) {
	svc := &fakeCW{eventPages: [][]types.FilteredLogEvent{{ev("1", "s", "one")}}}
	r := newTestReader(t, svc, "g", "", winStart, time.Time{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := r.StreamEvents(ctx, true)

	select {
	case e := <-ch:
		if e.ID != "1" {
			t.Errorf("first event = %+v", e)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the first event")
	}

	cancel()

	done := make(chan struct{})
	go func() { drain(ch); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("channel was not closed after cancellation")
	}
	if !errors.Is(r.Error(), context.Canceled) {
		t.Errorf("Error() = %v, want context.Canceled", r.Error())
	}
}

func TestStreamsToNames(t *testing.T) {
	got := streamsToNames([]types.LogStream{ls("a", 0, 0), ls("b", 0, 0)})
	if !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("names = %v", got)
	}
}
