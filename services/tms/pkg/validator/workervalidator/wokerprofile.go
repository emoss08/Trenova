package workervalidator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/hazmatexpiration"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	"go.uber.org/fx"
)

type WorkerProfileValidatorParams struct {
	fx.In

	HazmatExpRepo           repositories.HazmatExpirationRepository
	DispatchControlRepo     repositories.DispatchControlRepository
	ValidationEngineFactory framework.ValidationEngineFactory
}

type WorkerProfileValidator struct {
	engine              framework.ValidationEngineFactory
	hazmatExpRepo       repositories.HazmatExpirationRepository
	dispatchControlRepo repositories.DispatchControlRepository
}

func NewWorkerProfileValidator(p WorkerProfileValidatorParams) *WorkerProfileValidator {
	return &WorkerProfileValidator{
		engine:              p.ValidationEngineFactory,
		hazmatExpRepo:       p.HazmatExpRepo,
		dispatchControlRepo: p.DispatchControlRepo,
	}
}

func (v *WorkerProfileValidator) Validate(
	ctx context.Context,
	wp *worker.WorkerProfile,
	me *errortypes.MultiError,
) {
	engine := v.engine.CreateEngine().
		ForField("profile").
		WithParent(me)

	engine.AddRule(
		framework.NewConcreteRule("worker_profile_validation").
			WithStage(framework.ValidationStageCompliance).
			WithPriority(framework.ValidationPriorityHigh).
			WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
				v.validateWorkerCompliance(ctx, wp, multiErr)
				return nil
			}),
	)

	engine.ValidateInto(ctx, me)
}

func (v *WorkerProfileValidator) validateWorkerCompliance(
	ctx context.Context,
	wp *worker.WorkerProfile,
	me *errortypes.MultiError,
) {
	now := utils.NowUnix()

	sc, err := v.dispatchControlRepo.GetByOrgID(ctx, wp.OrganizationID)
	if err != nil {
		me.Add(
			"dispatchControl",
			errortypes.ErrSystemError,
			err.Error(),
		)
		return
	}

	v.validateDOB(wp, me)

	if sc.EnforceHOSCompliance {
		v.validateMVRCompliance(wp, now, me)
	}

	if sc.EnforceMedicalCertCompliance {
		v.validateMedicalCertificate(wp, me)
	}

	if sc.EnforceDrugAndAlcoholCompliance {
		v.validateDrugAndAlcoholCompliance(wp, me)
	}

	if sc.EnforceDriverQualificationCompliance {
		v.validateDriverQualification(wp, now, me)
	}

	if sc.EnforceHazmatCompliance {
		v.validateHazmatCompliance(ctx, wp, now, me)
	}
}

func (v *WorkerProfileValidator) validateDOB(wp *worker.WorkerProfile, me *errortypes.MultiError) {
	if utils.IsAtLeastAge(wp.DOB, 21) {
		me.Add(
			"dob",
			errortypes.ErrComplianceViolation,
			"Worker must be at least 21 years old",
		)
	}
}

func (v *WorkerProfileValidator) validateMVRCompliance(
	wp *worker.WorkerProfile,
	now int64,
	me *errortypes.MultiError,
) {
	if wp.LastMVRCheck < now-utils.YearsAgoToSeconds(1) {
		me.Add(
			"lastMvrCheck",
			errortypes.ErrComplianceViolation,
			"Annual MVR Check is overdue (49 CFR § 391.25(c)(2))",
		)
	}
	if wp.MVRDueDate != nil && *wp.MVRDueDate < now {
		me.Add(
			"mvrDueDate",
			errortypes.ErrComplianceViolation,
			"MVR Due Date is past (49 CFR § 391.25(c)(2))",
		)
	}
}

func (v *WorkerProfileValidator) validateMedicalCertificate(
	wp *worker.WorkerProfile,
	me *errortypes.MultiError,
) {
	if wp.PhysicalDueDate == nil || *wp.PhysicalDueDate == 0 {
		return
	}

	// * If the last physical was more than 24 months ago, then it is overdue
	if *wp.PhysicalDueDate < utils.YearsAgoUnix(2) {
		me.Add(
			"physicalDueDate",
			errortypes.ErrComplianceViolation,
			"Medical examination is required at least every 24 months (49 CFR § 391.45)",
		)
	}
}

func (v *WorkerProfileValidator) validateDrugAndAlcoholCompliance(
	wp *worker.WorkerProfile,
	me *errortypes.MultiError,
) {
	if wp.LastDrugTest <= wp.HireDate {
		return
	}

	me.Add(
		"lastDrugTest",
		errortypes.ErrComplianceViolation,
		"Pre-employment drug test is required (49 CFR § 382.301(a))",
	)
}

func (v *WorkerProfileValidator) validateDriverQualification(
	wp *worker.WorkerProfile,
	now int64,
	multiErr *errortypes.MultiError,
) {
	if wp.LicenseExpiry < now {
		multiErr.Add(
			"profile.licenseExpiry",
			errortypes.ErrComplianceViolation,
			"Commercial driver's license is expired (49 CFR § 391.11(b)(5))",
		)
	}
}

func (v *WorkerProfileValidator) validateHazmatCompliance(
	ctx context.Context,
	wp *worker.WorkerProfile,
	now int64,
	me *errortypes.MultiError,
) {
	// ! Only validate hazmat requirements if the endorsement includes hazmat
	if wp.Endorsement != worker.EndorsementHazmat &&
		wp.Endorsement != worker.EndorsementTankerHazmat {
		return
	}

	// ! Check if hazmat endorsement is already expired
	if wp.HazmatExpiry < now {
		me.Add(
			"hazmatExpiry",
			errortypes.ErrComplianceViolation,
			"Hazmat endorsement is expired (49 CFR § 383.93)",
		)
	}

	// ! Get state specific expiration or use federal standard
	var yearsAllowed int8
	exp, err := v.hazmatExpRepo.GetHazmatExpirationByStateID(ctx, wp.LicenseStateID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			me.Add(
				"hazmatExpiry",
				errortypes.ErrSystemError,
				fmt.Sprintf("Failed to get hazmat expiration: %s", err.Error()),
			)
			return
		}
		// ! Use federal standard if no state-specific rule exists
		yearsAllowed = hazmatexpiration.DefaultHazmatExpirationYears
	} else {
		yearsAllowed = exp.Years
	}

	maxAllowedUnix := utils.MaxAllowedUnix(now, yearsAllowed)

	// ! Check if the hazmat expiry exceeds the maximum allowed date
	if wp.HazmatExpiry > maxAllowedUnix {
		me.Add(
			"hazmatExpiry",
			errortypes.ErrComplianceViolation,
			fmt.Sprintf("Hazmat endorsement exceeds the maximum allowed period of %d years (%s)",
				yearsAllowed,
				hazmatexpiration.HazmatComplianceCode,
			),
		)
	}
}
