package schedulingrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule/workerscheduledb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerscheduleports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultTemplatePageSize   = 200
	defaultAssignmentPageSize = 200
	defaultSwapPageSize       = 200
)

var templateErrors = dberror.Mapping{Entity: "ShiftTemplate"}

var assignmentErrors = dberror.Mapping{Entity: "WorkerShiftAssignment"}

var preferenceErrors = dberror.Mapping{Entity: "WorkerAvailabilityPreference"}

var swapErrors = dberror.Mapping{Entity: "ShiftSwapRequest"}

type templateCount struct {
	TemplateID pulid.ID
	Count      int64
}

type Params struct {
	fx.In

	DB          *postgres.Connection
	Logger      *zap.Logger
	Assignments workerscheduleports.RotaAssignments
}

type repository struct {
	db          *postgres.Connection
	l           *zap.Logger
	assignments workerscheduleports.RotaAssignments
}

func New(p Params) workerscheduleports.SchedulingRepository {
	return &repository{
		db:          p.DB,
		l:           p.Logger.Named("postgres.scheduling-repository"),
		assignments: p.Assignments,
	}
}

func limitOr(requested, fallback int) int {
	if requested > 0 && requested <= fallback {
		return requested
	}

	return fallback
}

func (r *repository) ListTemplates(
	ctx context.Context,
	req *workerscheduleports.ListShiftTemplatesRequest,
) ([]*workerschedule.ShiftTemplate, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*workerschedule.ShiftTemplate, error) {
		t := workerscheduledb.ShiftTemplates
		entities, err := t.Where(keel.When(req.ActiveOnly, t.Status.Eq(domaintypes.StatusActive))).
			Order(t.Name.Asc()).
			Limit(limitOr(req.Limit, defaultTemplatePageSize)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list shift templates", zap.Error(err))
			return nil, fmt.Errorf("list shift templates: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) GetTemplatesByIDs(
	ctx context.Context,
	req *workerscheduleports.GetShiftTemplatesByIDsRequest,
) ([]*workerschedule.ShiftTemplate, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*workerschedule.ShiftTemplate, error) {
		if len(req.TemplateIDs) == 0 {
			return []*workerschedule.ShiftTemplate{}, nil
		}

		st := workerscheduledb.ShiftTemplates
		entities, err := st.Where(st.ID.In(req.TemplateIDs...)).
			Limit(len(req.TemplateIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get shift templates", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) GetTemplateByID(
	ctx context.Context,
	req *workerscheduleports.GetShiftTemplateByIDRequest,
) (*workerschedule.ShiftTemplate, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*workerschedule.ShiftTemplate, error) {
		entity, err := workerscheduledb.ShiftTemplates.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, templateErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateTemplate(
	ctx context.Context,
	entity *workerschedule.ShiftTemplate,
) (*workerschedule.ShiftTemplate, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*workerschedule.ShiftTemplate, error) {
		if err := workerscheduledb.ShiftTemplates.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create shift template", zap.Error(err))
			return nil, templateErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateTemplate(
	ctx context.Context,
	entity *workerschedule.ShiftTemplate,
) (*workerschedule.ShiftTemplate, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*workerschedule.ShiftTemplate, error) {
		if err := workerscheduledb.ShiftTemplates.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update shift template", zap.Error(err))
			return nil, templateErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CountTemplateAssignments(
	ctx context.Context,
	_ pagination.TenantInfo,
	templateID pulid.ID,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		a := workerscheduledb.WorkerShiftAssignments
		total, err := a.Where(a.ShiftTemplateID.Eq(templateID), a.EffectiveTo.IsNull()).
			Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count template assignments", zap.Error(err))
			return 0, fmt.Errorf("count template assignments: %w", err)
		}

		return int(total), nil
	})
}

func (r *repository) CountTemplateAssignmentsByIDs(
	ctx context.Context,
	req *workerscheduleports.CountShiftTemplateAssignmentsRequest,
) (map[pulid.ID]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]int, error) {
		if len(req.TemplateIDs) == 0 {
			return map[pulid.ID]int{}, nil
		}

		a := workerscheduledb.WorkerShiftAssignments
		rows, err := keel.Into[templateCount](ctx, r.db.Keel(ctx), a.
			Group(a.ShiftTemplateID).
			Pick(a.ShiftTemplateID, keel.Count()).
			Where(a.ShiftTemplateID.In(req.TemplateIDs...), a.EffectiveTo.IsNull()))
		if err != nil {
			r.l.Error("failed to count template assignments by template", zap.Error(err))
			return nil, fmt.Errorf("count template assignments by template: %w", err)
		}

		counts := make(map[pulid.ID]int, len(rows))
		for _, row := range rows {
			counts[row.TemplateID] = int(row.Count)
		}

		return counts, nil
	})
}

func (r *repository) ListAssignments(
	ctx context.Context,
	req *workerscheduleports.ListShiftAssignmentsRequest,
) ([]*workerschedule.WorkerShiftAssignment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*workerschedule.WorkerShiftAssignment, error) {
			a := workerscheduledb.WorkerShiftAssignments
			var activeAt keel.Pred
			if req.ActiveAt > 0 {
				activeAt = keel.And(
					a.EffectiveFrom.Lte(req.ActiveAt),
					keel.Or(a.EffectiveTo.IsNull(), a.EffectiveTo.Gte(req.ActiveAt)),
				)
			}

			entities, err := a.Where(
				keel.When(req.WorkerID.IsNotNil(), a.WorkerID.Eq(req.WorkerID)),
				activeAt,
			).
				Load(listquery.When(req.IncludeTemplate, a.ShiftTemplate())...).
				Order(a.EffectiveFrom.Desc()).
				Limit(limitOr(req.Limit, defaultAssignmentPageSize)).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list shift assignments", zap.Error(err))
				return nil, fmt.Errorf("list shift assignments: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *repository) GetAssignmentByID(
	ctx context.Context,
	req *workerscheduleports.GetShiftAssignmentByIDRequest,
) (*workerschedule.WorkerShiftAssignment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*workerschedule.WorkerShiftAssignment, error) {
			entity, err := workerscheduledb.WorkerShiftAssignments.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, assignmentErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) CreateAssignment(
	ctx context.Context,
	entity *workerschedule.WorkerShiftAssignment,
) (*workerschedule.WorkerShiftAssignment, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workerschedule.WorkerShiftAssignment, error) {
			if err := workerscheduledb.WorkerShiftAssignments.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				r.l.Error("failed to create shift assignment", zap.Error(err))
				return nil, assignmentErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateAssignment(
	ctx context.Context,
	entity *workerschedule.WorkerShiftAssignment,
) (*workerschedule.WorkerShiftAssignment, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workerschedule.WorkerShiftAssignment, error) {
			if err := workerscheduledb.WorkerShiftAssignments.Save(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				r.l.Error("failed to update shift assignment", zap.Error(err))
				return nil, assignmentErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) ListPreferences(
	ctx context.Context,
	req *workerscheduleports.ListAvailabilityPreferencesRequest,
) ([]*workerschedule.WorkerAvailabilityPreference, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*workerschedule.WorkerAvailabilityPreference, error) {
			p := workerscheduledb.WorkerAvailabilityPreferences
			// An empty worker reads the whole roster's stated availability,
			// which is what the rota needs: seven rows a worker at most, so
			// one read beats one per line on the board.
			byWorker := keel.When(req.WorkerID.IsNotNil(), p.WorkerID.Eq(req.WorkerID))
			entities, err := p.Where(byWorker).
				Order(p.WorkerID.Asc(), p.DayOfWeek.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list availability preferences", zap.Error(err))
				return nil, fmt.Errorf("list availability preferences: %w", err)
			}

			return entities, nil
		},
	)
}

// UpsertPreference writes one weekday's statement, replacing whatever was
// there. A worker has exactly one statement per weekday, so a collision is a
// correction rather than a second opinion.
func (r *repository) UpsertPreference(
	ctx context.Context,
	entity *workerschedule.WorkerAvailabilityPreference,
) (*workerschedule.WorkerAvailabilityPreference, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workerschedule.WorkerAvailabilityPreference, error) {
			db := r.db.Keel(ctx)
			p := workerscheduledb.WorkerAvailabilityPreferences
			_, err := p.Upsert(ctx, db, []*workerschedule.WorkerAvailabilityPreference{entity},
				keel.OnConflict(p.OrganizationID, p.BusinessUnitID, p.WorkerID, p.DayOfWeek).
					Update(p.Preference, p.Note))
			if err != nil {
				r.l.Error("failed to upsert availability preference", zap.Error(err))
				return nil, preferenceErrors.Map(err, entity.ID.String())
			}

			stored, err := p.Where(
				p.WorkerID.Eq(entity.WorkerID),
				p.DayOfWeek.Eq(entity.DayOfWeek),
			).One(ctx, db)
			if err != nil {
				return nil, preferenceErrors.Map(err, entity.ID.String())
			}

			return stored, nil
		},
	)
}

func swapsQuery(
	req *workerscheduleports.ListShiftSwapsRequest,
) *keel.Query[*workerschedule.ShiftSwapRequest] {
	s := workerscheduledb.ShiftSwapRequests

	return s.Where(
		keel.When(req.WorkerID.IsNotNil(), keel.Or(
			s.RequestingWorkerID.Eq(req.WorkerID),
			s.CounterpartyWorkerID.Eq(req.WorkerID),
		)),
		keel.When(
			req.OpenOnly,
			s.Status.In(workerschedule.SwapProposed, workerschedule.SwapAccepted),
		),
		keel.When(req.Since > 0, s.ShiftDate.Gte(req.Since)),
	).
		Load(listquery.When(req.IncludeWorkers, s.RequestingWorker(), s.CounterpartyWorker())...).
		Order(s.ShiftDate.Asc()).
		Limit(limitOr(req.Limit, defaultSwapPageSize))
}

func (r *repository) ListSwaps(
	ctx context.Context,
	req *workerscheduleports.ListShiftSwapsRequest,
) ([]*workerschedule.ShiftSwapRequest, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*workerschedule.ShiftSwapRequest, error) {
			entities, err := swapsQuery(req).List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list shift swaps", zap.Error(err))
				return nil, fmt.Errorf("list shift swaps: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *repository) GetSwapByID(
	ctx context.Context,
	req *workerscheduleports.GetShiftSwapByIDRequest,
) (*workerschedule.ShiftSwapRequest, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*workerschedule.ShiftSwapRequest, error) {
			entity, err := workerscheduledb.ShiftSwapRequests.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, swapErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) CreateSwap(
	ctx context.Context,
	entity *workerschedule.ShiftSwapRequest,
) (*workerschedule.ShiftSwapRequest, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workerschedule.ShiftSwapRequest, error) {
			if err := workerscheduledb.ShiftSwapRequests.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				r.l.Error("failed to create shift swap", zap.Error(err))
				return nil, swapErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateSwap(
	ctx context.Context,
	entity *workerschedule.ShiftSwapRequest,
) (*workerschedule.ShiftSwapRequest, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workerschedule.ShiftSwapRequest, error) {
			if err := workerscheduledb.ShiftSwapRequests.Save(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				r.l.Error("failed to update shift swap", zap.Error(err))
				return nil, swapErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}
