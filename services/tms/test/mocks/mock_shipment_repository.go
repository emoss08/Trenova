/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package mocks

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
)

type MockShipmentRepository struct {
	mock.Mock
}

func (m *MockShipmentRepository) UpdateStatus(
	ctx context.Context,
	req *repositories.UpdateShipmentStatusRequest,
) (*shipment.Shipment, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*shipment.Shipment), args.Error(1)
}

// Add other required methods as no-ops for the interface
func (m *MockShipmentRepository) List(
	ctx context.Context,
	opts *repositories.ListShipmentOptions,
) (*ports.ListResult[*shipment.Shipment], error) {
	return nil, nil
}

func (m *MockShipmentRepository) TransferOwnership(
	ctx context.Context,
	req *repositories.TransferOwnershipRequest,
) (*shipment.Shipment, error) {
	return nil, nil
}

func (m *MockShipmentRepository) UnCancel(
	ctx context.Context,
	req *repositories.UnCancelShipmentRequest,
) (*shipment.Shipment, error) {
	return nil, nil
}

func (m *MockShipmentRepository) GetAll(
	ctx context.Context,
) (*ports.ListResult[*shipment.Shipment], error) {
	args := m.Called(ctx)
	return args.Get(0).(*ports.ListResult[*shipment.Shipment]), args.Error(1)
}

func (m *MockShipmentRepository) GetByID(
	ctx context.Context,
	opts *repositories.GetShipmentByIDOptions,
) (*shipment.Shipment, error) {
	return nil, nil
}

func (m *MockShipmentRepository) Create(
	ctx context.Context,
	entity *shipment.Shipment,
	userID pulid.ID,
) (*shipment.Shipment, error) {
	return nil, nil
}

func (m *MockShipmentRepository) Update(
	ctx context.Context,
	entity *shipment.Shipment,
	userID pulid.ID,
) (*shipment.Shipment, error) {
	return nil, nil
}

func (m *MockShipmentRepository) Cancel(
	ctx context.Context,
	req *repositories.CancelShipmentRequest,
) (*shipment.Shipment, error) {
	return nil, nil
}

func (m *MockShipmentRepository) GetByOrgID(
	ctx context.Context,
	orgID pulid.ID,
) (*ports.ListResult[*shipment.Shipment], error) {
	return nil, nil
}

func (m *MockShipmentRepository) Duplicate(
	ctx context.Context,
	req *repositories.DuplicateShipmentRequest,
) (*shipment.Shipment, error) {
	return nil, nil
}

func (m *MockShipmentRepository) CheckForDuplicateBOLs(
	ctx context.Context,
	currentBOL string,
	orgID pulid.ID,
	buID pulid.ID,
	excludeID *pulid.ID,
) ([]repositories.DuplicateBOLsResult, error) {
	return nil, nil
}

func (m *MockShipmentRepository) BulkDuplicate(
	ctx context.Context,
	req *repositories.DuplicateShipmentRequest,
) ([]*shipment.Shipment, error) {
	return nil, nil
}

func (m *MockShipmentRepository) GetByDateRange(
	ctx context.Context,
	req *repositories.GetShipmentsByDateRangeRequest,
) (*ports.ListResult[*shipment.Shipment], error) {
	return nil, nil
}

func (m *MockShipmentRepository) CalculateShipmentTotals(
	ctx context.Context,
	shp *shipment.Shipment,
	userID pulid.ID,
) (*repositories.ShipmentTotalsResponse, error) {
	return nil, nil
}

func (m *MockShipmentRepository) GetPreviousRates(
	ctx context.Context,
	req *repositories.GetPreviousRatesRequest,
) (*ports.ListResult[*shipment.Shipment], error) {
	return nil, nil
}

func (m *MockShipmentRepository) DelayShipments(ctx context.Context) ([]*shipment.Shipment, error) {
	return nil, nil
}

func (m *MockShipmentRepository) GetDelayedShipments(
	ctx context.Context,
) ([]*shipment.Shipment, error) {
	return nil, nil
}
