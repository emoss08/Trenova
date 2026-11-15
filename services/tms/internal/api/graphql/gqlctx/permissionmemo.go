package gqlctx

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

type permissionMemoKey struct{}

type PermissionMemo struct {
	mu      sync.RWMutex
	results map[string]permissionports.PermissionCheckResult
}

func NewPermissionMemo() *PermissionMemo {
	return &PermissionMemo{results: make(map[string]permissionports.PermissionCheckResult)}
}

func WithPermissionMemo(ctx context.Context, memo *PermissionMemo) context.Context {
	return context.WithValue(ctx, permissionMemoKey{}, memo)
}

func PermissionMemoFrom(ctx context.Context) (*PermissionMemo, bool) {
	memo, ok := ctx.Value(permissionMemoKey{}).(*PermissionMemo)
	return memo, ok && memo != nil
}

func (m *PermissionMemo) Lookup(
	key string,
) (result permissionports.PermissionCheckResult, found bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result, found = m.results[key]
	return result, found
}

func (m *PermissionMemo) Store(key string, result permissionports.PermissionCheckResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.results[key] = result
}

func (m *PermissionMemo) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.results)
}
