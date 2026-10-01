package resolvertest

import (
	"context"

	servicesport "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
)

type RecordingPermissionEngine struct {
	mocks.AllowAllPermissionEngine
	Request *servicesport.PermissionCheckRequest
}

func (e *RecordingPermissionEngine) Check(
	_ context.Context,
	req *servicesport.PermissionCheckRequest,
) (*servicesport.PermissionCheckResult, error) {
	e.Request = req
	return &servicesport.PermissionCheckResult{Allowed: true}, nil
}
