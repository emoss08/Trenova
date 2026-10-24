package fixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const employmentSeedDay = int64(86400)

type WorkerEmploymentEventSeed struct {
}

// WorkerEmploymentEventSeed gives every seeded driver a hire on the timeline and
// sprinkles the informational events (probation, promotion, rate change) so the
// Timeline tab has a story to tell without changing any worker's state.
//
// Depends on:
//   - Worker: the drivers whose hire dates open the timeline
var WorkerEmploymentEvent = Seeds.Add(seed.Seed{
	Name:        "worker_employment_event",
	Kind:        seed.Fixture,
	Description: "Seeds hire and informational employment events for seeded workers",
	After:       []*seed.Seed{Worker},
	Run:         (&WorkerEmploymentEventSeed{}).Run,
})

func (s *WorkerEmploymentEventSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}
	admin, err := userByUsername(ctx, db, "admin")
	if err != nil {
		return fmt.Errorf("get admin user: %w", err)
	}

	w := workerdb.Workers
	workers, err := w.AllTenants().
		Load(w.Profile()).
		Where(w.OrganizationID.Eq(org.ID), w.BusinessUnitID.Eq(org.BusinessUnitID)).
		Order(w.CreatedAt.Asc(), w.ID.Asc()).
		All(ctx, db)
	if err != nil {
		return fmt.Errorf("load workers: %w", err)
	}

	now := timeutils.NowUnix()
	for i, wrk := range workers {
		events := s.plan(wrk, i, now, admin.ID)
		for _, event := range events {
			event.OrganizationID = org.ID
			event.BusinessUnitID = org.BusinessUnitID
			event.WorkerID = wrk.ID
		}
		inserted, insertErr := workerdb.WorkerEmploymentEvents.CreateOrSkip(ctx, db, events)
		if insertErr != nil {
			return fmt.Errorf("insert employment events for %s: %w", wrk.ID, insertErr)
		}
		sr.Count(inserted)
	}
	return nil
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
			ID:          deterministicID("wee_", wrk.ID.String()+"Hired"),
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
			ID:           deterministicID("wee_", wrk.ID.String()+":Probation"),
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
			ID:           deterministicID("wee_", wrk.ID.String()+":Rate"),
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
			ID:           deterministicID("wee_", wrk.ID.String()+":Promoted"),
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
