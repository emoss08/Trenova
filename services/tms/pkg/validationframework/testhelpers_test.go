package validationframework

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
)

type mockTenantedEntity struct {
	id             pulid.ID
	organizationID pulid.ID
	businessUnitID pulid.ID
	tableName      string
	Name           string
	Code           string
	Type           string
	ParentID       pulid.ID
	StartDate      *int64
	EndDate        *int64
	Weight         *float64
	validationErr  bool
}

func (m *mockTenantedEntity) GetID() pulid.ID {
	return m.id
}

func (m *mockTenantedEntity) GetTableName() string {
	return m.tableName
}

func (m *mockTenantedEntity) GetOrganizationID() pulid.ID {
	return m.organizationID
}

func (m *mockTenantedEntity) GetBusinessUnitID() pulid.ID {
	return m.businessUnitID
}

func (m *mockTenantedEntity) Validate(multiErr *errortypes.MultiError) {
	if m.validationErr {
		multiErr.Add("name", errortypes.ErrRequired, "Name is required")
	}
}

type mockUniquenessChecker struct {
	mock.Mock
}

func newMockUniquenessChecker() *mockUniquenessChecker {
	return &mockUniquenessChecker{}
}

func (m *mockUniquenessChecker) CheckUniqueness(
	ctx context.Context,
	req *UniquenessRequest,
) (bool, error) {
	args := m.Called(ctx, req)
	return args.Bool(0), args.Error(1)
}
