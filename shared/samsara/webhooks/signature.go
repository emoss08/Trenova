package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"
)

const (
	HeaderSignature = "X-Samsara-Signature"
	HeaderTimestamp = "X-Samsara-Timestamp"

	signaturePrefix = "v1="
)

var (
	ErrSignatureSecretRequired = errors.New("webhook signature secret is required")
	ErrSignatureInvalid        = errors.New("webhook signature does not match")
	ErrSignatureFormatInvalid  = errors.New("webhook signature header format is invalid")
	ErrTimestampSkewExceeded   = errors.New("webhook timestamp skew exceeds tolerance")
)

func Sign(secret, timestamp string, body []byte) (string, error) {
	digest, err := computeDigest(secret, timestamp, body)
	if err != nil {
		return "", err
	}
	return signaturePrefix + hex.EncodeToString(digest), nil
}

func VerifySignature(secret, timestamp string, body []byte, signatureHeader string) error {
	provided, err := parseSignatureHeader(signatureHeader)
	if err != nil {
		return err
	}

	expected, err := computeDigest(secret, timestamp, body)
	if err != nil {
		return err
	}

	if !hmac.Equal(expected, provided) {
		return ErrSignatureInvalid
	}
	return nil
}

func VerifySignatureWithTolerance(
	secret string,
	timestamp string,
	body []byte,
	signatureHeader string,
	now time.Time,
	maxSkew time.Duration,
) error {
	ts, err := parseTimestamp(timestamp)
	if err != nil {
		return err
	}

	skew := now.Sub(ts)
	if skew < 0 {
		skew = -skew
	}
	if skew > maxSkew {
		return ErrTimestampSkewExceeded
	}

	return VerifySignature(secret, timestamp, body, signatureHeader)
}

func computeDigest(secret, timestamp string, body []byte) ([]byte, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, ErrSignatureSecretRequired
	}

	mac := hmac.New(sha256.New, secretKey(secret))
	mac.Write([]byte("v1:"))
	mac.Write([]byte(timestamp))
	mac.Write([]byte(":"))
	mac.Write(body)
	return mac.Sum(nil), nil
}

func secretKey(secret string) []byte {
	if decoded, err := base64.StdEncoding.DecodeString(secret); err == nil {
		return decoded
	}
	return []byte(secret)
}

func parseSignatureHeader(header string) ([]byte, error) {
	trimmed := strings.TrimSpace(header)
	if trimmed == "" {
		return nil, ErrSignatureFormatInvalid
	}

	entries := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})
	for _, entry := range entries {
		if !strings.HasPrefix(entry, signaturePrefix) {
			continue
		}
		digest, err := hex.DecodeString(strings.TrimPrefix(entry, signaturePrefix))
		if err != nil || len(digest) != sha256.Size {
			return nil, ErrSignatureFormatInvalid
		}
		return digest, nil
	}
	return nil, ErrSignatureFormatInvalid
}

func parseTimestamp(timestamp string) (time.Time, error) {
	trimmed := strings.TrimSpace(timestamp)
	if trimmed == "" {
		return time.Time{}, ErrSignatureFormatInvalid
	}

	if ts, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return ts, nil
	}
	if millis, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return time.UnixMilli(millis), nil
	}
	return time.Time{}, ErrSignatureFormatInvalid
}
