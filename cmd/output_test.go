package cmd

import (
	"bytes"
	"testing"
)

func TestPrintTable(t *testing.T) {
	var buf bytes.Buffer
	if err := printTable(&buf, "Name\tValue", []string{"a\t1", "averylongcellname\t2"}); err != nil {
		t.Fatal(err)
	}

	// padding uses tabs: the widest cell (17 chars + 2) rounds up to 3 tab stops
	want := "Name\t\t\tValue\na\t\t\t1\naverylongcellname\t2\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
