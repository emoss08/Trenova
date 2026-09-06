package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const employmentSeedDay = int64(86400)

type WorkerEmploymentEventSeed struct {
	seedhelpers.BaseSeed
}

// WorkerEmploymentEventSeed gives every seeded driver a hire on the timeline and
// sprinkles the informational events (probation, promotion, rate change) so the
// Timeline tab has a story to tell without changing any worker's state.
//
// Depends on:
//   - Worker: the drivers whose hire dates open the timeline
func NewWorkerEmploymentEventSeed() *WorkerEmploymentEventSeed {
	seed := &WorkerEmploymentEventSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"WorkerEmploymentEvent",
		"1.0.0",
		"Seeds hire and informational employment events for seeded workers",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedWorker)
	return seed
}

func (s *WorkerEmploymentEventSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				return err
			}
			admin, err := sc.GetUserByUsername(ctx, "admin")
			if err != nil {
				return fmt.Errorf("get admin user: %w", err)
			}

			workers := make([]*worker.Worker, 0, 32)
			cols := buncolgen.WorkerColumns
			if err = tx.NewSelect().
				Model(&workers).
				Relation(buncolgen.WorkerRelations.Profile).
				Where(cols.OrganizationID.Eq(), org.ID).
				Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
				Order(cols.CreatedAt.OrderAsc()).
				Scan(ctx); err != nil {
				return fmt.Errorf("load workers: %w", err)
			}

			now := timeutils.NowUnix()
			for i, wrk := range workers {
				for _, event := range s.plan(wrk, i, now, admin.ID) {
					event.OrganizationID = org.ID
					event.BusinessUnitID = org.BusinessUnitID
					event.WorkerID = wrk.ID
					result, insertErr := tx.NewInsert().
						Model(event).
						On("CONFLICT DO NOTHING").
						Exec(ctx)
					if insertErr != nil {
						return fmt.Errorf("insert employment event for %s: %w", wrk.ID, insertErr)
					}
					if inserted, _ := result.RowsAffected(); inserted == 0 {
						continue
					}
					if err = sc.TrackCreated(ctx, "worker_employment_events", event.ID, s.Name()); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

func (s *WorkerEmploymentEventSeed) plan(
	wrk *worker.Worker,
	index int,
	now int64,
	adminID pulid.ID,
) []*worker.WorkerEmploymentEvent {
	if wrk.Profile == nil || wrk.Profile.HireDate <= 0 {
		return nil
	}
	hire := wrk.Profile.HireDate
	events := []*worker.WorkerEmploymentEvent{
		{
			ID:          seedhelpers.DeterministicID("wee_", wrk.ID.String()+"Hired"),
			Kind:        worker.EmploymentEventHired,
			EffectiveAt: hire,
			ToValues: map[string]string{
				worker.EmploymentValueHireDate: fmt.Sprintf("%d", hire),
				worker.EmploymentValueStatus:   "Active",
			},
			FromValues: map[string]string{},
		},
	}
	if hire+90*employmentSeedDay < now {
		events = append(events, &worker.WorkerEmploymentEvent{
			ID:           seedhelpers.DeterministicID("wee_", wrk.ID.String()+":Probation"),
			Kind:         worker.EmploymentEventProbationEnded,
			EffectiveAt:  hire + 90*employmentSeedDay,
			Notes:        "Completed the 90-day probation period with a clean safety record.",
			RecordedByID: adminID,
			FromValues:   map[string]string{},
			ToValues:     map[string]string{},
		})
	}
	if index%3 == 1 && hire+400*employmentSeedDay < now {
		events = append(events, &worker.WorkerEmploymentEvent{
			ID:           seedhelpers.DeterministicID("wee_", wrk.ID.String()+":Rate"),
			Kind:         worker.EmploymentEventRateChanged,
			EffectiveAt:  hire + 365*employmentSeedDay,
			Reason:       "Annual review",
			RecordedByID: adminID,
			FromValues:   map[string]string{worker.EmploymentValueRate: "0.58"},
			ToValues: map[string]string{
				worker.EmploymentValueRate:     "0.62",
				worker.EmploymentValueRateUnit: "per mile",
			},
		})
	}
	if index%4 == 2 && hire+500*employmentSeedDay < now {
		events = append(events, &worker.WorkerEmploymentEvent{
			ID:           seedhelpers.DeterministicID("wee_", wrk.ID.String()+":Promoted"),
			Kind:         worker.EmploymentEventPromoted,
			EffectiveAt:  hire + 500*employmentSeedDay,
			Reason:       "Moved to the regional board",
			RecordedByID: adminID,
			FromValues:   map[string]string{worker.EmploymentValueDriverType: "Local"},
			ToValues:     map[string]string{worker.EmploymentValueDriverType: wrk.DriverType.String()},
		})
	}
	return events
}

func (s *WorkerEmploymentEventSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *WorkerEmploymentEventSeed) CanRollback() bool {
	return true
}
