package workerinjuryrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety/workersafetydb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workersafetyports"
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

var recordableClassifications = []workersafety.OSHACaseClassification{
	workersafety.CaseOtherRecordable,
	workersafety.CaseJobTransferOrRestriction,
	workersafety.CaseDaysAway,
	workersafety.CaseDeath,
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

func New(p Params) workersafetyports.WorkerInjuryRepository {
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
	i := workersafetydb.WorkerInjuries
	loads := make([]keel.Loader, 0, 3)
	loads = append(loads, listquery.When(includeWorker, i.Worker())...)
	loads = append(loads, listquery.When(includeDocument, i.Document())...)

	return append(loads, listquery.When(includeEvent, i.SafetyEvent())...)
}

func injuriesQuery(
	req *workersafetyports.ListWorkerInjuriesRequest,
) *keel.Query[*workersafety.WorkerInjury] {
	i := workersafetydb.WorkerInjuries

	return i.Where(
		keel.When(req.WorkerID.IsNotNil(), i.WorkerID.Eq(req.WorkerID)),
		keel.When(req.CaseYear > 0, i.CaseYear.Eq(req.CaseYear)),
		keel.When(req.RecordableOnly, i.Classification.In(recordableClassifications...)),
		keel.When(req.OpenOnly, i.Status.Eq(workersafety.InjuryCaseOpen)),
	).
		Load(injuryLoads(req.IncludeWorker, req.IncludeDocument, req.IncludeEvent)...).
		Order(i.CaseYear.Desc(), i.CaseNumber.Desc()).
		Limit(limitOr(req.Limit, defaultInjuryPageSize))
}

func (r *repository) ListInjuries(
	ctx context.Context,
	req *workersafetyports.ListWorkerInjuriesRequest,
) ([]*workersafety.WorkerInjury, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*workersafety.WorkerInjury, error) {
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
	req *workersafetyports.GetWorkerInjuryByIDRequest,
) (*workersafety.WorkerInjury, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*workersafety.WorkerInjury, error) {
		i := workersafetydb.WorkerInjuries
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
	entity *workersafety.WorkerInjury,
) (*workersafety.WorkerInjury, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*workersafety.WorkerInjury, error) {
		if err := workersafetydb.WorkerInjuries.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create injury", zap.Error(err))
			return nil, injuryErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateInjury(
	ctx context.Context,
	entity *workersafety.WorkerInjury,
) (*workersafety.WorkerInjury, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*workersafety.WorkerInjury, error) {
		if err := workersafetydb.WorkerInjuries.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update injury", zap.Error(err))
			return nil, injuryErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) DeleteInjury(
	ctx context.Context,
	req *workersafetyports.GetWorkerInjuryByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := workersafetydb.WorkerInjuries.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&workersafety.WorkerInjury{ID: req.ID},
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
	req *workersafetyports.NextCaseNumberRequest,
) (int32, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int32, error) {
		i := workersafetydb.WorkerInjuries
		highest, err := keel.IntoOne[int32](ctx, r.db.Keel(ctx), i.
			Pick(keel.Int32Of(i.CaseNumber.Max()).Coalesce(0)).
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
	req *workersafetyports.GetOSHASummaryRequest,
) (*workersafety.OSHAAnnualSummary, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*workersafety.OSHAAnnualSummary, error) {
		s := workersafetydb.OSHAAnnualSummaries
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
	req *workersafetyports.ListOSHASummariesRequest,
) ([]*workersafety.OSHAAnnualSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*workersafety.OSHAAnnualSummary, error) {
			s := workersafetydb.OSHAAnnualSummaries
			entities, err := s.Query().
				Order(s.Year.Desc()).
				Limit(limitOr(req.Limit, defaultSummaryPageSize)).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list osha summaries", zap.Error(err))
				return nil, fmt.Errorf("list osha summaries: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *repository) CreateSummary(
	ctx context.Context,
	entity *workersafety.OSHAAnnualSummary,
) (*workersafety.OSHAAnnualSummary, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workersafety.OSHAAnnualSummary, error) {
			if err := workersafetydb.OSHAAnnualSummaries.Create(ctx, r.db.Keel(ctx), entity); err != nil {
				r.l.Error("failed to create osha summary", zap.Error(err))
				return nil, summaryErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateSummary(
	ctx context.Context,
	entity *workersafety.OSHAAnnualSummary,
) (*workersafety.OSHAAnnualSummary, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workersafety.OSHAAnnualSummary, error) {
			if err := workersafetydb.OSHAAnnualSummaries.Save(ctx, r.db.Keel(ctx), entity); err != nil {
				r.l.Error("failed to update osha summary", zap.Error(err))
				return nil, summaryErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}
