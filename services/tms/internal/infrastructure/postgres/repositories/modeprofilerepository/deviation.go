package modeprofilerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type DeviationParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type deviationRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewDeviationRepository(p DeviationParams) repositories.DeviationRepository {
	return &deviationRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.mode-profile-deviation-repository"),
	}
}

func (r *deviationRepository) RecordMany(
	ctx context.Context,
	deviations []*modeprofile.Deviation,
) error {
	if len(deviations) == 0 {
		return nil
	}

	log := r.l.With(
		zap.String("operation", "RecordMany"),
		zap.Int("count", len(deviations)),
	)

	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&deviations).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error("failed to record deviations", zap.Error(err))
		return err
	}

	return nil
}

func (r *deviationRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListDeviationsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.DeviationTable.Alias,
		req.Filter,
		(*modeprofile.Deviation)(nil),
	)

	cols := buncolgen.DeviationColumns

	if req.ResourceType != "" {
		q = q.Where(cols.ResourceType.Eq(), req.ResourceType)
	}
	if req.ResourceID.IsNotNil() {
		q = q.Where(cols.ResourceID.Eq(), req.ResourceID)
	}
	if req.RuleKey != "" {
		q = q.Where(cols.RuleKey.Eq(), req.RuleKey)
	}
	if req.State != "" {
		q = q.Where(cols.State.Eq(), req.State)
	}

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *deviationRepository) List(
	ctx context.Context,
	req *repositories.ListDeviationsRequest,
) (*pagination.ListResult[*modeprofile.Deviation], error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*modeprofile.Deviation, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.DeviationRelations.Profile).
		Relation(buncolgen.DeviationRelations.AcknowledgedBy).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		Order(buncolgen.DeviationColumns.OccurredAt.OrderDesc()).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count deviations", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*modeprofile.Deviation]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *deviationRepository) GetByID(
	ctx context.Context,
	req *repositories.GetDeviationByIDRequest,
) (*modeprofile.Deviation, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.DeviationID.String()),
	)

	entity := new(modeprofile.Deviation)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.DeviationRelations.Profile).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DeviationScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.DeviationColumns.ID.Eq(), req.DeviationID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get deviation", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *deviationRepository) Acknowledge(
	ctx context.Context,
	req *repositories.AcknowledgeDeviationRequest,
) (*modeprofile.Deviation, error) {
	log := r.l.With(
		zap.String("operation", "Acknowledge"),
		zap.String("id", req.DeviationID.String()),
	)

	entity, err := r.GetByID(ctx, &repositories.GetDeviationByIDRequest{
		DeviationID: req.DeviationID,
		TenantInfo:  req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	acknowledgedBy := req.AcknowledgedByID

	entity.State = modeprofile.DeviationStateAcknowledged
	entity.AcknowledgedByID = &acknowledgedBy
	entity.AcknowledgedAt = &now
	entity.AcknowledgementReason = req.Reason

	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to acknowledge deviation", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(
		results, "Deviation", entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *deviationRepository) SupersedeOpenForResource(
	ctx context.Context,
	req *repositories.SupersedeDeviationsRequest,
) error {
	log := r.l.With(
		zap.String("operation", "SupersedeOpenForResource"),
		zap.String("resourceId", req.ResourceID.String()),
	)

	cols := buncolgen.DeviationColumns

	q := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*modeprofile.Deviation)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.DeviationScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ResourceType.Eq(), req.ResourceType).
				Where(cols.ResourceID.Eq(), req.ResourceID).
				Where(cols.State.Eq(), modeprofile.DeviationStateOpen)
		}).
		Set(cols.State.Set(), modeprofile.DeviationStateResolved).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix())

	if len(req.KeepRuleKeys) > 0 {
		q = q.Where(cols.RuleKey.NotIn(), bun.List(req.KeepRuleKeys))
	}

	if _, err := q.Exec(ctx); err != nil {
		log.Error("failed to supersede open deviations", zap.Error(err))
		return err
	}

	return nil
}

func (r *deviationRepository) LiveRuleKeysForResource(
	ctx context.Context,
	req *repositories.LiveDeviationKeysRequest,
) ([]string, error) {
	log := r.l.With(
		zap.String("operation", "LiveRuleKeysForResource"),
		zap.String("resourceId", req.ResourceID.String()),
	)

	cols := buncolgen.DeviationColumns
	keys := make([]string, 0, 8)

	if err := buncolgen.DeviationScopeTenant(
		r.db.DBForContext(ctx).NewSelect().Model((*modeprofile.Deviation)(nil)),
		req.TenantInfo,
	).
		Column(cols.RuleKey.Name).
		Distinct().
		Where(cols.ResourceType.Eq(), req.ResourceType).
		Where(cols.ResourceID.Eq(), req.ResourceID).
		Where(cols.State.In(), bun.List([]modeprofile.DeviationState{
			modeprofile.DeviationStateOpen,
			modeprofile.DeviationStateAcknowledged,
		})).
		Scan(ctx, &keys); err != nil {
		log.Error("failed to read live deviation rule keys", zap.Error(err))
		return nil, err
	}

	return keys, nil
}

func (r *deviationRepository) Ledger(
	ctx context.Context,
	req *repositories.DeviationLedgerRequest,
) ([]*repositories.DeviationLedgerEntry, error) {
	log := r.l.With(zap.String("operation", "Ledger"))

	cols := buncolgen.DeviationColumns
	entries := make([]*repositories.DeviationLedgerEntry, 0, 16)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*modeprofile.Deviation)(nil)).
		ColumnExpr(cols.RuleKey.As("rule_key")).
		ColumnExpr(cols.Capability.As("capability")).
		ColumnExpr(buncolgen.Count("occurrences")).
		ColumnExpr(
			buncolgen.CountFilter("open_count", cols.State.Eq()),
			modeprofile.DeviationStateOpen,
		).
		ColumnExpr(buncolgen.Max(cols.OccurredAt, "last_occurred_at")).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DeviationScopeTenant(sq, req.TenantInfo)
		}).
		GroupExpr(cols.RuleKey.Qualified()).
		GroupExpr(cols.Capability.Qualified()).
		OrderExpr("occurrences DESC")

	if req.Since > 0 {
		q = q.Where(cols.OccurredAt.Gte(), req.Since)
	}

	if err := q.Scan(ctx, &entries); err != nil {
		log.Error("failed to build deviation ledger", zap.Error(err))
		return nil, err
	}

	for _, entry := range entries {
		if def, err := modeprofile.RuleDefinitionFor(
			modeprofile.RuleKey(entry.RuleKey),
		); err == nil {
			entry.Label = def.Label
		} else {
			entry.Label = entry.RuleKey
		}
	}

	return entries, nil
}
