package extractionrolloutrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/extractionrollout"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	assignmentEntity  = "RolloutAssignment"
	defaultPurgeLimit = 1000
	maxPurgeLimit     = 10000
)

type assignmentRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewAssignments(p Params) repositories.RolloutAssignmentRepository {
	return &assignmentRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.extractionrollout-assignment-repository"),
	}
}

func uniqueExtraction() string {
	cols := buncolgen.RolloutAssignmentColumns
	return "CONFLICT (" + strings.Join([]string{
		cols.OrganizationID.Bare(),
		cols.BusinessUnitID.Bare(),
		cols.DocumentID.Bare(),
		cols.ExtractedAt.Bare(),
	}, ", ") + ") DO NOTHING"
}

func (r *assignmentRepository) Create(
	ctx context.Context,
	entity *extractionrollout.RolloutAssignment,
) (*extractionrollout.RolloutAssignment, bool, error) {
	res, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On(uniqueExtraction()).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to create rollout assignment",
			zap.String("documentId", entity.DocumentID.String()),
			zap.Error(err),
		)

		return nil, false, fmt.Errorf("create rollout assignment: %w", err)
	}

	inserted, err := res.RowsAffected()
	if err != nil {
		return nil, false, fmt.Errorf("create rollout assignment rows: %w", err)
	}
	if inserted > 0 {
		return entity, true, nil
	}

	existing, err := r.GetByExtraction(ctx, repositories.GetRolloutAssignmentRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		DocumentID:  entity.DocumentID,
		ExtractedAt: entity.ExtractedAt,
	})
	if err != nil {
		return nil, false, err
	}

	return existing, false, nil
}

func (r *assignmentRepository) GetByExtraction(
	ctx context.Context,
	req repositories.GetRolloutAssignmentRequest,
) (*extractionrollout.RolloutAssignment, error) {
	cols := buncolgen.RolloutAssignmentColumns
	entity := new(extractionrollout.RolloutAssignment)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RolloutAssignmentScopeTenant(sq, req.TenantInfo).
				Where(cols.DocumentID.Eq(), req.DocumentID).
				Where(cols.ExtractedAt.Eq(), req.ExtractedAt)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, assignmentEntity)
	}

	return entity, nil
}

func (r *assignmentRepository) Save(
	ctx context.Context,
	entity *extractionrollout.RolloutAssignment,
) (*extractionrollout.RolloutAssignment, error) {
	cols := buncolgen.RolloutAssignmentColumns
	previous := entity.Version

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.RolloutAssignmentScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).
				Where(cols.ID.Eq(), entity.ID).
				Where(cols.Version.Eq(), previous)
		}).
		ExcludeColumn(
			cols.ID.Bare(),
			cols.OrganizationID.Bare(),
			cols.BusinessUnitID.Bare(),
			cols.DocumentID.Bare(),
			cols.ExtractedAt.Bare(),
			cols.Arm.Bare(),
			cols.CandidateProviderID.Bare(),
			cols.CreatedAt.Bare(),
			cols.Version.Bare(),
		).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to save rollout assignment", zap.Error(err))

		return nil, fmt.Errorf("save rollout assignment: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, assignmentEntity, entity.ID.String()); err != nil {
		return nil, err
	}
	entity.Version = previous + 1

	return entity, nil
}

func (r *assignmentRepository) Totals(
	ctx context.Context,
	req repositories.TotalRolloutAssignmentsRequest,
) ([]repositories.RolloutAssignmentTotal, error) {
	cols := buncolgen.RolloutAssignmentColumns
	totals := make(
		[]repositories.RolloutAssignmentTotal,
		0,
		len(extractionrollout.AllArms())*len(extractionrollout.AllOutcomes())*3,
	)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*extractionrollout.RolloutAssignment)(nil)).
		Column(cols.Arm.Bare(), cols.Outcome.Bare()).
		ColumnExpr(
			buncolgen.Expr(
				"CASE WHEN {0} IS NULL THEN ? WHEN {0} = {1} THEN ? ELSE ? END AS served_by",
				cols.ServedProviderID,
				cols.CandidateProviderID,
			),
			extractionrollout.ServedByNone,
			extractionrollout.ServedByCandidate,
			extractionrollout.ServedByOther,
		).
		ColumnExpr(buncolgen.Count("count")).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RolloutAssignmentScopeTenant(sq, req.TenantInfo).
				Where(cols.CandidateProviderID.Eq(), req.CandidateProviderID).
				Where(cols.CreatedAt.Gte(), req.Since)
		}).
		Group(cols.Arm.Bare(), cols.Outcome.Bare()).
		GroupExpr("served_by").
		Scan(ctx, &totals)
	if err != nil {
		r.l.Error("failed to total rollout assignments", zap.Error(err))

		return nil, fmt.Errorf("total rollout assignments: %w", err)
	}

	return totals, nil
}

func (r *assignmentRepository) PurgeBefore(
	ctx context.Context,
	req repositories.PurgeRolloutAssignmentsRequest,
) (int64, error) {
	cols := buncolgen.RolloutAssignmentColumns
	limit := req.Limit
	if limit <= 0 {
		limit = defaultPurgeLimit
	}
	limit = min(limit, maxPurgeLimit)

	dba := r.db.DBForContext(ctx)
	expired := dba.NewSelect().
		Model((*extractionrollout.RolloutAssignment)(nil)).
		Column(cols.ID.Bare()).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RolloutAssignmentScopeTenant(sq, req.TenantInfo).
				Where(cols.CreatedAt.Lt(), req.Before)
		}).
		OrderExpr(cols.CreatedAt.OrderAsc()).
		Limit(limit)

	res, err := dba.NewDelete().
		Model((*extractionrollout.RolloutAssignment)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.RolloutAssignmentScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.ID.In(), expired)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to purge rollout assignments", zap.Error(err))

		return 0, fmt.Errorf("purge rollout assignments: %w", err)
	}

	purged, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("purge rollout assignments rows: %w", err)
	}

	return purged, nil
}
