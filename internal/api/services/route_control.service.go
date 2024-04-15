package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/routecontrol"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/google/uuid"
)

// RouteControlService is the service for route control settings.
type RouteControlService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewRouteControlService creates a new route control service.
func NewRouteControlService(s *api.Server) *RouteControlService {
	return &RouteControlService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetRouteControl creates a new route control settings for an organization.
func (r *RouteControlService) GetRouteControl(ctx context.Context, orgID, buID uuid.UUID) (*ent.RouteControl, error) {
	routeControl, err := r.Client.RouteControl.Query().Where(
		routecontrol.HasOrganizationWith(
			organization.ID(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Only(ctx)
	if err != nil {
		return nil, err
	}

	return routeControl, nil
}

// UpdateRouteControl updates the route control settings for an organization.
func (r *RouteControlService) UpdateRouteControl(ctx context.Context, rc *ent.RouteControl) (*ent.RouteControl, error) {
	updatedEntity := new(ent.RouteControl)
	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateRouteControl(ctx, tx, rc)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *RouteControlService) updateRouteControl(
	ctx context.Context, tx *ent.Tx, rc *ent.RouteControl,
) (*ent.RouteControl, error) {
	updateOp := tx.RouteControl.UpdateOneID(rc.ID).
		SetDistanceMethod(rc.DistanceMethod).
		SetMileageUnit(rc.MileageUnit).
		SetGenerateRoutes(rc.GenerateRoutes)

	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
