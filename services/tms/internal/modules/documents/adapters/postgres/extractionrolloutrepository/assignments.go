package extractionrolloutrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractionrollout"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractionrollout/extractionrolloutdb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionrolloutports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	defaultPurgeLimit = 1000
	maxPurgeLimit     = 10000
)

var assignmentErrors = dberror.Mapping{Entity: "RolloutAssignment"}

type assignmentRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewAssignments(p Params) extractionrolloutports.RolloutAssignmentRepository {
	return &assignmentRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.extractionrollout-assignment-repository"),
	}
}

func (r *assignmentRepository) Create(
	ctx context.Context,
	entity *extractionrollout.RolloutAssignment,
) (*extractionrollout.RolloutAssignment, bool, error) {
	return dbtx.Write2(
		ctx,
		r.db,
		func(ctx context.Context) (*extractionrollout.RolloutAssignment, bool, error) {
			ra := extractionrolloutdb.RolloutAssignments
			db := r.db.Keel(ctx)

			inserted, err := ra.CreateOrSkip(
				ctx,
				db,
				[]*extractionrollout.RolloutAssignment{entity},
				ra.OrganizationID,
				ra.BusinessUnitID,
				ra.DocumentID,
				ra.ExtractedAt,
			)
			if err != nil {
				r.l.Error("failed to create rollout assignment",
					zap.String("documentId", entity.DocumentID.String()),
					zap.Error(err),
				)
				return nil, false, fmt.Errorf("create rollout assignment: %w", err)
			}
			if inserted > 0 {
				return entity, true, nil
			}

			existing, err := byExtraction(entity.DocumentID, entity.ExtractedAt).One(ctx, db)
			if err != nil {
				return nil, false, assignmentErrors.Map(err, entity.DocumentID.String())
			}

			return existing, false, nil
		},
	)
}

func (r *assignmentRepository) GetByExtraction(
	ctx context.Context,
	req extractionrolloutports.GetRolloutAssignmentRequest,
) (*extractionrollout.RolloutAssignment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*extractionrollout.RolloutAssignment, error) {
			entity, err := byExtraction(req.DocumentID, req.ExtractedAt).One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, assignmentErrors.Map(err, req.DocumentID.String())
			}

			return entity, nil
		},
	)
}

func byExtraction(
	documentID pulid.ID,
	extractedAt int64,
) *keel.Query[*extractionrollout.RolloutAssignment] {
	ra := extractionrolloutdb.RolloutAssignments

	return ra.Where(ra.DocumentID.Eq(documentID), ra.ExtractedAt.Eq(extractedAt))
}

func (r *assignmentRepository) Save(
	ctx context.Context,
	entity *extractionrollout.RolloutAssignment,
) (*extractionrollout.RolloutAssignment, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*extractionrollout.RolloutAssignment, error) {
			ra := extractionrolloutdb.RolloutAssignments
			if err := ra.SaveExcept(
				ctx,
				r.db.Keel(ctx),
				entity,
				ra.DocumentID,
				ra.ExtractedAt,
				ra.Arm,
				ra.CandidateProviderID,
			); err != nil {
				r.l.Error("failed to save rollout assignment", zap.Error(err))
				return nil, assignmentErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *assignmentRepository) Totals(
	ctx context.Context,
	req extractionrolloutports.TotalRolloutAssignmentsRequest,
) ([]extractionrolloutports.RolloutAssignmentTotal, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]extractionrolloutports.RolloutAssignmentTotal, error) {
			ra := extractionrolloutdb.RolloutAssignments
			servedBy := keel.DynCase().
				When(
					ra.ServedProviderID.IsNull(),
					keel.Text(string(extractionrollout.ServedByNone)),
				).
				When(
					ra.ServedProviderID.EqCol(ra.CandidateProviderID),
					keel.Text(string(extractionrollout.ServedByCandidate)),
				).
				Else(keel.Text(string(extractionrollout.ServedByOther)))

			totals, err := keel.Into[extractionrolloutports.RolloutAssignmentTotal](
				ctx,
				r.db.Keel(ctx),
				ra.Pick(ra.Arm, servedBy, ra.Outcome, keel.Count()).
					Where(
						ra.CandidateProviderID.Eq(req.CandidateProviderID),
						ra.CreatedAt.Gte(req.Since),
					).
					Group(ra.Arm, ra.Outcome, servedBy),
			)
			if err != nil {
				r.l.Error("failed to total rollout assignments", zap.Error(err))
				return nil, fmt.Errorf("total rollout assignments: %w", err)
			}
			if totals == nil {
				totals = []extractionrolloutports.RolloutAssignmentTotal{}
			}

			return totals, nil
		},
	)
}

func (r *assignmentRepository) PurgeBefore(
	ctx context.Context,
	req extractionrolloutports.PurgeRolloutAssignmentsRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultPurgeLimit
		}
		limit = min(limit, maxPurgeLimit)

		ra := extractionrolloutdb.RolloutAssignments
		expired := ra.As("expired")
		purged, err := ra.Where(ra.ID.InQuery(expired.
			Pick(expired.ID).
			Where(expired.CreatedAt.Lt(req.Before)).
			Order(expired.CreatedAt.Asc()).
			Limit(limit),
		)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to purge rollout assignments", zap.Error(err))
			return 0, fmt.Errorf("purge rollout assignments: %w", err)
		}

		return purged, nil
	})
}
