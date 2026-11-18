package lib

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

func filteredEvent(message string) types.FilteredLogEvent {
	return types.FilteredLogEvent{
		Message:       aws.String(message),
		LogStreamName: aws.String("stream-1"),
		EventId:       aws.String("event-1"),
		IngestionTime: aws.Int64(1700000001000),
		Timestamp:     aws.Int64(1700000000000),
	}
}

func TestNewEventJSONMessage(t *testing.T) {
	e := NewEvent(filteredEvent(`{"level":"info","msg":"hello"}`), "group-1")

	if e.Stream != "stream-1" || e.Group != "group-1" || e.ID != "event-1" {
		t.Errorf("unexpected identity fields: %+v", e)
	}
	if e.Event["level"] != "info" || e.Event["msg"] != "hello" {
		t.Errorf("JSON message was not parsed: %+v", e.Event)
	}
	if !e.CreationTime.Equal(time.Unix(1700000000, 0)) {
		t.Errorf("CreationTime = %v", e.CreationTime)
	}
	if !e.IngestTime.Equal(time.Unix(1700000001, 0)) {
		t.Errorf("IngestTime = %v", e.IngestTime)
	}
}

func TestNewEventPlainMessage(t *testing.T) {
	e := NewEvent(filteredEvent("just some text"), "group-1")

	if len(e.Event) != 1 || e.Event["message"] != "just some text" {
		t.Errorf("plain message should be wrapped under \"message\": %+v", e.Event)
	}
}

func TestEventTimeShort(t *testing.T) {
	e := Event{CreationTime: time.Date(2024, 1, 2, 3, 4, 5, 0, time.Local)}
	if got, want := e.TimeShort(), "01-02 03:04:05"; got != want {
		t.Errorf("TimeShort() = %q, want %q", got, want)
	}
}

func TestEventPrettyPrint(t *testing.T) {
	e := NewEvent(filteredEvent(`{"a":1}`), "group-1")
	out := e.PrettyPrint()

	if !strings.Contains(out, "\n  ") {
		t.Errorf("output is not indented: %q", out)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if decoded["Stream"] != "stream-1" {
		t.Errorf("Stream = %v", decoded["Stream"])
	}
}

func TestEventPrettyPrintFallback(t *testing.T) {
	// A channel cannot be marshalled to JSON, forcing the fallback path.
	e := Event{Event: map[string]any{"bad": make(chan int)}, Stream: "s"}
	if out := e.PrettyPrint(); !strings.Contains(out, "Stream:s") {
		t.Errorf("fallback output = %q", out)
	}
}
