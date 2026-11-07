package demandforecastrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/postgis"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type areaLocationRow struct {
	ID pulid.ID
}

func (r *repository) ListAreaLocations(
	ctx context.Context,
	req *demandforecastports.AreaLocationsRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		l := locationdb.Locations
		center := dispatchmarket.CenterOf(req.Cell)
		bounds := dispatchmarket.BoundsOf(req.Cell)
		rows, err := keel.Into[areaLocationRow](ctx, r.db.Keel(ctx), l.
			Pick(l.ID).
			Where(
				l.Status.Eq(domaintypes.StatusActive),
				l.Geom.DWithin(
					postgis.Pt(center.Longitude, center.Latitude),
					dispatchmarket.CoverMeters(req.Cell),
				),
				l.Latitude.Gte(bounds.South),
				l.Latitude.Lt(bounds.North),
				l.Longitude.Gte(bounds.West),
				l.Longitude.Lt(bounds.East),
			).
			Order(l.ID.Asc()).
			Limit(demandforecastports.MaxAreaLocations))
		if err != nil {
			return nil, fmt.Errorf("read area locations: %w", err)
		}

		out := make([]pulid.ID, 0, len(rows))
		for _, row := range rows {
			out = append(out, row.ID)
		}

		return out, nil
	})
}
