package resolvertest

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
)

type GrantingPermissionEngine struct {
	mocks.AllowAllPermissionEngine

	mu       sync.Mutex
	Granted  map[string]bool
	Requests []*permissionports.PermissionCheckRequest
}

func (e *GrantingPermissionEngine) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.Requests = append(e.Requests, req)

	return &permissionports.PermissionCheckResult{
		Allowed: e.Granted[req.Resource+"|"+string(req.Operation)],
	}, nil
}
