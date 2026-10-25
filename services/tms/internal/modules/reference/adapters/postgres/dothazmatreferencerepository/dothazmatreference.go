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

func (r *repository) GetByID(
	ctx context.Context,
	req dothazmatreferenceports.GetDotHazmatReferenceByIDRequest,
) (*dothazmatreference.DotHazmatReference, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*dothazmatreference.DotHazmatReference, error) {
			entity, err := dothazmatreferencedb.DotHazmatReferences.Get(
				ctx,
				r.db.Keel(ctx),
				req.DotHazmatReferenceID,
			)
			if err != nil {
				return nil, dotHazmatReferenceErrors.Map(err, req.DotHazmatReferenceID.String())
			}

			return entity, nil
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
