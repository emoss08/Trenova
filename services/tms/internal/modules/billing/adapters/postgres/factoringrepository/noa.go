package factoringrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	noaEntity       = "FactoringNoa"
	openNoaIndex    = "uq_factoring_noas_open"
	maxExpiringNoas = 500
)

var noaErrors = dberror.Mapping{Entity: noaEntity}

type noaRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewNoaRepository(p Params) factoringports.NoaRepository {
	return &noaRepository{
		db: p.DB,
		l:  p.Logger.Named("repository.factoring-noa"),
	}
}

func (r *noaRepository) Create(ctx context.Context, entity *factoring.Noa) (*factoring.Noa, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoring.Noa, error) {
		if err := factoringdb.Noas.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, translateNoaError(err, entity.ID)
		}

		return entity, nil
	})
}

func (r *noaRepository) Update(ctx context.Context, entity *factoring.Noa) (*factoring.Noa, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoring.Noa, error) {
		n := factoringdb.Noas
		if err := n.SaveExcept(ctx, r.db.Keel(ctx), entity, n.CustomerID, n.FactoringCompanyID); err != nil {
			return nil, translateNoaError(err, entity.ID)
		}

		return entity, nil
	})
}

func (r *noaRepository) GetByID(
	ctx context.Context,
	req factoringports.GetNoaRequest,
) (*factoring.Noa, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoring.Noa, error) {
		entity, err := factoringdb.Noas.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, noaErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *noaRepository) GetOpen(
	ctx context.Context,
	req *factoringports.GetOpenNoaRequest,
) (*factoring.Noa, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoring.Noa, error) {
		n := factoringdb.Noas
		entity, err := n.Where(
			n.CustomerID.Eq(req.CustomerID),
			n.FactoringCompanyID.Eq(req.FactoringCompanyID),
			n.Status.Ne(factoring.NoaStatusReleased),
		).First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, noaErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *noaRepository) ListOpen(
	ctx context.Context,
	req *factoringports.ListOpenNoasRequest,
) (map[factoringports.NoaPair]*factoring.Noa, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[factoringports.NoaPair]*factoring.Noa, error) {
			result := make(map[factoringports.NoaPair]*factoring.Noa, len(req.Pairs))
			if len(req.Pairs) == 0 {
				return result, nil
			}

			rows, err := openNoas(req.Pairs).All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, err
			}

			for _, row := range rows {
				result[factoringports.NoaPair{
					CustomerID:         row.CustomerID,
					FactoringCompanyID: row.FactoringCompanyID,
				}] = row
			}

			return result, nil
		},
	)
}

func openNoas(pairs []factoringports.NoaPair) *keel.Query[factoring.Noa] {
	customers := make([]pulid.ID, len(pairs))
	factors := make([]pulid.ID, len(pairs))
	for i, pair := range pairs {
		customers[i] = pair.CustomerID
		factors[i] = pair.FactoringCompanyID
	}

	n := factoringdb.Noas
	wanted := keel.Unnest(
		"wanted",
		[]string{"customer_id", "factoring_company_id"},
		customers,
		factors,
	)

	return n.With(wanted).
		Join(
			wanted,
			wanted.Col("customer_id").EqCol(n.CustomerID),
			wanted.Col("factoring_company_id").EqCol(n.FactoringCompanyID),
		).
		Where(n.Status.Ne(factoring.NoaStatusReleased))
}

func (r *noaRepository) ListConnection(
	ctx context.Context,
	req *factoringports.ListNoaConnectionRequest,
) (*pagination.CursorListResult[*factoring.Noa], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*factoring.Noa], error) {
			n := factoringdb.Noas
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				n.Where(
					keel.When(req.CustomerID.IsNotNil(), n.CustomerID.Eq(req.CustomerID)),
					keel.When(
						req.FactoringCompanyID.IsNotNil(),
						n.FactoringCompanyID.Eq(req.FactoringCompanyID),
					),
					keel.When(len(req.Statuses) > 0, n.Status.In(req.Statuses...)),
				),
				&listquery.Request[factoring.Noa]{Filter: req.Filter, Cursor: req.Cursor},
			)
			if err != nil {
				r.l.Error("failed to list notices of assignment", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *noaRepository) ListExpiring(
	ctx context.Context,
	req factoringports.ListExpiringNoasRequest,
) ([]*factoring.Noa, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*factoring.Noa, error) {
		limit := req.Limit
		if limit <= 0 || limit > maxExpiringNoas {
			limit = maxExpiringNoas
		}

		n := factoringdb.Noas
		rows, err := n.Where(
			n.Status.Eq(factoring.NoaStatusAcknowledged),
			n.ExpiresAt.IsNotNull(),
			n.ExpiresAt.Gt(req.At),
			n.ExpiresAt.Lte(req.Before),
			n.ExpiryWarnedAt.IsNull(),
		).
			Order(n.ExpiresAt.Asc(), n.ID.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}
		if rows == nil {
			return []*factoring.Noa{}, nil
		}

		return rows, nil
	})
}

func (r *noaRepository) ClaimExpiryWarning(
	ctx context.Context,
	req factoringports.ClaimNoaExpiryWarningRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		n := factoringdb.Noas
		rows, err := n.Where(n.ID.Eq(req.ID), n.ExpiryWarnedAt.IsNull()).
			Update(ctx, r.db.Keel(ctx),
				n.ExpiryWarnedAt.To(req.At),
				n.Version.ToExpr(n.Version),
				n.UpdatedAt.ToExpr(n.UpdatedAt),
			)
		if err != nil {
			return false, noaErrors.Map(err, req.ID.String())
		}

		return rows > 0, nil
	})
}

func translateNoaError(err error, id pulid.ID) error {
	if dberror.IsUniqueConstraintViolation(err) &&
		dberror.ExtractConstraintName(err) == openNoaIndex {
		return errortypes.NewConflictError(
			"This customer already has an open notice of assignment for this factor",
		)
	}

	return noaErrors.Map(err, id.String())
}
