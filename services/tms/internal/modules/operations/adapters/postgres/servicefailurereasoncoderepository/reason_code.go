package servicefailurereasoncoderepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure/servicefailuredb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/keelkinds"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const reasonCodeEntity = "Service failure reason code"

var reasonCodeErrors = dberror.Mapping{Entity: reasonCodeEntity}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) servicefailureports.ServiceFailureReasonCodeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.service-failure-reason-code-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *servicefailureports.ListServiceFailureReasonCodesRequest,
) (*pagination.ListResult[*servicefailure.ReasonCode], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*servicefailure.ReasonCode], error) {
			if req.Filter != nil && req.Filter.Pagination.Limit <= 0 {
				req.Filter.Pagination.Limit = 50
			}

			rc := servicefailuredb.ReasonCodes
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				rc.Query(),
				&listquery.Request[servicefailure.ReasonCode]{
					Filter: req.Filter,
					Then:   []keel.Order{rc.SortOrder.Asc(), rc.Code.Asc()},
				},
			)
			if err != nil {
				return nil, fmt.Errorf("list service failure reason codes: %w", err)
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *servicefailureports.ListServiceFailureReasonCodeConnectionRequest,
) (*pagination.CursorListResult[*servicefailure.ReasonCode], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*servicefailure.ReasonCode], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				servicefailuredb.ReasonCodes.Query(),
				&listquery.Request[servicefailure.ReasonCode]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.ServiceFailureReasonCodeColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list service failure reason codes", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req servicefailureports.GetServiceFailureReasonCodeByIDRequest,
) (*servicefailure.ReasonCode, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*servicefailure.ReasonCode, error) {
		return r.get(ctx, req.ID)
	})
}

func (r *repository) get(ctx context.Context, id pulid.ID) (*servicefailure.ReasonCode, error) {
	rc := servicefailuredb.ReasonCodes
	entity, err := rc.Where(rc.ID.Eq(id)).
		Load(rc.ArchivedBy(), rc.ActivatedBy()).
		One(ctx, r.db.Keel(ctx))
	if err != nil {
		return nil, reasonCodeErrors.Map(err, id.String())
	}

	return entity, nil
}

func (r *repository) FindDefault(
	ctx context.Context,
	_ pagination.TenantInfo,
	appliesTo servicefailure.ReasonCodeAppliesTo,
) (*servicefailure.ReasonCode, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*servicefailure.ReasonCode, error) {
		rc := servicefailuredb.ReasonCodes
		entity, err := rc.Where(
			rc.Active.IsTrue(),
			rc.AppliesTo.In(
				appliesTo,
				servicefailure.ReasonCodeAppliesToBoth,
				servicefailure.ReasonCodeAppliesToAll,
			),
		).
			Order(
				keel.Desc(rc.AppliesTo.Eq(appliesTo)),
				rc.SortOrder.Asc(),
				rc.Code.Asc(),
			).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, reasonCodeErrors.Map(err, string(appliesTo))
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *servicefailure.ReasonCode,
) (*servicefailure.ReasonCode, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*servicefailure.ReasonCode, error) {
		if err := servicefailuredb.ReasonCodes.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, reasonCodeErrors.Map(err, entity.ID.String())
		}

		return r.get(ctx, entity.ID)
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *servicefailure.ReasonCode,
) (*servicefailure.ReasonCode, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*servicefailure.ReasonCode, error) {
		rc := servicefailuredb.ReasonCodes
		if err := rc.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			rc.Code,
			rc.Label,
			rc.Description,
			rc.Category,
			rc.AppliesTo,
			rc.DefaultStatusCode,
			rc.DefaultReasonCode,
			rc.DefaultExceptionCode,
			rc.DefaultNote,
			rc.Active,
			rc.SortOrder,
			rc.ExternalMap,
		); err != nil {
			return nil, reasonCodeErrors.Map(err, entity.ID.String())
		}

		return r.get(ctx, entity.ID)
	})
}

func (r *repository) Archive(
	ctx context.Context,
	id pulid.ID,
	_ pagination.TenantInfo,
	actorID pulid.ID,
) (*servicefailure.ReasonCode, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*servicefailure.ReasonCode, error) {
		rc := servicefailuredb.ReasonCodes
		n, err := rc.Where(rc.ID.Eq(id)).Update(
			ctx,
			r.db.Keel(ctx),
			rc.Active.To(false),
			rc.ArchivedAt.To(timeutils.NowUnix()),
			actorAssignment(rc.ArchivedByID, actorID),
			rc.ActivatedAt.ToNull(),
			rc.ActivatedByID.ToNull(),
		)
		if err != nil {
			return nil, fmt.Errorf("archive service failure reason code: %w", err)
		}
		if err = dberror.CheckAffected(n, reasonCodeEntity, id.String()); err != nil {
			return nil, err
		}

		return r.get(ctx, id)
	})
}

func (r *repository) Activate(
	ctx context.Context,
	id pulid.ID,
	_ pagination.TenantInfo,
	actorID pulid.ID,
) (*servicefailure.ReasonCode, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*servicefailure.ReasonCode, error) {
		rc := servicefailuredb.ReasonCodes
		n, err := rc.Where(rc.ID.Eq(id)).Update(
			ctx,
			r.db.Keel(ctx),
			rc.Active.To(true),
			rc.ActivatedAt.To(timeutils.NowUnix()),
			actorAssignment(rc.ActivatedByID, actorID),
		)
		if err != nil {
			return nil, fmt.Errorf("activate service failure reason code: %w", err)
		}
		if err = dberror.CheckAffected(n, reasonCodeEntity, id.String()); err != nil {
			return nil, err
		}

		return r.get(ctx, id)
	})
}

func (r *repository) Reorder(
	ctx context.Context,
	req *servicefailureports.ReorderServiceFailureReasonCodesRequest,
) ([]*servicefailure.ReasonCode, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*servicefailure.ReasonCode, error) {
		if len(req.ReasonIDs) == 0 {
			return []*servicefailure.ReasonCode{}, nil
		}

		ids := make([]string, len(req.ReasonIDs))
		orders := make([]int32, len(req.ReasonIDs))
		for idx, id := range req.ReasonIDs {
			ids[idx] = id.String()
			orders[idx] = int32((idx + 1) * 10)
		}

		rc := servicefailuredb.ReasonCodes
		db := r.db.Keel(ctx)
		positions := keel.Unnest("positions", []string{"id", "sort_order"}, ids, orders)
		if _, err := rc.With(positions).
			Join(positions, positions.Col("id").EqCol(rc.ID)).
			Where(keel.True()).
			Update(ctx, db, rc.SortOrder.ToExpr(positions.Col("sort_order"))); err != nil {
			return nil, fmt.Errorf("reorder service failure reason codes: %w", err)
		}

		entities, err := rc.Where(rc.ID.In(req.ReasonIDs...)).
			Order(rc.SortOrder.Asc()).
			All(ctx, db)
		if err != nil {
			return nil, fmt.Errorf("list reordered service failure reason codes: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *servicefailureports.ServiceFailureReasonCodeSelectOptionsRequest,
) (*pagination.ListResult[*servicefailure.ReasonCode], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*servicefailure.ReasonCode], error) {
			rc := servicefailuredb.ReasonCodes
			q := rc.Pick(
				rc.ID,
				rc.Code,
				rc.Label,
				rc.Category,
				rc.AppliesTo,
				rc.DefaultStatusCode,
				rc.DefaultReasonCode,
			).
				Where(
					rc.Active.IsTrue(),
					keel.When(req.AppliesTo.IsValid(), rc.AppliesTo.In(
						req.AppliesTo,
						servicefailure.ReasonCodeAppliesToBoth,
						servicefailure.ReasonCodeAppliesToAll,
					)),
				).
				Order(rc.SortOrder.Asc(), rc.Code.Asc())

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(
						rc.Code.ContainsFold(s),
						rc.Label.ContainsFold(s),
						rc.Description.ContainsFold(s),
					)
				},
			)
		},
	)
}

func actorAssignment(col keelkinds.PulidIDCol, actorID pulid.ID) keel.Assignment {
	if actorID.IsNil() {
		return col.ToNull()
	}

	return col.To(actorID)
}
