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
	ctx    context.Context
	client *ent.Client
}

// NewRouteControlOps creates a new route control service.
func NewRouteControlOps(ctx context.Context) *RouteControlOps {
	return &RouteControlOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetRouteControl creates a new route control settings for an organization.
func (r *RouteControlOps) GetRouteControl(orgID, buID uuid.UUID) (*ent.RouteControl, error) {
	routeControl, err := r.client.RouteControl.Query().Where(
		routecontrol.HasOrganizationWith(
			organization.ID(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Only(r.ctx)
	if err != nil {
		return nil, err
	}

	return routeControl, nil
}

// UpdateRouteControl updates the route control settings for an organization.
func (r *RouteControlOps) UpdateRouteControl(rc ent.RouteControl) (*ent.RouteControl, error) {
	updatedRC, err := r.client.RouteControl.
		UpdateOneID(rc.ID).
		SetDistanceMethod(rc.DistanceMethod).
		SetMileageUnit(rc.MileageUnit).
		SetGenerateRoutes(rc.GenerateRoutes).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedRC, nil
}
