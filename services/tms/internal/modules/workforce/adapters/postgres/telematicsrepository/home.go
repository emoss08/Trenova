package telematicsrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/shared/pulid"
)

type tractorHomeRow struct {
	TractorID     pulid.ID
	WorkerID      pulid.ID
	HomeLatitude  float64
	HomeLongitude float64
}

func (r *repository) ListTractorHomes(
	ctx context.Context,
	tractorIDs []pulid.ID,
) ([]telematicsports.TractorHome, error) {
	if len(tractorIDs) == 0 {
		return []telematicsports.TractorHome{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]telematicsports.TractorHome, error) {
		t := tractordb.Tractors
		w := workerdb.Workers
		q := t.Join(w, w.ID.EqCol(t.PrimaryWorkerID)).
			Pick(t.ID, w.ID, w.HomeLatitude, w.HomeLongitude).
			Where(
				t.ID.In(tractorIDs...),
				w.HomeLatitude.IsNotNull(),
				w.HomeLongitude.IsNotNull(),
			).
			Limit(len(tractorIDs))

		rows, err := keel.Into[tractorHomeRow](ctx, r.db.Keel(ctx), q)
		if err != nil {
			return nil, fmt.Errorf("list tractor homes: %w", err)
		}

		homes := make([]telematicsports.TractorHome, 0, len(rows))
		for _, row := range rows {
			homes = append(homes, telematicsports.TractorHome(row))
		}

		return homes, nil
	})
}

func (r *repository) RecordHomeVisits(
	ctx context.Context,
	visits []telematicsports.HomeVisit,
) error {
	if len(visits) == 0 {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		w := workerdb.Workers
		for _, visit := range visits {
			if _, err := w.Where(
				w.ID.Eq(visit.WorkerID),
				keel.Or(w.LastHomeAt.IsNull(), w.LastHomeAt.Lt(visit.At)),
			).Update(
				ctx,
				r.db.Keel(ctx),
				w.LastHomeAt.To(visit.At),
				w.Version.ToExpr(w.Version),
			); err != nil {
				return fmt.Errorf("record home visit: %w", err)
			}
		}

		return nil
	})
}
