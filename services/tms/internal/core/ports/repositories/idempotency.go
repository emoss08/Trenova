package repositories

import (
	"context"
	"errors"
	"time"
)

var ErrIdempotencyClaimLost = errors.New("idempotency claim is no longer held by this request")

type IdempotencyState string

const (
	IdempotencyStateProcessing = IdempotencyState("processing")
	IdempotencyStateCompleted  = IdempotencyState("completed")
)

type IdempotencyRecord struct {
	State       IdempotencyState
	Fingerprint string
	Status      int
	ContentType string
	Body        []byte
	BodyOmitted bool
}

type IdempotencyClaim struct {
	Key         string
	Fingerprint string
	Owner       string
	LockTTL     time.Duration
}

type IdempotencyCompletion struct {
	Key         string
	Owner       string
	Status      int
	ContentType string
	Body        []byte
	BodyOmitted bool
	TTL         time.Duration
}

type IdempotencyClaimResult struct {
	Claimed  bool
	Existing *IdempotencyRecord
}

type IdempotencyStore interface {
	Claim(ctx context.Context, claim IdempotencyClaim) (IdempotencyClaimResult, error)
	Complete(ctx context.Context, completion *IdempotencyCompletion) error
	Release(ctx context.Context, key, owner string) error
}
