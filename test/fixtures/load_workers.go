// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package fixtures

import (
	"math/rand"
	"time"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/types/pulid"
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

func LoadWorkers(ctx context.Context, db *bun.DB, fixture *dbfixture.Fixture) error {
	org := fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	cali := fixture.MustRow("UsState.ca").(*usstate.UsState)

	const numWorkers = 1000
	workers := make([]*worker.Worker, 0, numWorkers)
	profiles := make([]*worker.WorkerProfile, 0, numWorkers)

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

		return nil
	})
	if err != nil {
		return eris.Wrap(err, "failed to run transaction")
	}

	return nil
}
