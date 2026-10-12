package resolvertest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
)

type RecordingPermissionEngine struct {
	mocks.AllowAllPermissionEngine
	Request *permissionports.PermissionCheckRequest
}

func (e *RecordingPermissionEngine) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	e.Request = req
	return &permissionports.PermissionCheckResult{Allowed: true}, nil
}
