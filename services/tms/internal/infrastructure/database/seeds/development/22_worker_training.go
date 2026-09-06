package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const trainingSeedDay = int64(86400)

type WorkerTrainingSeed struct {
	seedhelpers.BaseSeed
}

// WorkerTrainingSeed gives the development org a course catalog with a
// required matrix per driver type, and puts every seeded driver somewhere in
// it — completed, in progress, due soon, overdue and lapsed — so the Training
// tab, the portal and the reminder sweep all have something to show.
//
// Depends on:
//   - Worker: the drivers being trained
func NewWorkerTrainingSeed() *WorkerTrainingSeed {
	seed := &WorkerTrainingSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"WorkerTraining",
		"1.0.0",
		"Seeds the training course catalog and per-worker training records",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedWorker)
	return seed
}

type trainingSeedRefs struct {
	orgID   pulid.ID
	buID    pulid.ID
	adminID pulid.ID
	now     int64
	courses map[string]*worker.TrainingCourse
}

func (s *WorkerTrainingSeed) Run(ctx context.Context, tx bun.Tx) error {
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
			refs := &trainingSeedRefs{
				orgID:   org.ID,
				buID:    org.BusinessUnitID,
				adminID: admin.ID,
				now:     timeutils.NowUnix(),
			}

			if err = s.ensureCourses(ctx, tx, refs); err != nil {
				return fmt.Errorf("ensure training courses: %w", err)
			}

			workers, err := s.loadWorkers(ctx, tx, refs)
			if err != nil {
				return fmt.Errorf("load workers: %w", err)
			}
			for i, wrk := range workers {
				if err = s.seedWorker(ctx, tx, sc, refs, wrk, i); err != nil {
					return fmt.Errorf("seed training for worker %s: %w", wrk.ID, err)
				}
			}
			return nil
		},
	)
}

func (s *WorkerTrainingSeed) catalog(refs *trainingSeedRefs) []*worker.TrainingCourse {
	months := func(n int32) *int32 { return &n }
	base := func(code, name string) *worker.TrainingCourse {
		return &worker.TrainingCourse{
			OrganizationID:          refs.orgID,
			BusinessUnitID:          refs.buID,
			Code:                    code,
			Name:                    name,
			Status:                  domaintypes.StatusActive,
			RenewalWindowDays:       30,
			DueDaysAfterAssignment:  30,
			RequiresAcknowledgement: true,
			RequiredForDriverTypes:  []worker.DriverType{},
		}
	}

	orientation := base("ORIENTATION", "New Driver Orientation")
	orientation.Description = "Company policies, dispatch procedures, and the safety programme."
	orientation.Category = worker.TrainingCategoryOrientation
	orientation.Delivery = worker.TrainingDeliveryClassroom
	orientation.DurationMinutes = 240
	orientation.IsRequired = true
	orientation.DueDaysAfterAssignment = 14
	orientation.SortOrder = 10

	defensive := base("DEFENSIVE", "Defensive Driving")
	defensive.Description = "Smith System defensive driving refresher."
	defensive.Category = worker.TrainingCategorySafety
	defensive.Delivery = worker.TrainingDeliveryOnline
	defensive.ContentURL = "https://training.example.com/courses/defensive-driving"
	defensive.DurationMinutes = 90
	defensive.ValidityMonths = months(24)
	defensive.IsRequired = true
	defensive.SortOrder = 20

	hos := base("HOS-REFRESH", "Hours of Service Refresher")
	hos.Description = "Annual review of the FMCSA hours-of-service rules and ELD use."
	hos.Category = worker.TrainingCategoryCompliance
	hos.Delivery = worker.TrainingDeliveryDocument
	hos.ContentURL = "https://training.example.com/docs/hos-refresher.pdf"
	hos.DurationMinutes = 30
	hos.ValidityMonths = months(12)
	hos.IsRequired = true
	hos.SortOrder = 30

	hazmat := base("HAZMAT-AWARE", "Hazmat Awareness")
	hazmat.Description = "49 CFR 172 Subpart H general awareness and function-specific training."
	hazmat.Category = worker.TrainingCategoryHazardousMaterials
	hazmat.Delivery = worker.TrainingDeliveryClassroom
	hazmat.DurationMinutes = 180
	hazmat.PassingScore = decimal.NewNullDecimal(decimal.NewFromInt(80))
	hazmat.ValidityMonths = months(36)
	hazmat.IsRequired = true
	hazmat.RequiredForDriverTypes = []worker.DriverType{worker.DriverTypeOTR, worker.DriverTypeRegional}
	hazmat.SortOrder = 40

	forklift := base("FORKLIFT-OPS", "Forklift Operation")
	forklift.Description = "OSHA 1910.178 powered industrial truck operator training."
	forklift.Category = worker.TrainingCategoryEquipment
	forklift.Delivery = worker.TrainingDeliveryOnTheJob
	forklift.DurationMinutes = 120
	forklift.PassingScore = decimal.NewNullDecimal(decimal.NewFromInt(70))
	forklift.ValidityMonths = months(36)
	forklift.SortOrder = 50

	winter := base("WINTER", "Winter Driving")
	winter.Description = "Chain-up, black ice and mountain pass procedures."
	winter.Category = worker.TrainingCategorySafety
	winter.Delivery = worker.TrainingDeliveryOnline
	winter.ContentURL = "https://training.example.com/courses/winter-driving"
	winter.DurationMinutes = 45
	winter.SortOrder = 60

	return []*worker.TrainingCourse{orientation, defensive, hos, hazmat, forklift, winter}
}

func (s *WorkerTrainingSeed) ensureCourses(
	ctx context.Context,
	tx bun.Tx,
	refs *trainingSeedRefs,
) error {
	rows := s.catalog(refs)
	if _, err := tx.NewInsert().Model(&rows).On("CONFLICT DO NOTHING").Exec(ctx); err != nil {
		return err
	}

	existing := make([]*worker.TrainingCourse, 0, len(rows))
	cols := buncolgen.TrainingCourseColumns
	if err := tx.NewSelect().
		Model(&existing).
		Where(cols.OrganizationID.Eq(), refs.orgID).
		Where(cols.BusinessUnitID.Eq(), refs.buID).
		Scan(ctx); err != nil {
		return err
	}
	refs.courses = make(map[string]*worker.TrainingCourse, len(existing))
	for _, course := range existing {
		refs.courses[course.Code] = course
	}
	return nil
}

func (s *WorkerTrainingSeed) loadWorkers(
	ctx context.Context,
	tx bun.Tx,
	refs *trainingSeedRefs,
) ([]*worker.Worker, error) {
	workers := make([]*worker.Worker, 0, 32)
	cols := buncolgen.WorkerColumns
	err := tx.NewSelect().
		Model(&workers).
		Where(cols.OrganizationID.Eq(), refs.orgID).
		Where(cols.BusinessUnitID.Eq(), refs.buID).
		Where(cols.Status.Eq(), domaintypes.StatusActive).
		Order(cols.CreatedAt.OrderAsc()).
		Scan(ctx)
	return workers, err
}

type seedTraining struct {
	code         string
	status       worker.TrainingStatus
	completedAgo int64
	dueIn        int64
	score        *int64
	waivedReason string
}

func (s *WorkerTrainingSeed) plan(wrk *worker.Worker, index int) []seedTraining {
	score := func(v int64) *int64 { return &v }
	plan := []seedTraining{
		{code: "ORIENTATION", status: worker.TrainingStatusCompleted, completedAgo: 400 + int64(index)*20},
	}

	switch index % 5 {
	case 0:
		plan = append(plan,
			seedTraining{code: "DEFENSIVE", status: worker.TrainingStatusCompleted, completedAgo: 200},
			seedTraining{code: "HOS-REFRESH", status: worker.TrainingStatusCompleted, completedAgo: 100},
		)
	case 1:
		plan = append(plan,
			seedTraining{code: "DEFENSIVE", status: worker.TrainingStatusCompleted, completedAgo: 700},
			seedTraining{code: "HOS-REFRESH", status: worker.TrainingStatusAssigned, dueIn: 5},
		)
	case 2:
		plan = append(plan,
			seedTraining{code: "DEFENSIVE", status: worker.TrainingStatusInProgress, dueIn: 12},
			seedTraining{code: "HOS-REFRESH", status: worker.TrainingStatusAssigned, dueIn: -6},
		)
	case 3:
		plan = append(plan,
			seedTraining{code: "DEFENSIVE", status: worker.TrainingStatusCompleted, completedAgo: 30},
			seedTraining{code: "HOS-REFRESH", status: worker.TrainingStatusCompleted, completedAgo: 380},
		)
	case 4:
		plan = append(plan,
			seedTraining{code: "DEFENSIVE", status: worker.TrainingStatusWaived, waivedReason: "Completed at previous carrier; certificate on file"},
			seedTraining{code: "HOS-REFRESH", status: worker.TrainingStatusCompleted, completedAgo: 20},
		)
	}

	if wrk.DriverType == worker.DriverTypeOTR || wrk.DriverType == worker.DriverTypeRegional {
		if index%3 == 0 {
			plan = append(plan, seedTraining{code: "HAZMAT-AWARE", status: worker.TrainingStatusCompleted, completedAgo: 500, score: score(92)})
		} else if index%3 == 1 {
			plan = append(plan, seedTraining{code: "HAZMAT-AWARE", status: worker.TrainingStatusAssigned, dueIn: 20})
		}
	}
	if index%4 == 1 {
		plan = append(plan, seedTraining{code: "FORKLIFT-OPS", status: worker.TrainingStatusCompleted, completedAgo: 250, score: score(84)})
	}
	if index%4 == 2 {
		plan = append(plan, seedTraining{code: "FORKLIFT-OPS", status: worker.TrainingStatusFailed, completedAgo: 15, score: score(55)})
	}
	if index%2 == 0 {
		plan = append(plan, seedTraining{code: "WINTER", status: worker.TrainingStatusCompleted, completedAgo: 60})
	}
	return plan
}

func (s *WorkerTrainingSeed) seedWorker(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *trainingSeedRefs,
	wrk *worker.Worker,
	index int,
) error {
	for _, item := range s.plan(wrk, index) {
		course, ok := refs.courses[item.code]
		if !ok {
			continue
		}
		if exists, err := s.hasRecord(ctx, tx, refs, wrk, course); err != nil || exists {
			if err != nil {
				return err
			}
			continue
		}

		entity := &worker.WorkerTrainingRecord{
			OrganizationID: refs.orgID,
			BusinessUnitID: refs.buID,
			WorkerID:       wrk.ID,
			CourseID:       course.ID,
			Status:         item.status,
			AssignedByID:   refs.adminID,
		}
		switch item.status {
		case worker.TrainingStatusCompleted, worker.TrainingStatusFailed:
			completed := refs.now - item.completedAgo*trainingSeedDay
			entity.AssignedAt = completed - 10*trainingSeedDay
			entity.StartedAt = &completed
			entity.CompletedAt = &completed
			entity.AcknowledgedAt = &completed
			entity.RecordedByID = refs.adminID
			passed := item.status == worker.TrainingStatusCompleted
			entity.Passed = &passed
			if item.score != nil {
				entity.Score = decimal.NewNullDecimal(decimal.NewFromInt(*item.score))
			}
			if passed {
				entity.ExpiresAt = course.ExpiryFor(completed)
				if entity.ExpiresAt != nil && *entity.ExpiresAt < refs.now {
					entity.Status = worker.TrainingStatusExpired
				}
			}
		case worker.TrainingStatusAssigned, worker.TrainingStatusInProgress:
			entity.AssignedAt = refs.now - 10*trainingSeedDay
			due := refs.now + item.dueIn*trainingSeedDay
			entity.DueAt = &due
			if item.status == worker.TrainingStatusInProgress {
				started := refs.now - 3*trainingSeedDay
				entity.StartedAt = &started
			}
		case worker.TrainingStatusWaived:
			entity.AssignedAt = refs.now - 40*trainingSeedDay
			entity.WaivedReason = item.waivedReason
			entity.RecordedByID = refs.adminID
		case worker.TrainingStatusExpired, worker.TrainingStatusCancelled:
			continue
		}

		if _, err := tx.NewInsert().Model(entity).Exec(ctx); err != nil {
			return err
		}
		if err := sc.TrackCreated(ctx, "worker_training_records", entity.ID, s.Name()); err != nil {
			return err
		}
	}
	return nil
}

func (s *WorkerTrainingSeed) hasRecord(
	ctx context.Context,
	tx bun.Tx,
	refs *trainingSeedRefs,
	wrk *worker.Worker,
	course *worker.TrainingCourse,
) (bool, error) {
	cols := buncolgen.WorkerTrainingRecordColumns
	return tx.NewSelect().
		Model((*worker.WorkerTrainingRecord)(nil)).
		Where(cols.OrganizationID.Eq(), refs.orgID).
		Where(cols.BusinessUnitID.Eq(), refs.buID).
		Where(cols.WorkerID.Eq(), wrk.ID).
		Where(cols.CourseID.Eq(), course.ID).
		Exists(ctx)
}

func (s *WorkerTrainingSeed) Down(ctx context.Context, tx bun.Tx) error {
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

func (s *WorkerTrainingSeed) CanRollback() bool {
	return true
}
