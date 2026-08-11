package logquery

import (
	"strings"
	"testing"
)

func TestReadLinesFiltersKeywordAndLevel(t *testing.T) {
	raw := strings.NewReader("info boot\nerror database token=secret\nwarn cache\n")
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
}
