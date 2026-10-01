package resolvertest

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
)

type GrantingPermissionEngine struct {
	mocks.AllowAllPermissionEngine

	mu       sync.Mutex
	Granted  map[string]bool
	Requests []*services.PermissionCheckRequest
}

func (e *GrantingPermissionEngine) Check(
	_ context.Context,
	req *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.Requests = append(e.Requests, req)

	return &services.PermissionCheckResult{
		Allowed: e.Granted[req.Resource+"|"+string(req.Operation)],
	}, nil
}
