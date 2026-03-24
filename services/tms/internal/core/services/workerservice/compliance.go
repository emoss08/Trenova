package workerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	MinDriverAgeInterstate    = 21
	MedicalCertValidityMonths = 24
	MVRCheckValidityMonths    = 12
	HazmatCertValidityYears   = 5
)

func createAgeComplianceRule(
	dcRepo repositories.DispatchControlRepository,
) validationframework.TenantedRule[*worker.Worker] {
	return validationframework.NewTenantedRule[*worker.Worker]("age_compliance_49cfr_391.11").
		OnBoth().
		WithStage(validationframework.ValidationStageCompliance).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			w *worker.Worker,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			dc, err := dcRepo.GetOrCreate(ctx, valCtx.OrganizationID, valCtx.BusinessUnitID)
			if err != nil {
				return err
			}

			if !dc.EnforceDriverQualificationCompliance {
				return nil
			}

			if w.Profile == nil {
				return nil
			}

			if !timeutils.IsAtLeastAge(w.Profile.DOB, MinDriverAgeInterstate) {
				errCode := getComplianceErrorCode(dc.ComplianceEnforcementLevel)
				multiErr.Add(
					"profile.dob",
					errCode,
					"Driver must be at least 21 years old for interstate commerce (49 CFR 391.11(b)(1))",
				)
			}

			return nil
		})
}

func createCDLComplianceRule(
	dcRepo repositories.DispatchControlRepository,
) validationframework.TenantedRule[*worker.Worker] {
	return validationframework.NewTenantedRule[*worker.Worker]("cdl_compliance_49cfr_391.11").
		OnBoth().
		WithStage(validationframework.ValidationStageCompliance).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			w *worker.Worker,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			dc, err := dcRepo.GetOrCreate(ctx, valCtx.OrganizationID, valCtx.BusinessUnitID)
			if err != nil {
				return err
			}

			if !dc.EnforceDriverQualificationCompliance {
				return nil
			}

			if w.Profile == nil {
				return nil
			}

			if timeutils.IsExpired(w.Profile.LicenseExpiry) {
				errCode := getComplianceErrorCode(dc.ComplianceEnforcementLevel)
				multiErr.Add("profile.licenseExpiry", errCode,
					"Commercial driver's license is expired (49 CFR 391.11(b)(5))")
			}

			return nil
		})
}

func createMedicalCertComplianceRule(
	dcRepo repositories.DispatchControlRepository,
) validationframework.TenantedRule[*worker.Worker] {
	return validationframework.
		NewTenantedRule[*worker.Worker]("medical_cert_compliance_49cfr_391.45").
		OnBoth().
		WithStage(validationframework.ValidationStageCompliance).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			w *worker.Worker,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			dc, err := dcRepo.GetOrCreate(ctx, valCtx.OrganizationID, valCtx.BusinessUnitID)
			if err != nil {
				return err
			}

			if !dc.EnforceMedicalCertCompliance {
				return nil
			}

			if w.Profile == nil {
				return nil
			}

			errCode := getComplianceErrorCode(dc.ComplianceEnforcementLevel)

			if w.Profile.MedicalCardExpiry != nil &&
				timeutils.IsExpired(*w.Profile.MedicalCardExpiry) {
				multiErr.Add(
					"profile.medicalCardExpiry",
					errCode,
					"Medical certificate is expired (49 CFR 391.45)",
				)
			}

			if w.Profile.PhysicalDueDate != nil && timeutils.IsOverdue(*w.Profile.PhysicalDueDate) {
				multiErr.Add(
					"profile.physicalDueDate",
					errCode,
					"Physical examination is overdue. Medical examination required at least every 24 months (49 CFR 391.45)",
				)
			}

			return nil
		})
}

func createMVRComplianceRule(
	dcRepo repositories.DispatchControlRepository,
) validationframework.TenantedRule[*worker.Worker] {
	return validationframework.NewTenantedRule[*worker.Worker]("mvr_compliance_49cfr_391.25").
		OnBoth().
		WithStage(validationframework.ValidationStageCompliance).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			w *worker.Worker,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			dc, err := dcRepo.GetOrCreate(ctx, valCtx.OrganizationID, valCtx.BusinessUnitID)
			if err != nil {
				return err
			}

			if !dc.EnforceHOSCompliance {
				return nil
			}

			if w.Profile == nil {
				return nil
			}

			errCode := getComplianceErrorCode(dc.ComplianceEnforcementLevel)

			if w.Profile.LastMVRCheck > 0 &&
				!timeutils.IsWithinMonths(w.Profile.LastMVRCheck, MVRCheckValidityMonths) {
				multiErr.Add(
					"profile.lastMvrCheck",
					errCode,
					"Annual MVR check is overdue (49 CFR 391.25(c)(2))",
				)
			}

			if w.Profile.MVRDueDate != nil && timeutils.IsOverdue(*w.Profile.MVRDueDate) {
				multiErr.Add(
					"profile.mvrDueDate",
					errCode,
					"MVR due date has passed (49 CFR 391.25(c)(2))",
				)
			}

			return nil
		})
}

func createDrugTestComplianceRule(
	dcRepo repositories.DispatchControlRepository,
) validationframework.TenantedRule[*worker.Worker] {
	return validationframework.NewTenantedRule[*worker.Worker](
		"drug_test_compliance_49cfr_382.301",
	).
		OnBoth().
		WithStage(validationframework.ValidationStageCompliance).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			w *worker.Worker,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			dc, err := dcRepo.GetOrCreate(ctx, valCtx.OrganizationID, valCtx.BusinessUnitID)
			if err != nil {
				return err
			}

			if !dc.EnforceDrugAndAlcoholCompliance {
				return nil
			}

			if w.Profile == nil {
				return nil
			}

			if w.Profile.LastDrugTest > 0 && w.Profile.LastDrugTest <= w.Profile.HireDate {
				errCode := getComplianceErrorCode(dc.ComplianceEnforcementLevel)
				multiErr.Add("profile.lastDrugTest", errCode,
					"Pre-employment drug test is required before hire date (49 CFR 382.301(a))")
			}

			return nil
		})
}

func createHazmatComplianceRule(
	dcRepo repositories.DispatchControlRepository,
) validationframework.TenantedRule[*worker.Worker] {
	return validationframework.NewTenantedRule[*worker.Worker]("hazmat_compliance_49cfr_383.93").
		OnBoth().
		WithStage(validationframework.ValidationStageCompliance).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			w *worker.Worker,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			dc, err := dcRepo.GetOrCreate(ctx, valCtx.OrganizationID, valCtx.BusinessUnitID)
			if err != nil {
				return err
			}

			if !dc.EnforceHazmatCompliance {
				return nil
			}

			if w.Profile == nil {
				return nil
			}

			if !w.Profile.Endorsement.RequiresHazmatExpiry() {
				return nil
			}

			errCode := getComplianceErrorCode(dc.ComplianceEnforcementLevel)

			if w.Profile.HazmatExpiry == nil || *w.Profile.HazmatExpiry <= 0 {
				multiErr.Add("profile.hazmatExpiry", errCode,
					"Hazmat expiry date is required for H or X endorsement (49 CFR 383.93)")
				return nil
			}

			if timeutils.IsExpired(*w.Profile.HazmatExpiry) {
				multiErr.Add("profile.hazmatExpiry", errCode,
					"Hazmat endorsement is expired (49 CFR 383.93)")
			}

			maxAllowed := timeutils.MaxAllowedUnix(timeutils.NowUnix(), HazmatCertValidityYears)
			if *w.Profile.HazmatExpiry > maxAllowed {
				multiErr.Add("profile.hazmatExpiry", errCode,
					"Hazmat endorsement exceeds maximum validity period of 5 years (49 CFR 383.93)")
			}

			return nil
		})
}

func getComplianceErrorCode(level dispatchcontrol.ComplianceEnforcementLevel) errortypes.ErrorCode {
	if level.ShouldBlock() {
		return errortypes.ErrComplianceViolation
	}
	return errortypes.ErrInvalid
}
