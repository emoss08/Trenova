package fixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const trainingSeedDay = int64(86400)

type WorkerTrainingSeed struct {
}

// WorkerTrainingSeed gives the development org a course catalog with a
// required matrix per driver type, and puts every seeded driver somewhere in
// it — completed, in progress, due soon, overdue and lapsed — so the Training
// tab, the portal and the reminder sweep all have something to show.
//
// Depends on:
//   - Worker: the drivers being trained
var WorkerTraining = Seeds.Add(seed.Seed{
	Name:        "worker_training",
	Kind:        seed.Fixture,
	Description: "Seeds the training course catalog and per-worker training records",
	After:       []*seed.Seed{Worker},
	Run:         (&WorkerTrainingSeed{}).Run,
})

type trainingSeedRefs struct {
	orgID   pulid.ID
	buID    pulid.ID
	adminID pulid.ID
	now     int64
	courses map[string]*workertraining.TrainingCourse
}

func (s *WorkerTrainingSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}
	admin, err := userByUsername(ctx, db, "admin")
	if err != nil {
		return fmt.Errorf("get admin user: %w", err)
	}
	refs := &trainingSeedRefs{
		orgID:   org.ID,
		buID:    org.BusinessUnitID,
		adminID: admin.ID,
		now:     timeutils.NowUnix(),
	}

	if err = s.ensureCourses(ctx, db, refs); err != nil {
		return fmt.Errorf("ensure training courses: %w", err)
	}

	workers, err := s.loadWorkers(ctx, db, refs)
	if err != nil {
		return fmt.Errorf("load workers: %w", err)
	}
	for i, wrk := range workers {
		if err = s.seedWorker(ctx, db, sr, refs, wrk, i); err != nil {
			return fmt.Errorf("seed training for worker %s: %w", wrk.ID, err)
		}
	}
	return nil
}

func (s *WorkerTrainingSeed) catalog(refs *trainingSeedRefs) []*workertraining.TrainingCourse {
	months := func(n int32) *int32 { return &n }
	base := func(code, name string) *workertraining.TrainingCourse {
		return &workertraining.TrainingCourse{
			OrganizationID:          refs.orgID,
			BusinessUnitID:          refs.buID,
			Code:                    code,
			Name:                    name,
			Status:                  domaintypes.StatusActive,
			RenewalWindowDays:       30,
			DueDaysAfterAssignment:  30,
			RequiresAcknowledgement: true,
			RequiredForDriverTypes:  []worker.DriverType{},
			Category:                workertraining.TrainingCategoryOther,
			Delivery:                workertraining.TrainingDeliveryOnline,
		}
	}

	orientation := base("ORIENTATION", "New Driver Orientation")
	orientation.Description = "Company policies, dispatch procedures, and the safety programme."
	orientation.Category = workertraining.TrainingCategoryOrientation
	orientation.Delivery = workertraining.TrainingDeliveryClassroom
	orientation.DurationMinutes = 240
	orientation.IsRequired = true
	orientation.DueDaysAfterAssignment = 14
	orientation.SortOrder = 10

	defensive := base("DEFENSIVE", "Defensive Driving")
	defensive.Description = "Smith System defensive driving refresher."
	defensive.Category = workertraining.TrainingCategorySafety
	defensive.Delivery = workertraining.TrainingDeliveryOnline
	defensive.ContentURL = "https://training.example.com/courses/defensive-driving"
	defensive.DurationMinutes = 90
	defensive.ValidityMonths = months(24)
	defensive.IsRequired = true
	defensive.SortOrder = 20

	hos := base("HOS-REFRESH", "Hours of Service Refresher")
	hos.Description = "Annual review of the FMCSA hours-of-service rules and ELD use."
	hos.Category = workertraining.TrainingCategoryCompliance
	hos.Delivery = workertraining.TrainingDeliveryDocument
	hos.ContentURL = "https://training.example.com/docs/hos-refresher.pdf"
	hos.DurationMinutes = 30
	hos.ValidityMonths = months(12)
	hos.IsRequired = true
	hos.SortOrder = 30

	hazmat := base("HAZMAT-AWARE", "Hazmat Awareness")
	hazmat.Description = "49 CFR 172 Subpart H general awareness and function-specific training."
	hazmat.Category = workertraining.TrainingCategoryHazardousMaterials
	hazmat.Delivery = workertraining.TrainingDeliveryClassroom
	hazmat.DurationMinutes = 180
	hazmat.PassingScore = decimal.NewNullDecimal(decimal.NewFromInt(80))
	hazmat.ValidityMonths = months(36)
	hazmat.IsRequired = true
	hazmat.RequiredForDriverTypes = []worker.DriverType{
		worker.DriverTypeOTR,
		worker.DriverTypeRegional,
	}
	hazmat.SortOrder = 40

	forklift := base("FORKLIFT-OPS", "Forklift Operation")
	forklift.Description = "OSHA 1910.178 powered industrial truck operator training."
	forklift.Category = workertraining.TrainingCategoryEquipment
	forklift.Delivery = workertraining.TrainingDeliveryOnTheJob
	forklift.DurationMinutes = 120
	forklift.PassingScore = decimal.NewNullDecimal(decimal.NewFromInt(70))
	forklift.ValidityMonths = months(36)
	forklift.SortOrder = 50

	winter := base("WINTER", "Winter Driving")
	winter.Description = "Chain-up, black ice and mountain pass procedures."
	winter.Category = workertraining.TrainingCategorySafety
	winter.Delivery = workertraining.TrainingDeliveryOnline
	winter.ContentURL = "https://training.example.com/courses/winter-driving"
	winter.DurationMinutes = 45
	winter.SortOrder = 60

	return []*workertraining.TrainingCourse{orientation, defensive, hos, hazmat, forklift, winter}
}

func (s *WorkerTrainingSeed) ensureCourses(
	ctx context.Context,
	db *keel.DB,
	refs *trainingSeedRefs,
) error {
	rows := s.catalog(refs)
	if _, err := workerdb.TrainingCourses.CreateOrSkip(ctx, db, rows); err != nil {
		return err
	}

	existing, err := workerdb.TrainingCourses.AllTenants().Where(
		workerdb.TrainingCourses.OrganizationID.Eq(refs.orgID),
		workerdb.TrainingCourses.BusinessUnitID.Eq(refs.buID),
	).All(ctx, db)
	if err != nil {
		return err
	}
	refs.courses = make(map[string]*workertraining.TrainingCourse, len(existing))
	for _, course := range existing {
		refs.courses[course.Code] = course
	}
	return nil
}

func (s *WorkerTrainingSeed) loadWorkers(
	ctx context.Context,
	db *keel.DB,
	refs *trainingSeedRefs,
) ([]*worker.Worker, error) {
	workers, err := workerdb.Workers.AllTenants().Where(
		workerdb.Workers.OrganizationID.Eq(refs.orgID),
		workerdb.Workers.BusinessUnitID.Eq(refs.buID),
		workerdb.Workers.Status.Eq(domaintypes.StatusActive),
	).Order(workerdb.Workers.CreatedAt.Asc()).All(ctx, db)
	return workers, err
}

type seedTraining struct {
	code         string
	status       workertraining.TrainingStatus
	completedAgo int64
	dueIn        int64
	score        *int64
	waivedReason string
}

func (s *WorkerTrainingSeed) plan(wrk *worker.Worker, index int) []seedTraining {
	score := func(v int64) *int64 { return &v }
	plan := []seedTraining{
		{
			code:         "ORIENTATION",
			status:       workertraining.TrainingStatusCompleted,
			completedAgo: 400 + int64(index)*20,
		},
	}

	switch index % 5 {
	case 0:
		plan = append(
			plan,
			seedTraining{
				code:         "DEFENSIVE",
				status:       workertraining.TrainingStatusCompleted,
				completedAgo: 200,
			},
			seedTraining{
				code:         "HOS-REFRESH",
				status:       workertraining.TrainingStatusCompleted,
				completedAgo: 100,
			},
		)
	case 1:
		plan = append(
			plan,
			seedTraining{
				code:         "DEFENSIVE",
				status:       workertraining.TrainingStatusCompleted,
				completedAgo: 700,
			},
			seedTraining{
				code:   "HOS-REFRESH",
				status: workertraining.TrainingStatusAssigned,
				dueIn:  5,
			},
		)
	case 2:
		plan = append(
			plan,
			seedTraining{
				code:   "DEFENSIVE",
				status: workertraining.TrainingStatusInProgress,
				dueIn:  12,
			},
			seedTraining{
				code:   "HOS-REFRESH",
				status: workertraining.TrainingStatusAssigned,
				dueIn:  -6,
			},
		)
	case 3:
		plan = append(
			plan,
			seedTraining{
				code:         "DEFENSIVE",
				status:       workertraining.TrainingStatusCompleted,
				completedAgo: 30,
			},
			seedTraining{
				code:         "HOS-REFRESH",
				status:       workertraining.TrainingStatusCompleted,
				completedAgo: 380,
			},
		)
	case 4:
		plan = append(
			plan,
			seedTraining{
				code:         "DEFENSIVE",
				status:       workertraining.TrainingStatusWaived,
				waivedReason: "Completed at previous carrier; certificate on file",
			},
			seedTraining{
				code:         "HOS-REFRESH",
				status:       workertraining.TrainingStatusCompleted,
				completedAgo: 20,
			},
		)
	}

	if wrk.DriverType == worker.DriverTypeOTR || wrk.DriverType == worker.DriverTypeRegional {
		if index%3 == 0 {
			plan = append(
				plan,
				seedTraining{
					code:         "HAZMAT-AWARE",
					status:       workertraining.TrainingStatusCompleted,
					completedAgo: 500,
					score:        score(92),
				},
			)
		} else if index%3 == 1 {
			plan = append(plan, seedTraining{code: "HAZMAT-AWARE", status: workertraining.TrainingStatusAssigned, dueIn: 20})
		}
	}
	if index%4 == 1 {
		plan = append(
			plan,
			seedTraining{
				code:         "FORKLIFT-OPS",
				status:       workertraining.TrainingStatusCompleted,
				completedAgo: 250,
				score:        score(84),
			},
		)
	}
	if index%4 == 2 {
		plan = append(
			plan,
			seedTraining{
				code:         "FORKLIFT-OPS",
				status:       workertraining.TrainingStatusFailed,
				completedAgo: 15,
				score:        score(55),
			},
		)
	}
	if index%2 == 0 {
		plan = append(
			plan,
			seedTraining{
				code:         "WINTER",
				status:       workertraining.TrainingStatusCompleted,
				completedAgo: 60,
			},
		)
	}
	return plan
}

func (s *WorkerTrainingSeed) seedWorker(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *trainingSeedRefs,
	wrk *worker.Worker,
	index int,
) error {
	for _, item := range s.plan(wrk, index) {
		course, ok := refs.courses[item.code]
		if !ok {
			continue
		}
		if exists, err := s.hasRecord(ctx, db, refs, wrk, course); err != nil || exists {
			if err != nil {
				return err
			}
			continue
		}

		entity := &workertraining.WorkerTrainingRecord{
			OrganizationID: refs.orgID,
			BusinessUnitID: refs.buID,
			WorkerID:       wrk.ID,
			CourseID:       course.ID,
			Status:         item.status,
			AssignedByID:   refs.adminID,
		}
		switch item.status {
		case workertraining.TrainingStatusCompleted, workertraining.TrainingStatusFailed:
			completed := refs.now - item.completedAgo*trainingSeedDay
			entity.AssignedAt = completed - 10*trainingSeedDay
			entity.StartedAt = &completed
			entity.CompletedAt = &completed
			entity.AcknowledgedAt = &completed
			entity.RecordedByID = refs.adminID
			passed := item.status == workertraining.TrainingStatusCompleted
			entity.Passed = &passed
			if item.score != nil {
				entity.Score = decimal.NewNullDecimal(decimal.NewFromInt(*item.score))
			}
			if passed {
				entity.ExpiresAt = course.ExpiryFor(completed)
				if entity.ExpiresAt != nil && *entity.ExpiresAt < refs.now {
					entity.Status = workertraining.TrainingStatusExpired
				}
			}
		case workertraining.TrainingStatusAssigned, workertraining.TrainingStatusInProgress:
			entity.AssignedAt = refs.now - 10*trainingSeedDay
			due := refs.now + item.dueIn*trainingSeedDay
			entity.DueAt = &due
			if item.status == workertraining.TrainingStatusInProgress {
				started := refs.now - 3*trainingSeedDay
				entity.StartedAt = &started
			}
		case workertraining.TrainingStatusWaived:
			entity.AssignedAt = refs.now - 40*trainingSeedDay
			entity.WaivedReason = item.waivedReason
			entity.RecordedByID = refs.adminID
		case workertraining.TrainingStatusExpired, workertraining.TrainingStatusCancelled:
			continue
		}

		if err := workerdb.WorkerTrainingRecords.Create(ctx, db, entity); err != nil {
			return err
		}
	}
	return nil
}

func (s *WorkerTrainingSeed) hasRecord(
	ctx context.Context,
	db *keel.DB,
	refs *trainingSeedRefs,
	wrk *worker.Worker,
	course *workertraining.TrainingCourse,
) (bool, error) {
	return workerdb.WorkerTrainingRecords.AllTenants().Where(
		workerdb.WorkerTrainingRecords.OrganizationID.Eq(refs.orgID),
		workerdb.WorkerTrainingRecords.BusinessUnitID.Eq(refs.buID),
		workerdb.WorkerTrainingRecords.WorkerID.Eq(wrk.ID),
		workerdb.WorkerTrainingRecords.CourseID.Eq(course.ID),
	).Exists(ctx, db)
}
