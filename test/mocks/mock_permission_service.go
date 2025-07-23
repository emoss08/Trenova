// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package mocks

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/stretchr/testify/mock"
)

type MockPermissionService struct {
	mock.Mock
}

func (m *MockPermissionService) List(
	ctx context.Context,
	req *services.ListPermissionsRequest,
) (*ports.ListResult[*permission.Permission], error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*ports.ListResult[*permission.Permission]), args.Error(1)
}
func (m *MockPermissionService) HasPermission(
	ctx context.Context,
	check *services.PermissionCheck,
) (services.PermissionCheckResult, error) {
	args := m.Called(ctx, check)
	return args.Get(0).(services.PermissionCheckResult), args.Error(1)
}

func (m *MockPermissionService) HasAnyPermissions(
	ctx context.Context,
	checks []*services.PermissionCheck,
) (services.PermissionCheckResult, error) {
	args := m.Called(ctx, checks)
	return args.Get(0).(services.PermissionCheckResult), args.Error(1)
}

func (m *MockPermissionService) HasFieldPermission(
	ctx context.Context,
	check *services.PermissionCheck,
) (services.PermissionCheckResult, error) {
	args := m.Called(ctx, check)
	return args.Get(0).(services.PermissionCheckResult), args.Error(1)
}

func (m *MockPermissionService) HasAllPermissions(
	ctx context.Context,
	checks []*services.PermissionCheck,
) (services.PermissionCheckResult, error) {
	args := m.Called(ctx, checks)
	return args.Get(0).(services.PermissionCheckResult), args.Error(1)
}

func (m *MockPermissionService) HasAnyFieldPermissions(
	ctx context.Context,
	fields []string,
	check *services.PermissionCheck,
) (services.PermissionCheckResult, error) {
	args := m.Called(ctx, fields, check)
	return args.Get(0).(services.PermissionCheckResult), args.Error(1)
}

func (m *MockPermissionService) HasAllFieldPermissions(
	ctx context.Context,
	fields []string,
	check *services.PermissionCheck,
) (services.PermissionCheckResult, error) {
	args := m.Called(ctx, fields, check)
	return args.Get(0).(services.PermissionCheckResult), args.Error(1)
}

func (m *MockPermissionService) HasScopedPermission(
	ctx context.Context,
	check *services.PermissionCheck,
	requiredScope permission.Scope,
) (services.PermissionCheckResult, error) {
	args := m.Called(ctx, check, requiredScope)
	return args.Get(0).(services.PermissionCheckResult), args.Error(1)
}

func (m *MockPermissionService) HasDependentPermissions(
	ctx context.Context,
	check *services.PermissionCheck,
) (services.PermissionCheckResult, error) {
	args := m.Called(ctx, check)
	return args.Get(0).(services.PermissionCheckResult), args.Error(1)
}

func (m *MockPermissionService) HasTemporalPermission(
	ctx context.Context,
	check *services.PermissionCheck,
) (services.PermissionCheckResult, error) {
	args := m.Called(ctx, check)
	return args.Get(0).(services.PermissionCheckResult), args.Error(1)
}

func (m *MockPermissionService) CheckFieldAccess(
	ctx context.Context,
	userID pulid.ID,
	resource permission.Resource,
	field string,
) services.FieldAccess {
	args := m.Called(ctx, userID, resource, field)
	return args.Get(0).(services.FieldAccess)
}

func (m *MockPermissionService) CheckFieldModification(
	ctx context.Context,
	userID pulid.ID,
	resource permission.Resource,
	field string,
) services.FieldPermissionCheck {
	args := m.Called(ctx, userID, resource, field)
	return args.Get(0).(services.FieldPermissionCheck)
}

func (m *MockPermissionService) CheckFieldView(
	ctx context.Context,
	userID pulid.ID,
	resource permission.Resource,
	field string,
) services.FieldPermissionCheck {
	args := m.Called(ctx, userID, resource, field)
	return args.Get(0).(services.FieldPermissionCheck)
}

func (m *MockPermissionService) SetDefaultField(
	key string,
	value any,
) {
	// * do nothing
	_ = m.Called(key, value)
}

func (m *MockPermissionService) GetServiceStatus() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockPermissionService) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockPermissionService) Stop() error {
	args := m.Called()
	return args.Error(0)
}
