package dispatchplanpricerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice/dispatchplanpricedb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) dispatchplanports.PriceRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.dispatch-plan-price-repository"),
	}
}

func (r *repository) Replace(
	ctx context.Context,
	req *dispatchplanports.ReplacePricesRequest,
) error {
	if err := validatePrices(req); err != nil {
		return err
	}
	if len(req.MoveIDs) == 0 {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := dispatchplanpricedb.DispatchPlanPrices
		if _, err := p.Where(
			p.MoveID.In(req.MoveIDs...),
			p.GeneratedAt.Lte(req.GeneratedAt),
		).Delete(ctx, r.db.Keel(ctx)); err != nil {
			return fmt.Errorf("clear plan prices: %w", err)
		}
		if len(req.Prices) == 0 {
			return nil
		}

		if _, err := p.Upsert(
			ctx,
			r.db.Keel(ctx),
			req.Prices,
			keel.OnConflict(p.OrganizationID, p.BusinessUnitID, p.MoveID).DoNothing(),
		); err != nil {
			return fmt.Errorf("write plan prices: %w", err)
		}

		return nil
	})
}

func (r *repository) ListByMoves(
	ctx context.Context,
	req *dispatchplanports.ListPricesRequest,
) ([]*dispatchplanprice.DispatchPlanPrice, error) {
	if len(req.MoveIDs) == 0 {
		return []*dispatchplanprice.DispatchPlanPrice{}, nil
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*dispatchplanprice.DispatchPlanPrice, error) {
			p := dispatchplanpricedb.DispatchPlanPrices
			prices, err := p.Where(p.MoveID.In(req.MoveIDs...)).
				Limit(len(req.MoveIDs)).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list plan prices by move: %w", err)
			}

			return prices, nil
		},
	)
}

func validatePrices(req *dispatchplanports.ReplacePricesRequest) error {
	planned := make(map[pulid.ID]struct{}, len(req.MoveIDs))
	for _, moveID := range req.MoveIDs {
		planned[moveID] = struct{}{}
	}
	for _, price := range req.Prices {
		if price.OrganizationID != req.TenantInfo.OrgID ||
			price.BusinessUnitID != req.TenantInfo.BuID {
			return errortypes.NewBusinessError("A plan price belongs to another organization")
		}
		if _, ok := planned[price.MoveID]; !ok {
			return errortypes.NewBusinessError("A plan price names a move outside the plan")
		}
		if price.GeneratedAt != req.GeneratedAt {
			return errortypes.NewBusinessError("A plan price belongs to another plan")
		}
	}

	return nil
}
