package agenttoolstest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

type FakePermissionCheck struct {
	Allowed  bool `json:"-"`
	err      error
	Requests []*permissionports.PermissionCheckRequest `json:"-"`
}

func (f *FakePermissionCheck) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	f.Requests = append(f.Requests, req)
	if f.err != nil {
		return nil, f.err
	}

	return &permissionports.PermissionCheckResult{Allowed: f.Allowed}, nil
}
