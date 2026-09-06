package gqlctx

import (
	"context"
	"sync"
)

type permissionMemoKey struct{}

type PermissionMemo struct {
	mu      sync.RWMutex
	results map[string]bool
}

func NewPermissionMemo() *PermissionMemo {
	return &PermissionMemo{results: make(map[string]bool)}
}

func WithPermissionMemo(ctx context.Context, memo *PermissionMemo) context.Context {
	return context.WithValue(ctx, permissionMemoKey{}, memo)
}

func PermissionMemoFrom(ctx context.Context) (*PermissionMemo, bool) {
	memo, ok := ctx.Value(permissionMemoKey{}).(*PermissionMemo)
	return memo, ok && memo != nil
}

func (m *PermissionMemo) Lookup(key string) (allowed, found bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	allowed, found = m.results[key]
	return allowed, found
}

func (m *PermissionMemo) Store(key string, allowed bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.results[key] = allowed
}

func (m *PermissionMemo) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.results)
}
