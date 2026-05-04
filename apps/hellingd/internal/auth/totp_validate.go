package auth

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // RFC 6238 TOTP requires HMAC-SHA1 compatibility.
	"encoding/binary"
	"fmt"
	"time"
)

// ValidateTOTP checks a 6-digit TOTP code against a base32-encoded secret.
// Uses RFC 6238 with SHA-1, 30-second period, 6 digits. Allows ±1 period
// drift to account for clock skew.
func ValidateTOTP(secret, code string) bool {
	if len(code) != 6 {
		return false
	}
	now := time.Now().UTC().Unix()
	// Check current period and ±1 adjacent periods.
	for _, offset := range []int64{0, -30, 30} {
		if totpAt(secret, now+offset) == code {
			return true
		}
	}
	return false
}

// totpAt computes the TOTP value at a specific Unix timestamp.
func totpAt(secret string, t int64) string {
	if t < 0 {
		return ""
	}
	counter := uint64(t / 30) //nolint:gosec // t is rejected above when negative.

	// Decode base32 secret (RFC 4648, no padding).
	key := decodeBase32(secret)
	if len(key) == 0 {
		return ""
	}

	// HMAC-SHA1.
	mac := hmac.New(sha1.New, key)
	if err := binary.Write(mac, binary.BigEndian, counter); err != nil {
		return ""
	}
	sum := mac.Sum(nil)

	// Dynamic truncation per RFC 4226 §5.4.
	offset := sum[len(sum)-1] & 0x0f
	binary := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	otp := binary % 1_000_000

	return fmt.Sprintf("%06d", otp)
}

// decodeBase32 decodes a base32 string with optional padding. Returns nil on
// any decode error (caller treats as invalid secret).
func decodeBase32(s string) []byte {
	// Normalize: uppercase, strip padding.
	normalized := ""
	for _, c := range s {
		if c >= 'a' && c <= 'z' {
			normalized += string(c - 32)
		} else if c >= 'A' && c <= 'Z' || c >= '2' && c <= '7' {
			normalized += string(c)
		}
	}
	// Pad to multiple of 8.
	for len(normalized)%8 != 0 {
		normalized += "="
	}

	// Decode using a simple lookup table.
	alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	lookup := make(map[byte]byte)
	for i := 0; i < len(alphabet); i++ {
		lookup[alphabet[i]] = byte(i)
	}

	var buf []byte
	var bits uint16
	var bitsLen uint8
	for i := 0; i < len(normalized); i++ {
		if normalized[i] == '=' {
			break
		}
		v, ok := lookup[normalized[i]]
		if !ok {
			return nil
		}
		bits = (bits << 5) | uint16(v)
		bitsLen += 5
		if bitsLen >= 8 {
			bitsLen -= 8
			buf = append(buf, byte(bits>>bitsLen))
			bits &= (1 << bitsLen) - 1
		}
	}
	return buf
}
