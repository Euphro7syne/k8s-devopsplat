package auth

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateTOTPRFC6238Vector(t *testing.T) {
	const secret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	code, err := generateTOTP(secret, 1, 8)
	if err != nil {
		t.Fatalf("generate totp: %v", err)
	}
	if code != "94287082" {
		t.Fatalf("expected RFC 6238 code 94287082, got %s", code)
	}
}

func TestTOTPValidateAndProvisioningURI(t *testing.T) {
	fixed := time.Unix(1_700_000_000, 0).UTC()
	totp := NewTOTP("ops-platform")
	totp.nowFunc = func() time.Time { return fixed }

	secret, err := totp.GenerateSecret()
	if err != nil {
		t.Fatalf("generate secret: %v", err)
	}
	code, err := generateTOTP(secret, uint64(fixed.Unix()/totpPeriod), totpDigits)
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}
	if !totp.Validate(secret, code) {
		t.Fatalf("expected generated code to validate")
	}
	if totp.Validate(secret, "00000x") {
		t.Fatalf("expected non-numeric code to fail")
	}

	uri := totp.ProvisioningURI("admin@example.com", secret)
	for _, expected := range []string{"otpauth://totp/", "admin@example.com", "issuer=ops-platform", "secret=" + secret} {
		if !strings.Contains(uri, expected) {
			t.Fatalf("expected provisioning URI %q to contain %q", uri, expected)
		}
	}
}
