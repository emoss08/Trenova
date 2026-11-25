package shipmentboardresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (r *QueryResolver) attachCapacityPhotos(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	units []*gqlmodel.CapacityUnit,
) error {
	ids := make([]pulid.ID, 0, len(units))
	for _, unit := range units {
		if unit != nil && unit.Kind == gqlmodel.CapacityUnitKindDriver {
			ids = append(ids, pulid.ID(unit.ID))
		}
	}
	photos, err := r.WorkerPhotoService.URLsByWorker(ctx, tenantInfo, ids)
	if err != nil {
		return err
	}
	for _, unit := range units {
		if unit != nil && unit.Kind == gqlmodel.CapacityUnitKindDriver {
			unit.Photo = photos[pulid.ID(unit.ID)]
		}
	}
	return nil
}

func (r *QueryResolver) attachSuggestionPhotos(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	drivers []*gqlmodel.DriverCoverageSuggestion,
) error {
	ids := make([]pulid.ID, 0, len(drivers))
	for _, driver := range drivers {
		if driver != nil {
			ids = append(ids, pulid.ID(driver.WorkerID))
		}
	}
	photos, err := r.WorkerPhotoService.URLsByWorker(ctx, tenantInfo, ids)
	if err != nil {
		return err
	}
	for _, driver := range drivers {
		if driver != nil {
			driver.Photo = photos[pulid.ID(driver.WorkerID)]
		}
	}
	return nil
}
