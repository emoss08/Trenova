// Package webhooksig verifies the signatures providers put on their webhooks.
//
// It exists because the check has to be the same everywhere. A second copy that
// drifts is not a second implementation, it is a second security posture, and
// the weaker one is the one an attacker picks.
package webhooksig

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrMissingHeaders = errors.New("the signature headers are missing")
	ErrBadTimestamp   = errors.New("the signature timestamp is not a unix time")
	ErrStale          = errors.New("the signature is outside the accepted time window")
	ErrNoMatch        = errors.New("the signature does not match the body")
)

// DefaultMaxSkew is how far out a delivery's timestamp may be.
//
// Five minutes each way covers ordinary clock drift and a provider's own retry
// latency. It is what bounds a replay: without it, a body captured once is
// valid for as long as the secret lives.
const DefaultMaxSkew = 5 * time.Minute

// SvixParams is one delivery to check. Svix is the signing scheme Resend uses,
// and several other providers besides.
type SvixParams struct {
	// Secret is the endpoint's signing secret, with or without its `whsec_`
	// prefix.
	Secret string
	// ID, Timestamp and Signature come from the svix-id, svix-timestamp and
	// svix-signature headers.
	ID        string
	Timestamp string
	Signature string
	Body      []byte
	// Now and MaxSkew are injectable so the window itself can be tested rather
	// than taken on trust. Zero values mean the real clock and DefaultMaxSkew.
	Now     time.Time
	MaxSkew time.Duration
}

// VerifySvix checks the signature and the age of a delivery.
//
// The timestamp check is not decoration. The signature covers the timestamp, so
// an attacker cannot move it — which is exactly what makes it worth reading: it
// is the one part of a captured delivery that cannot be replayed forever.
func VerifySvix(p SvixParams) error {
	if p.Secret == "" || p.ID == "" || p.Timestamp == "" || p.Signature == "" {
		return ErrMissingHeaders
	}

	if err := checkSkew(p); err != nil {
		return err
	}

	expected, err := svixMAC(p.Secret, p.ID, p.Timestamp, p.Body)
	if err != nil {
		return err
	}

	// A delivery carries every signature still in rotation, space separated, so
	// a secret roll does not drop messages. One match is enough.
	for part := range strings.SplitSeq(p.Signature, " ") {
		_, sig, ok := strings.Cut(part, ",")
		if !ok {
			sig = strings.TrimPrefix(part, "v1,")
		}
		decoded, decodeErr := base64.StdEncoding.DecodeString(sig)
		if decodeErr == nil && hmac.Equal(decoded, expected) {
			return nil
		}
	}

	return ErrNoMatch
}

// SignSvix is the svix-signature header value for a delivery: what a provider
// sends, and what a test or a developer sends to exercise an endpoint.
func SignSvix(secret, id, timestamp string, body []byte) (string, error) {
	mac, err := svixMAC(secret, id, timestamp, body)
	if err != nil {
		return "", err
	}

	return "v1," + base64.StdEncoding.EncodeToString(mac), nil
}

func svixMAC(secret, id, timestamp string, body []byte) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
	if err != nil {
		return nil, fmt.Errorf("the signing secret is not valid base64: %w", err)
	}

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id + "." + timestamp + "."))
	mac.Write(body)

	return mac.Sum(nil), nil
}

func checkSkew(p SvixParams) error {
	seconds, err := strconv.ParseInt(p.Timestamp, 10, 64)
	if err != nil {
		return ErrBadTimestamp
	}

	now := p.Now
	if now.IsZero() {
		now = time.Now()
	}
	skew := p.MaxSkew
	if skew <= 0 {
		skew = DefaultMaxSkew
	}

	// Checked in both directions: a delivery from the future is as much a sign
	// of a forged timestamp as one from last week is of a replay.
	drift := now.Sub(time.Unix(seconds, 0))
	if drift < 0 {
		drift = -drift
	}
	if drift > skew {
		return fmt.Errorf("%w: %s out", ErrStale, drift.Round(time.Second))
	}

	return nil
}
