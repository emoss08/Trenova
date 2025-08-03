/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package mocks

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/stretchr/testify/mock"
)

type MockTractorRepository struct {
	mock.Mock
}

func (m *MockTractorRepository) List(
	ctx context.Context,
	req *repositories.ListTractorRequest,
) (*ports.ListResult[*tractor.Tractor], error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*ports.ListResult[*tractor.Tractor]), args.Error(1)
}

func (m *MockTractorRepository) GetByID(
	ctx context.Context,
	req *repositories.GetTractorByIDRequest,
) (*tractor.Tractor, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*tractor.Tractor), args.Error(1)
}

func (m *MockTractorRepository) GetByPrimaryWorkerID(
	ctx context.Context,
	req repositories.GetTractorByPrimaryWorkerIDRequest,
) (*tractor.Tractor, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*tractor.Tractor), args.Error(1)
}

func (m *MockTractorRepository) Create(
	ctx context.Context,
	t *tractor.Tractor,
) (*tractor.Tractor, error) {
	args := m.Called(ctx, t)
	return args.Get(0).(*tractor.Tractor), args.Error(1)
}

func (m *MockTractorRepository) Update(
	ctx context.Context,
	t *tractor.Tractor,
) (*tractor.Tractor, error) {
	args := m.Called(ctx, t)
	return args.Get(0).(*tractor.Tractor), args.Error(1)
}

func (m *MockTractorRepository) Assignment(
	ctx context.Context,
	req repositories.TractorAssignmentRequest,
) (*repositories.AssignmentResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*repositories.AssignmentResponse), args.Error(1)
}
