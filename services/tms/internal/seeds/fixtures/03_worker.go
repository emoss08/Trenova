package fixtures

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type WorkerSeed struct {
}

var Worker = Seeds.Add(seed.Seed{
	Name:        "worker",
	Kind:        seed.Fixture,
	Description: "Creates worker data for development",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&WorkerSeed{}).Run,
})

type workerDef struct {
	firstName  string
	lastName   string
	gender     worker.Gender
	workerType worker.WorkerType
	driverType worker.DriverType
	city       string
	address    string
	postalCode string
	stateAbbr  string
	email      string
	phone      string
	cdlClass   worker.CDLClass
	endorsment worker.EndorsementType
	license    string
}

func (s *WorkerSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	now := timeutils.NowUnix()

	defs := []workerDef{
		{
			"John",
			"Smith",
			worker.GenderMale,
			worker.WorkerTypeEmployee,
			worker.DriverTypeOTR,
			"Los Angeles",
			"123 Main St",
			"90001",
			"CA",
			"john.smith@example.com",
			"+15550101001",
			worker.CDLClassA,
			worker.EndorsementTypeNone,
			"CA12345678",
		},
		{
			"Jane",
			"Doe",
			worker.GenderFemale,
			worker.WorkerTypeEmployee,
			worker.DriverTypeRegional,
			"Dallas",
			"456 Oak Ave",
			"75201",
			"TX",
			"jane.doe@example.com",
			"+15550102001",
			worker.CDLClassA,
			worker.EndorsementTypeHazmat,
			"TX98765432",
		},
		{
			"Mike",
			"Johnson",
			worker.GenderMale,
			worker.WorkerTypeContractor,
			worker.DriverTypeLocal,
			"Chicago",
			"789 Elm Blvd",
			"60601",
			"IL",
			"mike.j@example.com",
			"+15550103001",
			worker.CDLClassB,
			worker.EndorsementTypeTanker,
			"IL11223344",
		},
		{
			"Sarah",
			"Williams",
			worker.GenderFemale,
			worker.WorkerTypeEmployee,
			worker.DriverTypeTeam,
			"Phoenix",
			"321 Pine Dr",
			"85001",
			"AZ",
			"sarah.w@example.com",
			"+15550104001",
			worker.CDLClassA,
			worker.EndorsementTypeTankerHazmat,
			"AZ55667788",
		},
		{
			"Robert",
			"Brown",
			worker.GenderMale,
			worker.WorkerTypeEmployee,
			worker.DriverTypeOTR,
			"Denver",
			"654 Cedar Ln",
			"80201",
			"CO",
			"robert.b@example.com",
			"+15550105001",
			worker.CDLClassA,
			worker.EndorsementTypeNone,
			"CO99887766",
		},
	}

	ptoTypes := []worker.PTOType{
		worker.PTOTypeVacation,
		worker.PTOTypeSick,
		worker.PTOTypePersonal,
		worker.PTOTypeHoliday,
		worker.PTOTypeBereavement,
		worker.PTOTypeMaternity,
		worker.PTOTypePaternity,
	}

	ptoReasons := []string{
		"Family vacation",
		"Not feeling well",
		"Personal appointment",
		"National holiday",
		"Family emergency",
		"Doctor visit",
		"Home maintenance",
		"Wedding attendance",
		"Moving day",
		"Jury duty",
	}

	daySeconds := int64(86400)

	for workerIdx, def := range defs {
		state, stateErr := stateByAbbreviation(ctx, db, def.stateAbbr)
		if stateErr != nil {
			return fmt.Errorf("get state %s: %w", def.stateAbbr, stateErr)
		}

		workerID := pulid.MustNew("wrk_")
		profileID := pulid.MustNew("wrkp_")

		dob := now - (30 * 365 * daySeconds)
		hireDate := now - (2 * 365 * daySeconds)
		licenseExpiry := now + (3 * 365 * daySeconds)

		var hazmatExpiry *int64
		if def.endorsment.RequiresHazmatExpiry() {
			exp := now + (2 * 365 * daySeconds)
			hazmatExpiry = &exp
		}

		w := &worker.Worker{
			ID:                   workerID,
			BusinessUnitID:       org.BusinessUnitID,
			OrganizationID:       org.ID,
			StateID:              state.ID,
			Status:               domaintypes.StatusActive,
			Type:                 def.workerType,
			DriverType:           def.driverType,
			FirstName:            def.firstName,
			LastName:             def.lastName,
			AddressLine1:         def.address,
			City:                 def.city,
			PostalCode:           def.postalCode,
			Email:                def.email,
			PhoneNumber:          def.phone,
			Gender:               def.gender,
			CanBeAssigned:        true,
			AvailableForDispatch: true,
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		if err = workerdb.Workers.Create(ctx, db, w); err != nil {
			return fmt.Errorf("insert worker %s %s: %w", def.firstName, def.lastName, err)
		}

		profile := &worker.WorkerProfile{
			ID:             profileID,
			WorkerID:       workerID,
			BusinessUnitID: org.BusinessUnitID,
			OrganizationID: org.ID,
			LicenseStateID: state.ID,
			DOB:            dob,
			LicenseNumber:  def.license,
			CDLClass:       def.cdlClass,
			Endorsement:    def.endorsment,
			HazmatExpiry:   hazmatExpiry,
			LicenseExpiry:  licenseExpiry,
			HireDate:       hireDate,
			// Pending, not Compliant: no credential exists yet at this
			// point in the seed order, so claiming compliance here is
			// asserting a fact nothing has established. The credential
			// seed derives the real status from the domain roll-up once
			// the credentials are in.
			ComplianceStatus:   worker.ComplianceStatusPending,
			IsQualified:        true,
			CreatedAt:          now,
			UpdatedAt:          now,
			TrainingHealth:     worker.TrainingHealthCurrent,
			SafetyRating:       worker.SafetyRatingExcellent,
			SafetyScore:        100,
			DrugAlcoholStatus:  worker.DrugAlcoholUnknown,
			ReturnToDutyStatus: worker.ReturnToDutyNotRequired,
		}

		if err = workerdb.WorkerProfiles.Create(ctx, db, profile); err != nil {
			return fmt.Errorf(
				"insert worker profile for %s %s: %w",
				def.firstName,
				def.lastName,
				err,
			)
		}

		ptoPerWorker := 80
		ptos := make([]*worker.WorkerPTO, 0, ptoPerWorker)
		seed := int64(13_579 + (workerIdx * 97))
		rng := rand.New(rand.NewSource(seed))
		typeShift := workerIdx % len(ptoTypes)

		for i := range ptoPerWorker {
			ptoID := pulid.MustNew("wrkpto_")
			baseOffsetDays := int64(-240 + (i * 6) + (workerIdx * 3))
			jitterDays := int64(
				rng.Intn(5) - 2,
			) //nolint:gosec // deterministic seed data generation
			startDate := now + ((baseOffsetDays + jitterDays) * daySeconds)
			durationDays := int64(
				1 + rng.Intn(5),
			) //nolint:gosec // deterministic seed data generation
			endDate := startDate + (daySeconds * durationDays)
			typeIdx := (i + typeShift + rng.Intn(len(ptoTypes))) % len(
				ptoTypes,
			) //nolint:gosec // deterministic seed data generation

			roll := rng.Intn(100) //nolint:gosec // deterministic seed data generation
			var status worker.PTOStatus
			switch {
			case roll < 50:
				status = worker.PTOStatusApproved
			case roll < 80:
				status = worker.PTOStatusRequested
			case roll < 93:
				status = worker.PTOStatusRejected
			default:
				status = worker.PTOStatusCancelled
			}

			pto := &worker.WorkerPTO{
				ID:             ptoID,
				WorkerID:       workerID,
				BusinessUnitID: org.BusinessUnitID,
				OrganizationID: org.ID,
				Status:         status,
				Type:           ptoTypes[typeIdx],
				StartDate:      startDate,
				EndDate:        endDate,
				Days:           pto.ComputePTODays(startDate, endDate, nil, true, nil),
				Reason:         ptoReasons[rng.Intn(len(ptoReasons))], //nolint:gosec // deterministic seed data generation
				CreatedAt:      now,
				UpdatedAt:      now,
			}

			ptos = append(ptos, pto)
		}

		if err = workerdb.WorkerPTOs.CreateMany(ctx, db, ptos); err != nil {
			return fmt.Errorf(
				"insert PTO records for %s %s: %w",
				def.firstName,
				def.lastName,
				err,
			)
		}

	}

	return nil
}
