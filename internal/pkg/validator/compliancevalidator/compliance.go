package compliancevalidator

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/compliance"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	HazmatExpRepo       repositories.HazmatExpirationRepository
	ShipmentControlRepo repositories.ShipmentControlRepository
}

type Validator struct {
	hazExpRepo repositories.HazmatExpirationRepository
	scp        repositories.ShipmentControlRepository
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		hazExpRepo: p.HazmatExpRepo,
		scp:        p.ShipmentControlRepo,
	}
}

// ValidateWorkerCompliance validates the overall DOT compliance for a worker's profile.
// This method orchestrates multiple specific validation checks and accumulates any errors in a MultiError.
func (v *Validator) ValidateWorkerCompliance(ctx context.Context, wp *worker.WorkerProfile, multiErr *errors.MultiError) {
	now := timeutils.NowUnix()

	// Load the shipment controls for the organization
	sc, err := v.scp.GetByOrgID(ctx, wp.OrganizationID)
	if err != nil {
		multiErr.Add("shipmentControl", errors.ErrSystemError, err.Error())
		return
	}

	v.validateDOB(wp, multiErr)

	if sc.EnforceHOSCompliance {
		v.validateMVRCompliance(wp, now, multiErr)
	}

	if sc.EnforceMedicalCertCompliance {
		v.validateMedicalCertificate(wp, multiErr)
	}

	if sc.EnforceDrugAndAlcoholCompliance {
		v.validateDrugAndAlcoholCompliance(wp, multiErr)
	}

	if sc.EnforceDriverQualificationCompliance {
		v.validateDriverQualification(wp, now, multiErr)
	}

	if sc.EnforceHazmatCompliance {
		v.validateHazmatCompliance(ctx, wp, now, multiErr)
	}
}

// validateDOB checks if the worker meets the minimum age requirement of 21 years.
// If the requirement is not met, it adds an error to the provided MultiError.
func (v *Validator) validateDOB(wp *worker.WorkerProfile, multiErr *errors.MultiError) {
	if !timeutils.IsAtLeastAge(wp.DOB, 21) {
		multiErr.Add(
			"profile.dob",
			errors.ErrComplianceViolation,
			"Worker must be at least 21 years old (49 CFR § 391.11(b)(1))",
		)
	}
}

// validateMVRCompliance validates the worker's Motor Vehicle Record (MVR) compliance.
// This includes checking the annual MVR check and ensuring the MVR renewal is not overdue.
func (v *Validator) validateMVRCompliance(wp *worker.WorkerProfile, now int64, multiErr *errors.MultiError) {
	// * Annual MVR check is required per 49 CFR 391.25(c)(2)
	if wp.LastMVRCheck < now-timeutils.YearsToSeconds(1) {
		multiErr.Add(
			"profile.lastMVRCheck",
			errors.ErrComplianceViolation,
			"Annual MVR Check is overdue (49 CFR § 391.25(c)(2))",
		)
	}

	// * MVR Due Date Check
	if wp.MVRDueDate != nil && *wp.MVRDueDate < now {
		multiErr.Add(
			"profile.mvrDueDate",
			errors.ErrComplianceViolation,
			"MVR renewal is overdue (49 CFR § 391.25(c)(2))",
		)
	}
}

// validateMedicalCertificate checks if the worker's medical examination is up-to-date.
// It validates that the medical examination occurs at least every 24 months.
func (v *Validator) validateMedicalCertificate(wp *worker.WorkerProfile, multiErr *errors.MultiError) {
	if wp.PhysicalDueDate == nil || *wp.PhysicalDueDate == 0 {
		return
	}

	// * If the last physical was more than 24 months ago, then it is overdue
	if *wp.PhysicalDueDate < timeutils.YearsAgoUnix(2) {
		multiErr.Add(
			"profile.physicalDueDate",
			errors.ErrComplianceViolation,
			"Medical examination is required at least every 24 months (49 CFR § 391.45)",
		)
	}
}

// validateDrugAndAlcoholCompliance ensures the worker underwent a pre-employment drug test.
// It verifies that the drug test date is before the worker's hire date.
func (v *Validator) validateDrugAndAlcoholCompliance(wp *worker.WorkerProfile, multiErr *errors.MultiError) {
	// * Ensure the last drug test was before the hire date or on the hire date
	// * otherwise, it is not compliant
	if wp.LastDrugTest <= wp.HireDate {
		return
	}

	multiErr.Add(
		"profile.lastDrugTest",
		errors.ErrComplianceViolation,
		"Pre-employment drug test is required (49 CFR § 382.301(a))",
	)
}

// validateDriverQualification checks if the worker's Commercial Driver's License (CDL) is valid and not expired.
// If the license is expired, it adds an error to the provided MultiError.
func (v *Validator) validateDriverQualification(wp *worker.WorkerProfile, now int64, multiErr *errors.MultiError) {
	if wp.LicenseExpiry < now {
		multiErr.Add(
			"profile.licenseExpiry",
			errors.ErrComplianceViolation,
			"Commercial driver's license is expired (49 CFR § 391.11(b)(5))",
		)
	}
}

// validateHazmatCompliance validates compliance with hazmat endorsement requirements.
// This includes verifying that the hazmat endorsement is not expired and does not exceed the allowed validity period.
func (v *Validator) validateHazmatCompliance(ctx context.Context, wp *worker.WorkerProfile, now int64, multiErr *errors.MultiError) {
	// * Only validate hazmat requirements if the endorsement includes hazmat
	if wp.Endorsement != worker.EndorsementHazmat && wp.Endorsement != worker.EndorsementTankerHazmat {
		return
	}

	// * Check if hazmat endorsement is already expired
	if wp.HazmatExpiry < now {
		multiErr.Add(
			"profile.hazmatExpiry",
			errors.ErrComplianceViolation,
			"Hazmat endorsement is expired (49 CFR § 383.93)",
		)
	}

	// * Get state specific expiration or use federal standard
	var yearsAllowed int64
	exp, err := v.hazExpRepo.GetHazmatExpirationByStateID(ctx, wp.LicenseStateID)
	if err != nil {
		if !eris.Is(err, sql.ErrNoRows) {
			multiErr.Add(
				"profile.hazmatExpiry",
				errors.ErrSystemError,
				fmt.Sprintf("Failed to get hazmat expiration: %s", err.Error()),
			)
			return
		}
		// * Use federal standard if no state-specific rule exists
		yearsAllowed = compliance.DefaultHazmatExpirationYears
	} else {
		yearsAllowed = int64(exp.Years)
	}

	// Calculate the maximum allowed expiration date from today
	nowTime := time.Unix(now, 0)
	maxAllowedTime := nowTime.AddDate(int(yearsAllowed), 0, 0)
	maxAllowedUnix := maxAllowedTime.Unix()

	// * Check if the hazmat expiry exceeds the maximum allowed date
	if wp.HazmatExpiry > maxAllowedUnix {
		log.Debug().Int64("hazmatExpiry", wp.HazmatExpiry).Msg("hazmat expiry is greater than max allowed")
		multiErr.Add(
			"profile.hazmatExpiry",
			errors.ErrComplianceViolation,
			fmt.Sprintf("Hazmat endorsement exceeds the maximum allowed period of %d years (%s)",
				yearsAllowed,
				compliance.HazmatComplianceCode,
			),
		)
	}
}
