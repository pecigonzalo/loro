package lib

import (
	"testing"
	"time"
)

func TestGetTime(t *testing.T) {
	ref := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{"all", "all", time.Unix(0, 0), false},
		{"duration hours", "1h", ref.Add(-time.Hour), false},
		{"duration compound", "1h30m", ref.Add(-90 * time.Minute), false},
		{"zero is a unix timestamp, not a duration", "0", time.Unix(0, 0), false},
		{"unix timestamp", "1700000000", time.Unix(1700000000, 0), false},
		{"RFC3339 with Z", "2024-01-02T03:04:05Z", time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC), false},
		{"RFC3339 with offset", "2024-01-02T03:04:05+02:00", time.Date(2024, 1, 2, 1, 4, 5, 0, time.UTC), false},
		{"RFC3339 nano", "2024-01-02T03:04:05.123456789Z", time.Date(2024, 1, 2, 3, 4, 5, 123456789, time.UTC), false},
		{"local datetime", "2024-01-02T03:04:05", time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC), false},
		{"local minutes only", "2024-01-02T03:04", time.Date(2024, 1, 2, 3, 4, 0, 0, time.UTC), false},
		{"local hours only", "2024-01-02T03", time.Date(2024, 1, 2, 3, 0, 0, 0, time.UTC), false},
		{"local date only", "2024-01-02", time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), false},
		{"date with zone", "2024-01-02Z", time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), false},
		{"invalid", "not-a-time", time.Unix(0, 0), true},
		{"invalid non-numeric", "abc", time.Unix(0, 0), true},
		{"malformed date", "2024-13-45", time.Unix(0, 0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTime(tt.input, ref)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetTime(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !got.Equal(tt.want) {
				t.Errorf("GetTime(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetTimeNow(t *testing.T) {
	before := time.Now()
	got, err := GetTime("now", time.Time{})
	after := time.Now()
	if err != nil {
		t.Fatal(err)
	}
	if got.Before(before) || got.After(after) {
		t.Errorf("GetTime(now) = %v, want between %v and %v", got, before, after)
	}
}

func TestGetTimeLocalUsesReferenceZone(t *testing.T) {
	zone := time.FixedZone("TEST", 2*60*60)
	ref := time.Date(2024, 1, 1, 0, 0, 0, 0, zone)

	got, err := GetTime("2024-01-02T03:04:05", ref)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2024, 1, 2, 3, 4, 5, 0, zone)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseAWSTimestamp(t *testing.T) {
	ms := int64(1700000000123)

	tests := []struct {
		name string
		in   *int64
		want time.Time
	}{
		{"nil", nil, time.Unix(0, 0)},
		{"milliseconds", &ms, time.Unix(1700000000, 123*1e6)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseAWSTimestamp(tt.in); !got.Equal(tt.want) {
				t.Errorf("ParseAWSTimestamp() = %v, want %v", got, tt.want)
			}
		})
	}
}
