package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"ops-platform/internal/config"
)

type rateLimitPolicy struct {
	maxAttempts   int
	window        time.Duration
	blockDuration time.Duration
}

type rateLimitEntry struct {
	failures     []time.Time
	blockedUntil time.Time
	expiresAt    time.Time
}

type authRateLimiter struct {
	enabled bool
	login   rateLimitPolicy
	mfa     rateLimitPolicy

	mu      sync.Mutex
	entries map[string]*rateLimitEntry
	nowFunc func() time.Time
}

func newAuthRateLimiter(cfg config.AuthRateLimitConfig) *authRateLimiter {
	cfg = cfg.WithDefaults()
	return &authRateLimiter{
		enabled: cfg.Enabled,
		login: rateLimitPolicy{
			maxAttempts:   cfg.LoginMaxAttempts,
			window:        cfg.LoginWindow,
			blockDuration: cfg.LoginBlockDuration,
		},
		mfa: rateLimitPolicy{
			maxAttempts:   cfg.MFAMaxAttempts,
			window:        cfg.MFAWindow,
			blockDuration: cfg.MFABlockDuration,
		},
		entries: make(map[string]*rateLimitEntry),
		nowFunc: time.Now,
	}
}

func (l *authRateLimiter) loginKeys(sourceIP, email string) []string {
	return []string{
		"login:ip:" + rateLimitDigest(normalizeRateLimitIdentity(sourceIP)),
		"login:account:" + rateLimitDigest(strings.ToLower(strings.TrimSpace(email))),
	}
}

func (l *authRateLimiter) mfaKeys(sourceIP, mfaToken string) []string {
	return []string{
		"mfa:ip:" + rateLimitDigest(normalizeRateLimitIdentity(sourceIP)),
		"mfa:challenge:" + rateLimitDigest(strings.TrimSpace(mfaToken)),
	}
}

func (l *authRateLimiter) blocked(keys []string) bool {
	if l == nil || !l.enabled {
		return false
	}
	now := l.nowFunc().UTC()
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanupLocked(now)
	for _, key := range uniqueRateLimitKeys(keys) {
		entry := l.entries[key]
		if entry != nil && entry.blockedUntil.After(now) {
			return true
		}
	}
	return false
}

func (l *authRateLimiter) recordFailure(keys []string, policy rateLimitPolicy) bool {
	if l == nil || !l.enabled {
		return false
	}
	now := l.nowFunc().UTC()
	cutoff := now.Add(-policy.window)
	blocked := false

	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanupLocked(now)
	for _, key := range uniqueRateLimitKeys(keys) {
		entry := l.entries[key]
		if entry == nil {
			entry = &rateLimitEntry{}
			l.entries[key] = entry
		}
		if entry.blockedUntil.After(now) {
			blocked = true
			continue
		}

		entry.failures = pruneRateLimitFailures(entry.failures, cutoff)
		entry.failures = append(entry.failures, now)
		entry.expiresAt = now.Add(policy.window)
		if len(entry.failures) >= policy.maxAttempts {
			entry.failures = nil
			entry.blockedUntil = now.Add(policy.blockDuration)
			entry.expiresAt = entry.blockedUntil
			blocked = true
		}
	}
	return blocked
}

func (l *authRateLimiter) reset(keys []string) {
	if l == nil || !l.enabled {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, key := range uniqueRateLimitKeys(keys) {
		delete(l.entries, key)
	}
}

func (l *authRateLimiter) cleanupLocked(now time.Time) {
	for key, entry := range l.entries {
		if !entry.expiresAt.After(now) {
			delete(l.entries, key)
		}
	}
}

func pruneRateLimitFailures(failures []time.Time, cutoff time.Time) []time.Time {
	first := 0
	for first < len(failures) && failures[first].Before(cutoff) {
		first++
	}
	if first == 0 {
		return failures
	}
	return append([]time.Time(nil), failures[first:]...)
}

func uniqueRateLimitKeys(keys []string) []string {
	seen := make(map[string]struct{}, len(keys))
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	return result
}

func normalizeRateLimitIdentity(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unknown"
	}
	return value
}

func rateLimitDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
