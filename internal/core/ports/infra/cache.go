/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package infra

import (
	"context"
	"time"
)

type Entry struct {
	Value      any
	Expiration time.Duration
	CreatedAt  int64
}

type CacheClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
	Del(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// JSON Operations
	GetJSON(ctx context.Context, key string, dest any) error
	SetJSON(ctx context.Context, key string, value any, expiration time.Duration) error

	// Counter Operations
	Incr(ctx context.Context, key string) (int64, error)
	IncrBy(ctx context.Context, key string, value int64) (int64, error)

	// Hash Operations
	HGet(ctx context.Context, key, field string) (string, error)
	HSet(ctx context.Context, key, field string, value any) error

	// Set Operations
	SAdd(ctx context.Context, key string, members ...any) error
	SRem(ctx context.Context, key string, members ...any) error
	SMembers(ctx context.Context, key string) ([]string, error)

	// Key Operations
	Expire(ctx context.Context, key string, expiration time.Duration) error
	GetTTL(ctx context.Context, key string) (time.Duration, error)
	Keys(ctx context.Context, pattern string) ([]string, error)

	// Batch Operations
	Pipeline() Pipeliner

	// Cleanup
	Close() error
}

// Pipeliner defines the interface for batch operations
type Pipeliner interface {
	Exec(ctx context.Context) error
	Queue(cmd CacheCommand)
	Incr(ctx context.Context, key string)
}

type CacheCommand interface {
	Name() string
	Args() []any
}
