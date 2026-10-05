package totputils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // RFC 6238 authenticator apps derive codes with HMAC-SHA1
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	Digits      = 6
	Period      = 30 * time.Second
	DefaultSkew = 1
	secretBytes = 20
	digitsMod   = 1_000_000
)

var (
	ErrInvalidSecret = errors.New("totp secret is not valid base32")
	ErrInvalidCode   = errors.New("totp code is not valid")

	encoding = base32.StdEncoding.WithPadding(base32.NoPadding)
)

func GenerateSecret() (string, error) {
	raw := make([]byte, secretBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate totp secret: %w", err)
	}

	return encoding.EncodeToString(raw), nil
}

func Step(at time.Time) int64 {
	return at.Unix() / int64(Period.Seconds())
}

func CodeAt(secret string, step int64) (string, error) {
	key, err := decodeSecret(secret)
	if err != nil {
		return "", err
	}

	return codeFor(key, step), nil
}

type VerifyRequest struct {
	Secret string
	Code   string
	At     time.Time
	Skew   int
	After  int64
}

func Verify(req VerifyRequest) (int64, error) {
	code := NormalizeCode(req.Code)
	if len(code) != Digits {
		return 0, ErrInvalidCode
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return 0, ErrInvalidCode
		}
	}

	key, err := decodeSecret(req.Secret)
	if err != nil {
		return 0, err
	}

	current := Step(req.At)
	skew := int64(max(req.Skew, 0))
	matched := int64(-1)
	for offset := -skew; offset <= skew; offset++ {
		step := current + offset
		if step <= req.After {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(codeFor(key, step)), []byte(code)) == 1 {
			matched = step
		}
	}

	if matched < 0 {
		return 0, ErrInvalidCode
	}

	return matched, nil
}

func NormalizeCode(code string) string {
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' {
			return -1
		}
		return r
	}, strings.TrimSpace(code))
}

type URIParams struct {
	Issuer  string
	Account string
	Secret  string
}

func URI(p URIParams) string {
	label := url.PathEscape(p.Issuer) + ":" + url.PathEscape(p.Account)
	query := url.Values{}
	query.Set("secret", p.Secret)
	query.Set("issuer", p.Issuer)
	query.Set("algorithm", "SHA1")
	query.Set("digits", strconv.Itoa(Digits))
	query.Set("period", strconv.Itoa(int(Period.Seconds())))

	return "otpauth://totp/" + label + "?" + query.Encode()
}

func decodeSecret(secret string) ([]byte, error) {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(secret), " ", ""))
	normalized = strings.TrimRight(normalized, "=")
	key, err := encoding.DecodeString(normalized)
	if err != nil || len(key) == 0 {
		return nil, ErrInvalidSecret
	}

	return key, nil
}

func codeFor(key []byte, step int64) string {
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(step)) //nolint:gosec // steps are positive Unix intervals

	mac := hmac.New(sha1.New, key)
	mac.Write(counter[:])
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff

	return fmt.Sprintf("%0*d", Digits, value%digitsMod)
}
