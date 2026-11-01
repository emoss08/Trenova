package dothazmatreferencerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/reference/domain/dothazmatreference"
	"github.com/emoss08/trenova/internal/modules/reference/domain/dothazmatreference/dothazmatreferencedb"
	"github.com/emoss08/trenova/internal/modules/reference/ports/dothazmatreferenceports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var dotHazmatReferenceErrors = dberror.Mapping{Entity: "DotHazmatReference"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) dothazmatreferenceports.DotHazmatReferenceRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.dot-hazmat-reference-repository"),
	}
}

func (r *repository) GetByIDs(
	ctx context.Context,
	ids []pulid.ID,
) ([]*dothazmatreference.DotHazmatReference, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*dothazmatreference.DotHazmatReference, error) {
			if len(ids) == 0 {
				return []*dothazmatreference.DotHazmatReference{}, nil
			}

			d := dothazmatreferencedb.DotHazmatReferences
			entities, err := d.Where(d.ID.In(ids...)).
				Limit(len(ids)).
				Cache(resultcache.Catalog()...).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, dotHazmatReferenceErrors.Map(err, "")
			}

			return entities, nil
		},
	)
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*dothazmatreference.DotHazmatReference], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*dothazmatreference.DotHazmatReference], error) {
			d := dothazmatreferencedb.DotHazmatReferences
			q := d.Pick(
				d.ID,
				d.UnNumber,
				d.ProperShippingName,
				d.HazardClass,
				d.SubsidiaryHazard,
				d.PackingGroup,
				d.SpecialProvisions,
				d.ErgGuide,
				d.Symbols,
			)

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req,
				func(s string) keel.Pred {
					return keel.Or(d.UnNumber.HasPrefix(s), d.ProperShippingName.ContainsFold(s))
				},
			)
		},
	)
}
