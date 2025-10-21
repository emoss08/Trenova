package development

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/jaswdr/faker/v2"
	"github.com/uptrace/bun"
)

// WorkersSeed Creates workers data
type WorkersSeed struct {
	seedhelpers.BaseSeed
}

// NewWorkersSeed creates a new workers seed
func NewWorkersSeed() *WorkersSeed {
	seed := &WorkersSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"Workers",
		"1.0.0",
		"Creates workers data",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	seed.SetDependencies("USStates", "AdminAccount", "Permissions", "HazmatExpiration")

	return seed
}

// Run executes the seed
func (s *WorkersSeed) Run(ctx context.Context, db *bun.DB) error {
	return seedhelpers.RunInTransaction(
		ctx,
		db,
		s.Name(),
		func(ctx context.Context, tx bun.Tx, seedCtx *seedhelpers.SeedContext) error {
			var count int
			err := db.NewSelect().
				Model((*worker.Worker)(nil)).
				ColumnExpr("count(*)").
				Scan(ctx, &count)
			if err != nil {
				return err
			}

			if count > 0 {
				seedhelpers.LogSuccess("Workers already exist, skipping")
				return nil
			}

			defaultOrg, err := seedCtx.GetDefaultOrganization()
			if err != nil {
				return fmt.Errorf("get default organization: %w", err)
			}

			defaultBU, err := seedCtx.GetDefaultBusinessUnit()
			if err != nil {
				return fmt.Errorf("get default business unit: %w", err)
			}

			caState, err := seedCtx.GetState("CA")
			if err != nil {
				return fmt.Errorf("get California state: %w", err)
			}

			user, err := seedCtx.CreateUser(tx, &seedhelpers.UserOptions{
				Name:           "Test User",
				Username:       "testuser",
				Email:          "testuser@example.com",
				OrganizationID: defaultOrg.ID,
				BusinessUnitID: defaultBU.ID,
			})
			if err != nil {
				return err
			}

			const numWorkers = 1000
			workers := make([]*worker.Worker, 0, numWorkers)
			profiles := make([]*worker.WorkerProfile, 0, numWorkers)
			ptos := make([]*worker.WorkerPTO, 0, numWorkers*3)

			fake := faker.New()
			now := time.Now()

			for range numWorkers {
				workerID := pulid.MustNew("wrk_")

				wrk := &worker.Worker{
					ID:             workerID,
					BusinessUnitID: defaultBU.ID,
					OrganizationID: defaultOrg.ID,
					StateID:        caState.ID,
					Status:         domain.StatusActive,
					Type:           worker.WorkerTypeEmployee,
					FirstName:      fake.Person().FirstName(),
					LastName:       fake.Person().LastName(),
					AddressLine1:   fake.Address().StreetAddress(),
					AddressLine2:   fake.Address().SecondaryAddress(),
					City:           fake.Address().City(),
					PostalCode:     fake.Address().PostCode(),
					Gender:         getRandomGender(),
					CanBeAssigned:  true,
				}

				endorsementType := getRandomEndorsement()
				hireDate := generateRandomDate(now.AddDate(-5, 0, 0), now.AddDate(0, -1, 0))
				licenseExpiry := generateRandomDate(now, now.AddDate(4, 0, 0))

				profile := &worker.WorkerProfile{
					ID:             pulid.MustNew("wp_"),
					WorkerID:       workerID,
					BusinessUnitID: defaultBU.ID,
					OrganizationID: defaultOrg.ID,
					LicenseStateID: caState.ID,
					DOB: generateRandomDate(
						now.AddDate(-65, 0, 0),
						now.AddDate(-21, 0, 0),
					),
					LicenseNumber: fake.RandomStringWithLength(8),
					Endorsement:   endorsementType,
					LicenseExpiry: licenseExpiry,
					HireDate:      hireDate,
					PhysicalDueDate: func() *int64 {
						date := generateRandomDate(now, now.AddDate(1, 0, 0))
						return &date
					}(),
					MVRDueDate: func() *int64 {
						date := generateRandomDate(now, now.AddDate(1, 0, 0))
						return &date
					}(),
					ComplianceStatus:    worker.ComplianceStatusCompliant,
					IsQualified:         true,
					LastComplianceCheck: generateRandomDate(now.AddDate(0, -6, 0), now),
					LastMVRCheck:        generateRandomDate(now.AddDate(0, -6, 0), now),
					LastDrugTest:        generateRandomDate(now.AddDate(0, -6, 0), now),
				}

				if endorsementType == worker.EndorsementHazmat ||
					endorsementType == worker.EndorsementTankerHazmat {
					hazmatExpiry := generateRandomDate(now, now.AddDate(2, 0, 0))
					profile.HazmatExpiry = hazmatExpiry
				}

				workers = append(workers, wrk)
				profiles = append(profiles, profile)

				workerPTOs := generatePTOsForWorker(
					workerID,
					defaultOrg.ID,
					defaultBU.ID,
					user.ID,
					now,
				)
				ptos = append(ptos, workerPTOs...)
			}

			if _, err := tx.NewInsert().Model(&workers).Exec(ctx); err != nil {
				return fmt.Errorf("failed to bulk insert workers: %w", err)
			}

			if _, err := tx.NewInsert().Model(&profiles).Exec(ctx); err != nil {
				return fmt.Errorf("failed to bulk insert worker profiles: %w", err)
			}

			if len(ptos) > 0 {
				if _, err := tx.NewInsert().Model(&ptos).Exec(ctx); err != nil {
					return fmt.Errorf("failed to bulk insert worker PTOs: %w", err)
				}
			}

			seedhelpers.LogSuccess("Created workers fixtures",
				"- 1000 workers created",
			)

			return nil
		},
	)
}

func generateRandomDate(start, end time.Time) int64 {
	minimum := start.Unix()
	maximum := end.Unix()
	return rand.Int63n(maximum-minimum) + minimum
}

func getRandomEndorsement() worker.EndorsementType {
	endorsements := []worker.EndorsementType{
		worker.EndorsementNone,
		worker.EndorsementTanker,
		worker.EndorsementHazmat,
		worker.EndorsementTankerHazmat,
		worker.EndorsementPassenger,
		worker.EndorsementDoublesTriples,
	}
	return endorsements[rand.Intn(len(endorsements))]
}

func getRandomGender() domain.Gender {
	genders := []domain.Gender{
		domain.GenderMale,
		domain.GenderFemale,
	}
	return genders[rand.Intn(len(genders))]
}

func getRandomPTOType() worker.PTOType {
	ptoTypes := []worker.PTOType{
		worker.PTOTypePersonal,
		worker.PTOTypeVacation,
		worker.PTOTypeSick,
		worker.PTOTypeHoliday,
		worker.PTOTypeBereavement,
		worker.PTOTypeMaternity,
		worker.PTOTypePaternity,
	}
	return ptoTypes[rand.Intn(len(ptoTypes))]
}

func getRandomPTOStatus() worker.PTOStatus {
	ptoStatuses := []worker.PTOStatus{
		worker.PTOStatusRequested,
		worker.PTOStatusApproved,
		worker.PTOStatusRejected,
		worker.PTOStatusCancelled,
	}
	weights := []int{30, 50, 10, 10}

	totalWeight := 0
	for _, weight := range weights {
		totalWeight += weight
	}

	randomValue := rand.Intn(totalWeight)
	cumulativeWeight := 0

	for i, weight := range weights {
		cumulativeWeight += weight
		if randomValue < cumulativeWeight {
			return ptoStatuses[i]
		}
	}

	return worker.PTOStatusRequested
}

func generatePTOsForWorker(
	workerID,
	orgID,
	buID,
	userID pulid.ID,
	now time.Time,
) []*worker.WorkerPTO {
	numPTOs := rand.Intn(6)
	if numPTOs == 0 {
		return nil
	}

	ptos := make([]*worker.WorkerPTO, 0, numPTOs)

	for range numPTOs {
		ptoType := getRandomPTOType()
		status := getRandomPTOStatus()

		startDate := generateRandomDate(
			now.AddDate(0, 0, 0),
			now.AddDate(0, 1, 0),
		)

		var duration int64
		switch ptoType {
		case worker.PTOTypeMaternity:
			duration = int64(rand.Intn(84)+42) * 24 * 60 * 60
		case worker.PTOTypePaternity:
			duration = int64(rand.Intn(14)+7) * 24 * 60 * 60
		case worker.PTOTypeBereavement:
			duration = int64(rand.Intn(3)+1) * 24 * 60 * 60
		case worker.PTOTypeVacation:
			duration = int64(rand.Intn(10)+1) * 24 * 60 * 60
		case worker.PTOTypeSick:
			duration = int64(rand.Intn(5)+1) * 24 * 60 * 60
		case worker.PTOTypePersonal:
			duration = int64(rand.Intn(3)+1) * 24 * 60 * 60
		case worker.PTOTypeHoliday:
			duration = int64(1) * 24 * 60 * 60
		default:
			duration = int64(1) * 24 * 60 * 60
		}

		endDate := startDate + duration

		var reason string
		if status == worker.PTOStatusRejected || status == worker.PTOStatusCancelled {
			reasons := []string{
				"Insufficient staffing during peak season",
				"Conflicting mandatory training scheduled",
				"Business critical project deadline",
				"Family emergency resolved",
				"Medical appointment rescheduled",
				"Insufficient notice provided",
				"Not eligible due to recent hire date",
				"Personal matter resolved",
			}
			reason = reasons[rand.Intn(len(reasons))]
		}

		pto := &worker.WorkerPTO{
			ID:             pulid.MustNew("pto_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			WorkerID:       workerID,
			Status:         status,
			Type:           ptoType,
			StartDate:      startDate,
			EndDate:        endDate,
			Reason:         reason,
		}

		if status == worker.PTOStatusApproved {
			pto.ApproverID = &userID
		}
		if status == worker.PTOStatusRejected {
			pto.RejectorID = &userID
		}

		ptos = append(ptos, pto)
	}

	return ptos
}
