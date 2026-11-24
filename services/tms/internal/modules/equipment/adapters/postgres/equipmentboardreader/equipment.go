package equipmentboardreader

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Tractors struct{ db *postgres.Connection }

type Trailers struct{ db *postgres.Connection }

func NewTractors(p Params) *Tractors { return &Tractors{db: p.DB} }

func NewTrailers(p Params) *Trailers { return &Trailers{db: p.DB} }

func (*Tractors) Kind() recordkind.Kind { return recordkind.Kind(permission.ResourceTractor) }
func (*Trailers) Kind() recordkind.Kind { return recordkind.Kind(permission.ResourceTrailer) }

type summaryRow struct {
	ID       pulid.ID
	Label    string
	Subtitle string
	Status   string
}

func text(e keel.Expr) keel.StringExpr {
	return keel.CastText(e).Coalesce("")
}

func summaries(
	ctx context.Context,
	db *postgres.Connection,
	ids []pulid.ID,
	query func(ids []pulid.ID) keel.Projection,
) (map[pulid.ID]boardreading.Summary, error) {
	if len(ids) == 0 {
		return map[pulid.ID]boardreading.Summary{}, nil
	}

	rows, err := keel.Into[summaryRow](ctx, db.Keel(ctx), query(ids))
	if err != nil {
		return nil, err
	}
	out := make(map[pulid.ID]boardreading.Summary, len(rows))
	for _, row := range rows {
		status, tone := equipmentStatus(ctx, row.Status)
		out[row.ID] = boardreading.Summary{
			Label:      strings.TrimSpace(row.Label),
			Subtitle:   strings.TrimSpace(row.Subtitle),
			Status:     status,
			StatusTone: tone,
		}
	}

	return out, nil
}

func (r *Tractors) Summaries(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Summary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Summary, error) {
			return summaries(ctx, r.db, req.IDs, func(ids []pulid.ID) keel.Projection {
				t := tractordb.Tractors

				return t.Pick(t.ID, text(t.Code), text(keel.ConcatWS(" ", t.Make, t.Model)), text(t.Status)).
					Where(t.ID.In(ids...))
			})
		},
	)
}

func (r *Trailers) Summaries(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Summary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Summary, error) {
			return summaries(ctx, r.db, req.IDs, func(ids []pulid.ID) keel.Projection {
				t := trailerdb.Trailers

				return t.Pick(t.ID, text(t.Code), text(keel.ConcatWS(" ", t.Make, t.Model)), text(t.Status)).
					Where(t.ID.In(ids...))
			})
		},
	)
}
