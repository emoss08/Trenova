/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package fixtures

import (
	"math/rand"
	"time"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jaswdr/faker/v2"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dbfixture"
	"golang.org/x/net/context"
)

// generateRandomDate generates a random Unix timestamp between two dates
func generateRandomDate(start, end time.Time) int64 {
	minimum := start.Unix()
	maximum := end.Unix()
	return rand.Int63n(maximum-minimum) + minimum
}

// getRandomEndorsement returns a random worker endorsement type
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
	// Weight towards approved and requested statuses (more common)
	weights := []int{30, 50, 10, 10} // Requested: 30%, Approved: 50%, Rejected: 10%, Cancelled: 10%

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

	return worker.PTOStatusRequested // fallback
}

func generatePTOsForWorker(
	workerID pulid.ID,
	orgID, buID pulid.ID,
	now time.Time,
) []*worker.WorkerPTO {
	// Generate 0-5 PTOs per worker (weighted towards fewer PTOs)
	numPTOs := rand.Intn(6) // 0-5 PTOs
	if numPTOs == 0 {
		return nil
	}

	ptos := make([]*worker.WorkerPTO, 0, numPTOs)

	for i := 0; i < numPTOs; i++ {
		ptoType := getRandomPTOType()
		status := getRandomPTOStatus()

		// Generate random start date (can be past, present, or future)
		startDate := generateRandomDate(
			now.AddDate(0, 0, 0),
			now.AddDate(0, 1, 0),
		)

		// Generate duration based on PTO type
		var duration int64
		switch ptoType {
		case worker.PTOTypeMaternity:
			duration = int64(rand.Intn(84)+42) * 24 * 60 * 60 // 6-18 weeks
		case worker.PTOTypePaternity:
			duration = int64(rand.Intn(14)+7) * 24 * 60 * 60 // 1-3 weeks
		case worker.PTOTypeBereavement:
			duration = int64(rand.Intn(3)+1) * 24 * 60 * 60 // 1-3 days
		case worker.PTOTypeVacation:
			duration = int64(rand.Intn(10)+1) * 24 * 60 * 60 // 1-10 days
		case worker.PTOTypeSick:
			duration = int64(rand.Intn(5)+1) * 24 * 60 * 60 // 1-5 days
		case worker.PTOTypePersonal:
			duration = int64(rand.Intn(3)+1) * 24 * 60 * 60 // 1-3 days
		case worker.PTOTypeHoliday:
			duration = int64(1) * 24 * 60 * 60 // 1 day
		default:
			duration = int64(1) * 24 * 60 * 60 // 1 day fallback
		}

		endDate := startDate + duration

		// Generate reason for rejected/cancelled statuses
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

		// Add approver/rejector for approved/rejected status
		if status == worker.PTOStatusApproved {
			// In a real system, this would be a manager ID
			// For now, we'll leave it nil since we don't have user fixtures here
			pto.ApproverID = nil
		}
		if status == worker.PTOStatusRejected {
			pto.RejectorID = nil
		}

		ptos = append(ptos, pto)
	}

	return ptos
}

func LoadWorkers(ctx context.Context, db *bun.DB, fixture *dbfixture.Fixture) error {
	org := fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	cali := fixture.MustRow("UsState.ca").(*usstate.UsState)

	const numWorkers = 1000
	workers := make([]*worker.Worker, 0, numWorkers)
	profiles := make([]*worker.WorkerProfile, 0, numWorkers)
	ptos := make([]*worker.WorkerPTO, 0, numWorkers*3) // Average 3 PTOs per worker

	fake := faker.New()
	now := time.Now()

	// Generate worker data
	for range numWorkers {
		// Generate random state for both address and license

		workerID := pulid.MustNew("wrk_")

		// Create worker
		wrk := &worker.Worker{
			ID:             workerID,
			BusinessUnitID: bu.ID,
			OrganizationID: org.ID,
			StateID:        cali.ID,
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

		// Create worker profile
		endorsementType := getRandomEndorsement()
		hireDate := generateRandomDate(now.AddDate(-5, 0, 0), now.AddDate(0, -1, 0))
		licenseExpiry := generateRandomDate(now, now.AddDate(4, 0, 0))

		profile := &worker.WorkerProfile{
			ID:             pulid.MustNew("wp_"),
			WorkerID:       workerID,
			BusinessUnitID: bu.ID,
			OrganizationID: org.ID,
			LicenseStateID: cali.ID,
			DOB:            generateRandomDate(now.AddDate(-65, 0, 0), now.AddDate(-21, 0, 0)),
			LicenseNumber:  fake.RandomStringWithLength(8),
			Endorsement:    endorsementType,
			LicenseExpiry:  licenseExpiry,
			HireDate:       hireDate,
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

		// Set HazmatExpiry if endorsement includes Hazmat
		if endorsementType == worker.EndorsementHazmat ||
			endorsementType == worker.EndorsementTankerHazmat {
			hazmatExpiry := generateRandomDate(now, now.AddDate(2, 0, 0))
			profile.HazmatExpiry = hazmatExpiry
		}

		workers = append(workers, wrk)
		profiles = append(profiles, profile)

		// Generate PTOs for this worker
		workerPTOs := generatePTOsForWorker(workerID, org.ID, bu.ID, now)
		ptos = append(ptos, workerPTOs...)
	}

	// Begin transaction
	err := db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// Bulk insert workers
		if _, err := tx.NewInsert().Model(&workers).Exec(c); err != nil {
			return eris.Wrap(err, "failed to bulk insert workers")
		}

		// Bulk insert worker profiles
		if _, err := tx.NewInsert().Model(&profiles).Exec(c); err != nil {
			return eris.Wrap(err, "failed to bulk insert worker profiles")
		}

		// Bulk insert worker PTOs (only if we have any)
		if len(ptos) > 0 {
			if _, err := tx.NewInsert().Model(&ptos).Exec(c); err != nil {
				return eris.Wrap(err, "failed to bulk insert worker PTOs")
			}
		}

		return nil
	})
	if err != nil {
		return eris.Wrap(err, "failed to run transaction")
	}

	return nil
}
