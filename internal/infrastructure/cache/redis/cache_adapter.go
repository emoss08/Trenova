package redis

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/infra"
)

// CacheAdapter wraps the Redis client to implement infra.CacheClient
type CacheAdapter struct {
	client *Client
}

// NewCacheAdapter creates a new cache adapter
func NewCacheAdapter(client *Client) infra.CacheClient {
	return &CacheAdapter{client: client}
}

// Get retrieves a value by key
func (a *CacheAdapter) Get(ctx context.Context, key string) (string, error) {
	return a.client.Get(ctx, key)
}

// Set stores a value with expiration
func (a *CacheAdapter) Set(
	ctx context.Context,
	key string,
	value any,
	expiration time.Duration,
) error {
	return a.client.Set(ctx, key, value, expiration)
}

// Del deletes a single key (adapter for the variadic Del method)
func (a *CacheAdapter) Del(ctx context.Context, key string) error {
	return a.client.Del(ctx, key)
}

// Exists checks if a key exists
func (a *CacheAdapter) Exists(ctx context.Context, key string) (bool, error) {
	return a.client.Exists(ctx, key)
}

// GetJSON retrieves and unmarshals JSON data
func (a *CacheAdapter) GetJSON(ctx context.Context, key string, dest any) error {
	// Use "$" as the default path to get the entire JSON document
	return a.client.GetJSON(ctx, "$", key, dest)
}

// SetJSON marshals and stores JSON data
func (a *CacheAdapter) SetJSON(
	ctx context.Context,
	key string,
	value any,
	expiration time.Duration,
) error {
	// Use "$" as the default path to set the entire JSON document
	return a.client.SetJSON(ctx, "$", key, value, expiration)
}

// Incr increments a counter
func (a *CacheAdapter) Incr(ctx context.Context, key string) (int64, error) {
	return a.client.Incr(ctx, key)
}

// IncrBy increments a counter by a specific value
func (a *CacheAdapter) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return a.client.IncrBy(ctx, key, value)
}

// HGet gets a hash field value
func (a *CacheAdapter) HGet(ctx context.Context, key, field string) (string, error) {
	return a.client.HGet(ctx, key, field)
}

// HSet sets a hash field value
func (a *CacheAdapter) HSet(ctx context.Context, key, field string, value any) error {
	return a.client.HSet(ctx, key, field, value)
}

// SAdd adds members to a set
func (a *CacheAdapter) SAdd(ctx context.Context, key string, members ...any) error {
	return a.client.SAdd(ctx, key, members...)
}

// SRem removes members from a set
func (a *CacheAdapter) SRem(ctx context.Context, key string, members ...any) error {
	return a.client.SRem(ctx, key, members...)
}

// SMembers gets all members of a set
func (a *CacheAdapter) SMembers(ctx context.Context, key string) ([]string, error) {
	return a.client.SMembers(ctx, key)
}

// Expire sets a key expiration
func (a *CacheAdapter) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return a.client.Expire(ctx, key, expiration)
}

// GetTTL gets the TTL of a key
func (a *CacheAdapter) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return a.client.GetTTL(ctx, key)
}

// Keys gets keys matching a pattern
func (a *CacheAdapter) Keys(ctx context.Context, pattern string) ([]string, error) {
	return a.client.Keys(ctx, pattern)
}

// Pipeline returns a pipeliner for batch operations
func (a *CacheAdapter) Pipeline() infra.Pipeliner {
	// For now, return a simple implementation
	// This would need to be properly implemented based on your needs
	return &simplePipeliner{}
}

// Close closes the Redis connection
func (a *CacheAdapter) Close() error {
	return a.client.Close()
}

// simplePipeliner is a basic implementation of Pipeliner
type simplePipeliner struct{}

func (p *simplePipeliner) Exec(ctx context.Context) error {
	return nil
}

func (p *simplePipeliner) Queue(cmd infra.CacheCommand) {
	// No-op for now
}

func (p *simplePipeliner) Incr(ctx context.Context, key string) {
	// No-op for now
}
