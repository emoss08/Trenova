package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/emoss08/trenova/ent/routecontrol"
	"github.com/google/uuid"
)

// RouteControlOps is the service for route control settings.
type RouteControlOps struct {
	client *ent.Client
}

// NewRouteControlOps creates a new route control service.
func NewRouteControlOps() *RouteControlOps {
	return &RouteControlOps{
		client: database.GetClient(),
	}
}

// GetRouteControl creates a new route control settings for an organization.
func (r *RouteControlOps) GetRouteControl(ctx context.Context, orgID, buID uuid.UUID) (*ent.RouteControl, error) {
	routeControl, err := r.client.RouteControl.Query().Where(
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
func (r *RouteControlOps) UpdateRouteControl(ctx context.Context, rc ent.RouteControl) (*ent.RouteControl, error) {
	updatedRC, err := r.client.RouteControl.
		UpdateOneID(rc.ID).
		SetDistanceMethod(rc.DistanceMethod).
		SetMileageUnit(rc.MileageUnit).
		SetGenerateRoutes(rc.GenerateRoutes).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedRC, nil
}
