package workerdqfrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultVerificationPageSize = 200
	defaultRetentionPageSize    = 200
)

var verificationErrors = dberror.Mapping{
	Entity: "WorkerEmploymentVerification",
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_worker_employment_verifications_worker",
			Field:   "workerId",
			Message: "Worker does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_worker_employment_verifications_document",
			Field:   "documentId",
			Message: "Document does not exist or belongs to a different organization",
		},
	},
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) workerports.WorkerDQFRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-dqf-repository"),
	}
}

func limitOr(requested, fallback int) int {
	if requested > 0 && requested <= fallback {
		return requested
	}
	return fallback
}

func verificationLoads(includeWorker, includeDocument bool) []keel.Loader {
	v := workerdb.WorkerEmploymentVerifications

	return append(
		listquery.When(includeWorker, v.Worker()),
		listquery.When(includeDocument, v.Document())...,
	)
}

func verificationOrder() []keel.Order {
	v := workerdb.WorkerEmploymentVerifications
	return []keel.Order{v.EmployedTo.Desc().NullsLast(), v.CreatedAt.Desc()}
}

func (r *repository) ListVerifications(
	ctx context.Context,
	req *workerports.ListEmploymentVerificationsRequest,
) ([]*worker.WorkerEmploymentVerification, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*worker.WorkerEmploymentVerification, error) {
			v := workerdb.WorkerEmploymentVerifications
			entities, err := v.Where(
				keel.When(req.WorkerID.IsNotNil(), v.WorkerID.Eq(req.WorkerID)),
				keel.When(req.OutstandingOnly, v.Status.In(
					worker.VerificationPending,
					worker.VerificationRequested,
				)),
			).
				Load(verificationLoads(req.IncludeWorker, req.IncludeDocument)...).
				Order(verificationOrder()...).
				Limit(limitOr(req.Limit, defaultVerificationPageSize)).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list employment verifications", zap.Error(err))
				return nil, fmt.Errorf("list employment verifications: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *repository) ListVerificationsByWorkerIDs(
	ctx context.Context,
	req *workerports.ListEmploymentVerificationsByWorkerIDsRequest,
) (map[pulid.ID][]*worker.WorkerEmploymentVerification, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*worker.WorkerEmploymentVerification, error) {
			if len(req.WorkerIDs) == 0 {
				return map[pulid.ID][]*worker.WorkerEmploymentVerification{}, nil
			}

			v := workerdb.WorkerEmploymentVerifications
			entities, err := v.Where(v.WorkerID.In(req.WorkerIDs...)).
				Load(verificationLoads(req.IncludeWorker, req.IncludeDocument)...).
				Order(verificationOrder()...).
				Limit(len(req.WorkerIDs)*defaultVerificationPageSize).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list employment verifications by workers", zap.Error(err))
				return nil, fmt.Errorf("list employment verifications by workers: %w", err)
			}

			return sliceutils.GroupBy(
				entities,
				func(v *worker.WorkerEmploymentVerification) pulid.ID {
					return v.WorkerID
				},
			), nil
		},
	)
}

func (r *repository) GetVerificationByID(
	ctx context.Context,
	req *workerports.GetEmploymentVerificationByIDRequest,
) (*worker.WorkerEmploymentVerification, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*worker.WorkerEmploymentVerification, error) {
			v := workerdb.WorkerEmploymentVerifications
			entity, err := v.Where(v.ID.Eq(req.ID)).
				Load(verificationLoads(false, req.IncludeDocument)...).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, verificationErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) CreateVerification(
	ctx context.Context,
	entity *worker.WorkerEmploymentVerification,
) (*worker.WorkerEmploymentVerification, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*worker.WorkerEmploymentVerification, error) {
			err := workerdb.WorkerEmploymentVerifications.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to create employment verification", zap.Error(err))
				return nil, verificationErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateVerification(
	ctx context.Context,
	entity *worker.WorkerEmploymentVerification,
) (*worker.WorkerEmploymentVerification, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*worker.WorkerEmploymentVerification, error) {
			err := workerdb.WorkerEmploymentVerifications.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to update employment verification", zap.Error(err))
				return nil, verificationErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) DeleteVerification(
	ctx context.Context,
	req *workerports.GetEmploymentVerificationByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := workerdb.WorkerEmploymentVerifications.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&worker.WorkerEmploymentVerification{ID: req.ID},
		)
		if err != nil {
			r.l.Error("failed to delete employment verification", zap.Error(err))
			return verificationErrors.Map(err, req.ID.String())
		}

		return nil
	})
}

// ListRetentionCandidates finds terminated drivers whose file has passed its
// hold period. The window is applied in SQL so the caller reads the handful
// that qualify rather than every worker who ever left.
func (r *repository) ListRetentionCandidates(
	ctx context.Context,
	req *workerports.ListDQFRetentionCandidatesRequest,
) ([]workerports.DQFRetentionCandidate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]workerports.DQFRetentionCandidate, error) {
			candidates, err := keel.Into[workerports.DQFRetentionCandidate](
				ctx,
				r.db.Keel(ctx),
				retentionQuery(req),
			)
			if err != nil {
				r.l.Error("failed to list dqf retention candidates", zap.Error(err))
				return nil, fmt.Errorf("list dqf retention candidates: %w", err)
			}

			return candidates, nil
		},
	)
}

func retentionQuery(req *workerports.ListDQFRetentionCandidatesRequest) *keel.Query[worker.Worker] {
	w := workerdb.Workers
	wp := workerdb.WorkerProfiles

	return w.Join(wp, wp.WorkerID.EqCol(w.ID)).
		Pick(w.ID, w.OrganizationID, w.BusinessUnitID, w.FirstName, w.LastName, wp.TerminationDate).
		Where(
			wp.TerminationDate.IsNotNull(),
			keel.DynOf(wp.TerminationDate).Add(req.RetentionSeconds).Lte(req.AsOf),
		).
		Order(w.ID.Asc()).
		Limit(limitOr(req.Limit, defaultRetentionPageSize))
}
