package mocks

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/stretchr/testify/mock"
)

type MockAssignmentRepository struct {
	mock.Mock
}

func (m *MockAssignmentRepository) List(
	ctx context.Context,
	req repositories.ListAssignmentsRequest,
) (*ports.ListResult[*shipment.Assignment], error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*ports.ListResult[*shipment.Assignment]), args.Error(1)
}

func (m *MockAssignmentRepository) BulkAssign(
	ctx context.Context,
	req *repositories.AssignmentRequest,
) ([]*shipment.Assignment, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]*shipment.Assignment), args.Error(1)
}

func (m *MockAssignmentRepository) SingleAssign(
	ctx context.Context,
	a *shipment.Assignment,
) (*shipment.Assignment, error) {
	args := m.Called(ctx, a)
	return args.Get(0).(*shipment.Assignment), args.Error(1)
}

func (m *MockAssignmentRepository) Reassign(
	ctx context.Context,
	a *shipment.Assignment,
) (*shipment.Assignment, error) {
	args := m.Called(ctx, a)
	return args.Get(0).(*shipment.Assignment), args.Error(1)
}

func (m *MockAssignmentRepository) GetByID(
	ctx context.Context,
	opts repositories.GetAssignmentByIDOptions,
) (*shipment.Assignment, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*shipment.Assignment), args.Error(1)
}
