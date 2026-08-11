package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	totpDigits       = 6
	totpPeriod       = int64(30)
	totpAllowedSkew  = int64(1)
	totpSecretLength = 20
)

type TOTP struct {
	issuer  string
	nowFunc func() time.Time
}

func NewTOTP(issuer string) *TOTP {
	return &TOTP{
		issuer:  strings.TrimSpace(issuer),
		nowFunc: time.Now,
	}
}

func (t *TOTP) GenerateSecret() (string, error) {
	raw := make([]byte, totpSecretLength)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate totp secret: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw), nil
}

func (t *TOTP) ProvisioningURI(account, secret string) string {
	issuer := t.issuer
	if issuer == "" {
		issuer = "ops-platform"
	}
	account = strings.TrimSpace(account)
	uri := url.URL{
		Scheme: "otpauth",
		Host:   "totp",
		Path:   "/" + issuer + ":" + account,
	}
	query := uri.Query()
	query.Set("secret", normalizeTOTPSecret(secret))
	query.Set("issuer", issuer)
	query.Set("algorithm", "SHA1")
	query.Set("digits", strconv.Itoa(totpDigits))
	query.Set("period", strconv.FormatInt(totpPeriod, 10))
	uri.RawQuery = query.Encode()
	return uri.String()
}

func (t *TOTP) Validate(secret, code string) bool {
	code = strings.TrimSpace(code)
	if len(code) != totpDigits {
		return false
	}
	for _, digit := range code {
		if digit < '0' || digit > '9' {
			return false
		}
	}

	now := t.nowFunc().UTC().Unix() / totpPeriod
	for offset := -totpAllowedSkew; offset <= totpAllowedSkew; offset++ {
		expected, err := generateTOTP(secret, uint64(now+offset), totpDigits)
		if err != nil {
			return false
		}
		if subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
			return true
		}
	}
	return false
}

func generateTOTP(secret string, counter uint64, digits int) (string, error) {
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(normalizeTOTPSecret(secret))
	if err != nil || len(decoded) == 0 {
		return "", fmt.Errorf("invalid totp secret")
	}

	var counterBytes [8]byte
	binary.BigEndian.PutUint64(counterBytes[:], counter)
	mac := hmac.New(sha1.New, decoded)
	_, _ = mac.Write(counterBytes[:])
	digest := mac.Sum(nil)
	offset := digest[len(digest)-1] & 0x0f
	binaryCode := binary.BigEndian.Uint32(digest[offset:offset+4]) & 0x7fffffff

	modulus := uint32(1)
	for i := 0; i < digits; i++ {
		modulus *= 10
	}
	return fmt.Sprintf("%0*d", digits, binaryCode%modulus), nil
}

func normalizeTOTPSecret(secret string) string {
	secret = strings.ToUpper(strings.TrimSpace(secret))
	secret = strings.ReplaceAll(secret, " ", "")
	secret = strings.TrimRight(secret, "=")
	return secret
}
