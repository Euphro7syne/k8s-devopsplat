package logquery

import (
	"context"
	"strings"
	"testing"
)

func TestReadLinesFiltersKeywordAndLevel(t *testing.T) {
	raw := strings.NewReader("info boot\nerror database token=plain-value\nwarn cache\n")
	lines, err := readLines(raw, "database", "error")
	if err != nil {
		t.Fatalf("read lines: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected one line, got %d", len(lines))
	}
	if !strings.Contains(lines[0].Raw, "database") {
		t.Fatalf("unexpected line: %s", lines[0].Raw)
	}
	if strings.Contains(lines[0].Raw, "plain-value") || !strings.Contains(lines[0].Raw, "token=***") {
		t.Fatalf("expected sensitive log value to be masked: %s", lines[0].Raw)
	}
}

func TestStreamLinesFiltersAndSanitizesRealtimeOutput(t *testing.T) {
	raw := strings.NewReader("info boot\nerror authorization=Bearer-real-token\nwarn cache\n")
	var received []Line
	err := streamLines(context.Background(), raw, "authorization", "error", func(line Line) error {
		received = append(received, line)
		return nil
	})
	if err != nil {
		t.Fatalf("stream lines: %v", err)
	}
	if len(received) != 1 {
		t.Fatalf("expected one realtime line, got %d", len(received))
	}
	if strings.Contains(received[0].Raw, "real-token") || !strings.Contains(received[0].Raw, "authorization=***") {
		t.Fatalf("expected realtime sensitive value to be masked: %s", received[0].Raw)
	}
}
