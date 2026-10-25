package workerinjuryrepository

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
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultInjuryPageSize  = 500
	defaultSummaryPageSize = 20
)

var injuryErrors = dberror.Mapping{Entity: "WorkerInjury"}

var summaryErrors = dberror.Mapping{Entity: "OSHAAnnualSummary"}

var recordableClassifications = []worker.OSHACaseClassification{
	worker.CaseOtherRecordable,
	worker.CaseJobTransferOrRestriction,
	worker.CaseDaysAway,
	worker.CaseDeath,
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

func New(p Params) workerports.WorkerInjuryRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-injury-repository"),
	}
}

func limitOr(requested, fallback int) int {
	if requested > 0 && requested <= fallback {
		return requested
	}
	return fallback
}

func injuryLoads(includeWorker, includeDocument, includeEvent bool) []keel.Loader {
	i := workerdb.WorkerInjuries
	loads := make([]keel.Loader, 0, 3)
	loads = append(loads, listquery.When(includeWorker, i.Worker())...)
	loads = append(loads, listquery.When(includeDocument, i.Document())...)

	return append(loads, listquery.When(includeEvent, i.SafetyEvent())...)
}

func injuriesQuery(req *workerports.ListWorkerInjuriesRequest) *keel.Query[*worker.WorkerInjury] {
	i := workerdb.WorkerInjuries

	return i.Where(
		keel.When(req.WorkerID.IsNotNil(), i.WorkerID.Eq(req.WorkerID)),
		keel.When(req.CaseYear > 0, i.CaseYear.Eq(req.CaseYear)),
		keel.When(req.RecordableOnly, i.Classification.In(recordableClassifications...)),
		keel.When(req.OpenOnly, i.Status.Eq(worker.InjuryCaseOpen)),
	).
		Load(injuryLoads(req.IncludeWorker, req.IncludeDocument, req.IncludeEvent)...).
		Order(i.CaseYear.Desc(), i.CaseNumber.Desc()).
		Limit(limitOr(req.Limit, defaultInjuryPageSize))
}

func (r *repository) ListInjuries(
	ctx context.Context,
	req *workerports.ListWorkerInjuriesRequest,
) ([]*worker.WorkerInjury, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.WorkerInjury, error) {
		entities, err := injuriesQuery(req).List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list injuries", zap.Error(err))
			return nil, fmt.Errorf("list injuries: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) GetInjuryByID(
	ctx context.Context,
	req *workerports.GetWorkerInjuryByIDRequest,
) (*worker.WorkerInjury, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.WorkerInjury, error) {
		i := workerdb.WorkerInjuries
		entity, err := i.Where(i.ID.Eq(req.ID)).
			Load(injuryLoads(req.IncludeWorker, req.IncludeDocument, req.IncludeEvent)...).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, injuryErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateInjury(
	ctx context.Context,
	entity *worker.WorkerInjury,
) (*worker.WorkerInjury, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerInjury, error) {
		if err := workerdb.WorkerInjuries.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create injury", zap.Error(err))
			return nil, injuryErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateInjury(
	ctx context.Context,
	entity *worker.WorkerInjury,
) (*worker.WorkerInjury, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerInjury, error) {
		if err := workerdb.WorkerInjuries.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update injury", zap.Error(err))
			return nil, injuryErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) DeleteInjury(
	ctx context.Context,
	req *workerports.GetWorkerInjuryByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := workerdb.WorkerInjuries.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&worker.WorkerInjury{ID: req.ID},
		)
		if err != nil {
			r.l.Error("failed to delete injury", zap.Error(err))
			return injuryErrors.Map(err, req.ID.String())
		}

		return nil
	})
}

// NextCaseNumber is the next line on the log for the year. It reads the highest
// number rather than counting rows, so deleting a case recorded in error does
// not hand its number to the next one — an auditor reading a log with two
// case 4s has no way to tell which is which.
func (r *repository) NextCaseNumber(
	ctx context.Context,
	req *workerports.NextCaseNumberRequest,
) (int32, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int32, error) {
		i := workerdb.WorkerInjuries
		highest, err := keel.IntoOne[int32](ctx, r.db.Keel(ctx), i.
			Pick(keel.DynOf(i.CaseNumber).Max().Coalesce(0)).
			Where(i.CaseYear.Eq(req.CaseYear)))
		if err != nil {
			r.l.Error("failed to read the next case number", zap.Error(err))
			return 0, fmt.Errorf("read next case number: %w", err)
		}

		return highest + 1, nil
	})
}

func (r *repository) GetSummary(
	ctx context.Context,
	req *workerports.GetOSHASummaryRequest,
) (*worker.OSHAAnnualSummary, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.OSHAAnnualSummary, error) {
		s := workerdb.OSHAAnnualSummaries
		entity, err := s.Where(s.Year.Eq(req.Year)).
			Load(s.CertifiedBy()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}
			r.l.Error("failed to read the osha summary", zap.Error(err))
			return nil, fmt.Errorf("read osha summary: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) ListSummaries(
	ctx context.Context,
	req *workerports.ListOSHASummariesRequest,
) ([]*worker.OSHAAnnualSummary, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.OSHAAnnualSummary, error) {
		s := workerdb.OSHAAnnualSummaries
		entities, err := s.Query().
			Order(s.Year.Desc()).
			Limit(limitOr(req.Limit, defaultSummaryPageSize)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list osha summaries", zap.Error(err))
			return nil, fmt.Errorf("list osha summaries: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) CreateSummary(
	ctx context.Context,
	entity *worker.OSHAAnnualSummary,
) (*worker.OSHAAnnualSummary, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.OSHAAnnualSummary, error) {
		if err := workerdb.OSHAAnnualSummaries.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create osha summary", zap.Error(err))
			return nil, summaryErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateSummary(
	ctx context.Context,
	entity *worker.OSHAAnnualSummary,
) (*worker.OSHAAnnualSummary, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.OSHAAnnualSummary, error) {
		if err := workerdb.OSHAAnnualSummaries.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update osha summary", zap.Error(err))
			return nil, summaryErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}
