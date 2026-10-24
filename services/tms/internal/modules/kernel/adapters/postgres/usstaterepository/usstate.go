package usstaterepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/usstateports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var usStateErrors = dberror.Mapping{Entity: "UsState"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) usstateports.UsStateRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.us-state-repository"),
	}
}

func (r *repository) GetByID(
	ctx context.Context,
	req usstateports.GetUsStateByIDRequest,
) (*usstate.UsState, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*usstate.UsState, error) {
		entity, err := usstatedb.UsStates.Get(ctx, r.db.Keel(ctx), req.StateID)
		if err != nil {
			r.l.Error(
				"failed to get us state",
				zap.String("id", req.StateID.String()),
				zap.Error(err),
			)
			return nil, usStateErrors.Map(err, req.StateID.String())
		}

		return entity, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*usstate.UsState], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*usstate.UsState], error) {
			us := usstatedb.UsStates
			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				us.Pick(us.ID, us.CreatedAt, us.Name, us.Abbreviation, us.CountryIso3),
				req,
				func(s string) keel.Pred {
					return keel.Or(us.Name.ContainsFold(s), us.Abbreviation.ContainsFold(s))
				},
			)
		},
	)
}

func (r *repository) GetByAbbreviation(
	ctx context.Context,
	abbreviation string,
) (*usstate.UsState, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*usstate.UsState, error) {
		us := usstatedb.UsStates
		normalized := strings.ToUpper(strings.TrimSpace(abbreviation))
		entity, err := us.Where(us.Abbreviation.Eq(normalized)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Debug("failed to get us state by abbreviation",
				zap.String("abbreviation", abbreviation),
				zap.Error(err),
			)
			return nil, usStateErrors.Map(err, normalized)
		}

		return entity, nil
	})
}

func (r *repository) GetByIDs(ctx context.Context, ids []pulid.ID) ([]*usstate.UsState, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*usstate.UsState, error) {
		if len(ids) == 0 {
			return []*usstate.UsState{}, nil
		}

		us := usstatedb.UsStates
		entities, err := us.Where(us.ID.In(ids...)).Limit(len(ids)).List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get us states by ids", zap.Error(err))
			return nil, fmt.Errorf("get us states by ids: %w", err)
		}

		return entities, nil
	})
}
