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

func TestStringMasksJSONAuthenticationFields(t *testing.T) {
	input := `{"password":"hunter2","mfa_token":"signed.challenge","code":"123456","safe":"visible"}`
	output := String(input)
	for _, secret := range []string{"hunter2", "signed.challenge", "123456"} {
		if strings.Contains(output, secret) {
			t.Fatalf("expected %q to be masked in %q", secret, output)
		}
	}
	if !strings.Contains(output, `"safe":"visible"`) {
		t.Fatalf("expected safe field to remain visible: %q", output)
	}
}
