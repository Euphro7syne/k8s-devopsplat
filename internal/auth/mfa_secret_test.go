package auth

import (
	"strings"
	"testing"
)

func TestMFASecretCipherRoundTripAndLegacyPlaintext(t *testing.T) {
	cipher := NewMFASecretCipher("test-mfa-secret-key")

	const secret = "JBSWY3DPEHPK3PXP"
	encrypted, err := cipher.Encrypt(secret)
	if err != nil {
		t.Fatalf("encrypt secret: %v", err)
	}
	if encrypted == secret || !strings.HasPrefix(encrypted, encryptedMFASecretPrefix) {
		t.Fatalf("expected encrypted secret, got %q", encrypted)
	}
	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("decrypt secret: %v", err)
	}
	if decrypted != secret {
		t.Fatalf("expected %q, got %q", secret, decrypted)
	}

	legacy, err := cipher.Decrypt(secret)
	if err != nil {
		t.Fatalf("decrypt legacy secret: %v", err)
	}
	if legacy != secret {
		t.Fatalf("expected legacy plaintext compatibility")
	}
}
