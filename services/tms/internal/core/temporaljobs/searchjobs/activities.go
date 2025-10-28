package searchjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/meilisearch/providers"
	"github.com/emoss08/trenova/pkg/meilisearchtype"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	ShipmentRepo repositories.ShipmentRepository
	CustomerRepo repositories.CustomerRepository
	SearchHelper *providers.SearchHelper
}

type Activities struct {
	shipmentRepo repositories.ShipmentRepository
	customerRepo repositories.CustomerRepository
	searchHelper *providers.SearchHelper
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		shipmentRepo: p.ShipmentRepo,
		customerRepo: p.CustomerRepo,
		searchHelper: p.SearchHelper,
	}
}

func (a *Activities) IndexEntityActivity(
	ctx context.Context,
	payload *IndexEntityPayload,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info(
		"Starting insert ai log activity",
		"entityId", payload.EntityID,
	)

	switch payload.EntityType {
	case meilisearchtype.EntityTypeShipment:
		return a.indexShipment(ctx, payload)
	case meilisearchtype.EntityTypeCustomer:
		return a.indexCustomer(ctx, payload)
	default:
		return fmt.Errorf("unsupported entity type: %s", payload.EntityType)
	}
}

func (a *Activities) BulkIndexEntityActivity(
	ctx context.Context,
	payload *BulkIndexEntityPayload,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting bulk index entity activity",
		"entityType", payload.EntityType,
		"entityIDs", payload.EntityIDs,
	)

	switch payload.EntityType {
	case meilisearchtype.EntityTypeShipment:
		return a.bulkIndexShipments(ctx, payload)
	default:
		return fmt.Errorf("unsupported entity type: %s", payload.EntityType)
	}
}

func (a *Activities) indexShipment(ctx context.Context, payload *IndexEntityPayload) error {
	shp, err := a.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:    payload.EntityID,
		OrgID: payload.GetOrganizationID(),
		BuID:  payload.GetBusinessUnitID(),
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		appErr := temporaltype.ClassifyError(err)
		return appErr.ToTemporalError()
	}

	// We should retry this operation if it fails
	if err = a.searchHelper.Index(ctx, shp); err != nil {
		return temporaltype.NewRetryableError("failed to index shipment", err).ToTemporalError()
	}

	return nil
}

func (a *Activities) bulkIndexShipments(
	ctx context.Context,
	payload *BulkIndexEntityPayload,
) error {
	shipments, err := a.shipmentRepo.GetByIDs(ctx, &repositories.GetShipmentsByIDsRequest{
		IDs:   payload.EntityIDs,
		OrgID: payload.GetOrganizationID(),
		BuID:  payload.GetBusinessUnitID(),
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		appErr := temporaltype.ClassifyError(err)
		return appErr.ToTemporalError()
	}

	searchableShipments := make([]meilisearchtype.Searchable, 0, len(shipments))
	for _, shp := range shipments {
		searchableShipments = append(searchableShipments, shp)
	}

	if err = a.searchHelper.BatchIndex(ctx, searchableShipments); err != nil {
		return temporaltype.NewRetryableError("failed to index shipments", err).ToTemporalError()
	}

	return nil
}

func (a *Activities) indexCustomer(ctx context.Context, payload *IndexEntityPayload) error {
	cus, err := a.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
		ID:    payload.EntityID,
		OrgID: payload.GetOrganizationID(),
		BuID:  payload.GetBusinessUnitID(),
		CustomerFilterOptions: repositories.CustomerFilterOptions{
			IncludeState: true,
		},
	})
	if err != nil {
		appErr := temporaltype.ClassifyError(err)
		return appErr.ToTemporalError()
	}

	// We should retry this operation if it fails
	if err = a.searchHelper.Index(ctx, cus); err != nil {
		return temporaltype.NewRetryableError("failed to index customer", err).ToTemporalError()
	}

	return nil
}
