package sanitizer

import (
	"strings"
	"testing"
)

func TestStringMasksSensitiveValues(t *testing.T) {
	input := `password=abc token: "def" api_key='ghi' normal=value`
	output := String(input)
	for _, value := range []string{"abc", "def", "ghi"} {
		if strings.Contains(output, value) {
			t.Fatalf("expected %q to be masked in %q", value, output)
		}
	}
	if !strings.Contains(output, "normal=value") {
		t.Fatalf("expected non-sensitive value to remain, got %q", output)
	}
}
