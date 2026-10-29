package querytooltest

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/agentceiling"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/permtest"
	"github.com/emoss08/trenova/shared/pulid"
)

func PersonReaching(
	resource permission.Resource,
	sensitivity permission.FieldSensitivity,
) *FakePermissions {
	return &FakePermissions{
		Readable: map[string]*permissionports.ResourcePermissionDetail{
			resource.String(): {
				Resource:       resource.String(),
				Operations:     []permission.Operation{permission.OpRead},
				MaxSensitivity: sensitivity,
			},
		},
	}
}

func PersonReachingEach(
	sensitivity permission.FieldSensitivity,
	resources ...permission.Resource,
) *FakePermissions {
	readable := make(map[string]*permissionports.ResourcePermissionDetail, len(resources))
	for _, resource := range resources {
		readable[resource.String()] = &permissionports.ResourcePermissionDetail{
			Resource:       resource.String(),
			Operations:     []permission.Operation{permission.OpRead},
			MaxSensitivity: sensitivity,
		}
	}

	return &FakePermissions{Readable: readable}
}

type FakePermissions struct {
	permissionports.PermissionEngine

	Allowed  bool
	Captured *permissionports.PermissionCheckRequest
	// readable names the resources the actor may read from the report
	// catalog; nil means every resource, at full sensitivity.
	Readable map[string]*permissionports.ResourcePermissionDetail
}

func (f *FakePermissions) GetResourcePermissions(
	_ context.Context,
	_, _ pulid.ID,
	resource string,
) (*permissionports.ResourcePermissionDetail, error) {
	if f.Readable == nil {
		return &permissionports.ResourcePermissionDetail{
			Resource:       resource,
			Operations:     []permission.Operation{permission.OpRead},
			MaxSensitivity: permission.SensitivityConfidential,
		}, nil
	}
	if detail, ok := f.Readable[resource]; ok {
		return detail, nil
	}

	return &permissionports.ResourcePermissionDetail{Resource: resource}, nil
}

func (f *FakePermissions) GetResourcePermissionsBatch(
	ctx context.Context,
	userID, orgID pulid.ID,
	resources []string,
) (map[string]*permissionports.ResourcePermissionDetail, error) {
	return permtest.Batch(ctx, f, userID, orgID, resources)
}

func (f *FakePermissions) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	f.Captured = req

	allowed := f.Allowed
	switch {
	case req.PrincipalType == permissionports.PrincipalTypeAgent:
		// The engine answers an agent principal from the platform allow-list
		// before any role applies; the fake answers the same way.
		allowed = agentceiling.Allows(permission.Resource(req.Resource), req.Operation)
	case f.Readable != nil:
		detail, ok := f.Readable[req.Resource]
		allowed = ok && slices.Contains(detail.Operations, req.Operation)
	}

	return &permissionports.PermissionCheckResult{
		Allowed: allowed,
		Reason:  "not permitted",
	}, nil
}
