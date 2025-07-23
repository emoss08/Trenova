// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package mocks

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/stretchr/testify/mock"
)

type MockDedicatedLaneRepository struct {
	mock.Mock
}

func (m *MockDedicatedLaneRepository) List(
	ctx context.Context,
	req *repositories.ListDedicatedLaneRequest,
) (*ports.ListResult[*dedicatedlane.DedicatedLane], error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*ports.ListResult[*dedicatedlane.DedicatedLane]), args.Error(1)
}

func (m *MockDedicatedLaneRepository) GetByID(
	ctx context.Context,
	req *repositories.GetDedicatedLaneByIDRequest,
) (*dedicatedlane.DedicatedLane, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*dedicatedlane.DedicatedLane), args.Error(1)
}

func (m *MockDedicatedLaneRepository) FindByShipment(
	ctx context.Context,
	req *repositories.FindDedicatedLaneByShipmentRequest,
) (*dedicatedlane.DedicatedLane, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*dedicatedlane.DedicatedLane), args.Error(1)
}

func (m *MockDedicatedLaneRepository) Create(
	ctx context.Context,
	dl *dedicatedlane.DedicatedLane,
) (*dedicatedlane.DedicatedLane, error) {
	args := m.Called(ctx, dl)
	return args.Get(0).(*dedicatedlane.DedicatedLane), args.Error(1)
}

func (m *MockDedicatedLaneRepository) Update(
	ctx context.Context,
	dl *dedicatedlane.DedicatedLane,
) (*dedicatedlane.DedicatedLane, error) {
	args := m.Called(ctx, dl)
	return args.Get(0).(*dedicatedlane.DedicatedLane), args.Error(1)
}
