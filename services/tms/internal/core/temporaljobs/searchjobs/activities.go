package searchjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/meilisearch/providers"
	"github.com/emoss08/trenova/pkg/meilisearchtype"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	ShipmentRepo repositories.ShipmentRepository
	SearchHelper *providers.SearchHelper
}

type Activities struct {
	shipmentRepo repositories.ShipmentRepository
	searchHelper *providers.SearchHelper
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		shipmentRepo: p.ShipmentRepo,
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
		return err
	}

	return a.searchHelper.Index(ctx, shp)
}
