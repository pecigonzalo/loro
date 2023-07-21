package lib

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

// Event represents a log event
type Event struct {
	Event        map[string]interface{}
	Stream       string
	Group        string
	ID           string
	IngestTime   time.Time
	CreationTime time.Time
}

// NewEvent takes a cloudwatch log event and returns an Event
func NewEvent(cwEvent types.FilteredLogEvent, group string) Event {
	var ecsLogsEvent map[string]interface{}
	if err := json.Unmarshal([]byte(*cwEvent.Message), &ecsLogsEvent); err != nil {
		ecsLogsEvent = make(map[string]interface{})
		ecsLogsEvent["message"] = *cwEvent.Message
	}

	return Event{
		Event:        ecsLogsEvent,
		Stream:       *cwEvent.LogStreamName,
		Group:        group,
		ID:           *cwEvent.EventId,
		IngestTime:   ParseAWSTimestamp(cwEvent.IngestionTime),
		CreationTime: ParseAWSTimestamp(cwEvent.Timestamp),
	}

}

// TimeShort gives the timestamp of an event in a readable format
func (e Event) TimeShort() string {
	return e.CreationTime.Local().Format(ShortTimeFormat)
}

// PrettyPrint returns a formatted json from the full event
func (e Event) PrettyPrint() string {
	pretty, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return fmt.Sprintf("%+v", e)
	}

	return string(pretty)
}
