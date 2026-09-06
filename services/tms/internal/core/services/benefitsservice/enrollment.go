package benefitsservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// EnrollRequest puts a worker on a plan, or records that they declined it.
type EnrollRequest struct {
	TenantInfo   pagination.TenantInfo
	WorkerID     pulid.ID
	PlanID       pulid.ID
	CoverageTier driverpay.CoverageTier
	// Waive records a decision not to take the cover. A waiver still produces
	// an enrollment row, because "declined" and "nobody asked" are different
	// facts and only one of them is a problem at audit.
	Waive        bool
	WaivedReason string
	// EmployeeCostMinor overrides the tier-scaled price for a carrier that
	// prices each tier separately. Zero means take the plan's own arithmetic.
	EmployeeCostMinor int64
	EffectiveFrom     int64
	Notes             string
	UserID            pulid.ID
}

// Enroll puts a worker on a plan and, when they are taking the cover, opens the
// recurring deduction that collects their contribution.
//
// The deduction is an ordinary one: same table, same settlement path, same
// reversal on a void. Only its kind and priority mark it out, so a benefit is
// taken after a garnishment and before nothing.
func (s *Service) Enroll(
	ctx context.Context,
	req *EnrollRequest,
) (*driverpay.WorkerBenefitEnrollment, error) {
	log := s.l.With(
		zap.String("operation", "Enroll"),
		zap.String("workerId", req.WorkerID.String()),
	)

	plan, err := s.repo.GetPlanByID(ctx, &repositories.GetBenefitPlanByIDRequest{
		ID:         req.PlanID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	// A plan nobody is allowed to join is not a plan somebody can be put on.
	if plan.Status != "Active" {
		return nil, errortypes.NewValidationError(
			"benefitPlanId",
			errortypes.ErrInvalidOperation,
			"That plan is archived and cannot be enrolled in",
		)
	}

	effectiveFrom := req.EffectiveFrom
	if effectiveFrom <= 0 {
		effectiveFrom = timeutils.NowUnix()
	}

	tier := req.CoverageTier
	if tier == "" {
		tier = driverpay.TierEmployee
	}

	employeeCost := req.EmployeeCostMinor
	if employeeCost <= 0 {
		employeeCost = plan.CostForTier(tier)
	}

	entity := &driverpay.WorkerBenefitEnrollment{
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		WorkerID:          req.WorkerID,
		BenefitPlanID:     plan.ID,
		Status:            driverpay.EnrollmentActive,
		CoverageTier:      tier,
		EffectiveFrom:     effectiveFrom,
		EmployeeCostMinor: employeeCost,
		EmployerCostMinor: plan.EmployerCostMinor,
		Notes:             req.Notes,
		EnrolledByID:      req.UserID,
	}
	if req.Waive {
		// A waiver costs nothing and takes nothing: carrying the plan's price
		// on it would show up in the employer's total as cover nobody has.
		entity.Status = driverpay.EnrollmentWaived
		entity.WaivedReason = req.WaivedReason
		entity.EmployeeCostMinor = 0
		entity.EmployerCostMinor = 0
	}
	entity.Normalise()

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if entity.Status.Deducts() && entity.EmployeeCostMinor > 0 {
		deduction, dErr := s.openContribution(ctx, req.TenantInfo, entity, plan)
		if dErr != nil {
			return nil, dErr
		}
		entity.RecurringDeductionID = deduction.ID
	}

	created, err := s.repo.CreateEnrollment(ctx, entity)
	if err != nil {
		log.Error("failed to create benefit enrollment", zap.Error(err))
		return nil, err
	}

	comment := "Enrolled in " + plan.Name
	if created.Status == driverpay.EnrollmentWaived {
		comment = "Declined " + plan.Name
	}
	s.audit(
		created.GetResourceID(), permission.OpCreate, req.UserID, req.TenantInfo,
		created, nil, comment,
	)

	return created, nil
}

func (s *Service) openContribution(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	enrollment *driverpay.WorkerBenefitEnrollment,
	plan *driverpay.BenefitPlan,
) (*driverpay.RecurringDeduction, error) {
	deduction := &driverpay.RecurringDeduction{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		WorkerID:       enrollment.WorkerID,
		PayCodeID:      plan.PayCodeID,
		Status:         driverpay.DeductionStatusActive,
		Kind:           driverpay.DeductionKindBenefit,
		Priority:       driverpay.DeductionKindBenefit.DefaultPriority(),
		Frequency:      driverpay.DeductionFrequencyEverySettlement,
		Description:    deductionDescription(plan, enrollment.CoverageTier),
		AmountMinor:    enrollment.EmployeeCostMinor,
		StartDate:      enrollment.EffectiveFrom,
		CurrencyCode:   plan.CurrencyCode,
		CreatedByID:    enrollment.EnrolledByID,
	}
	// No cap: a benefit contribution runs for as long as the cover does, and
	// the enrollment ending is what stops it.
	multiErr := errortypes.NewMultiError()
	deduction.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.deductionRepo.Create(ctx, deduction)
}

// EndEnrollmentRequest closes cover.
type EndEnrollmentRequest struct {
	TenantInfo  pagination.TenantInfo
	ID          pulid.ID
	EffectiveTo int64
	Notes       string
	Version     int64
	UserID      pulid.ID
}

// EndEnrollment closes the cover and stops the contribution with it. The
// deduction is ended rather than deleted, so a settlement that already took it
// still has something to point at.
func (s *Service) EndEnrollment(
	ctx context.Context,
	req *EndEnrollmentRequest,
) (*driverpay.WorkerBenefitEnrollment, error) {
	original, err := s.repo.GetEnrollmentByID(ctx, &repositories.GetBenefitEnrollmentByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if original.Status == driverpay.EnrollmentEnded {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalidOperation,
			"That cover has already ended",
		)
	}
	if req.Version > 0 && original.Version != req.Version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"Enrollment was changed by someone else. Reload and try again",
		)
	}

	previous := *original
	endsAt := req.EffectiveTo
	if endsAt <= 0 {
		endsAt = timeutils.NowUnix()
	}
	if endsAt < original.EffectiveFrom {
		return nil, errortypes.NewValidationError(
			"effectiveTo",
			errortypes.ErrInvalid,
			"Cover cannot end before it begins",
		)
	}

	original.Status = driverpay.EnrollmentEnded
	original.EffectiveTo = &endsAt
	if req.Notes != "" {
		original.Notes = req.Notes
	}
	original.Normalise()

	if err = s.closeContribution(ctx, req.TenantInfo, original, endsAt); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateEnrollment(ctx, original)
	if err != nil {
		return nil, err
	}

	s.audit(
		updated.GetResourceID(), permission.OpUpdate, req.UserID, req.TenantInfo,
		updated, &previous, "Benefit cover ended",
	)

	return updated, nil
}

func (s *Service) closeContribution(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	enrollment *driverpay.WorkerBenefitEnrollment,
	endsAt int64,
) error {
	if enrollment.RecurringDeductionID.IsNil() {
		return nil
	}

	deduction, err := s.deductionRepo.GetByID(
		ctx,
		repositories.GetRecurringDeductionByIDRequest{
			ID:         enrollment.RecurringDeductionID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		// The deduction may have been removed independently. The enrollment
		// still has to close: cover that has ended has ended.
		s.l.Warn("benefit deduction not found while ending cover",
			zap.String("enrollmentId", enrollment.ID.String()),
			zap.Error(err))
		return nil
	}

	deduction.Status = driverpay.DeductionStatusCompleted
	// The end date is set as well as the status so a settlement generated for
	// an earlier period still sees it as live for that period.
	deduction.EndDate = &endsAt
	if _, err = s.deductionRepo.Update(ctx, deduction); err != nil {
		return err
	}

	return nil
}

// TotalCompensation is what a job is actually worth over a period: what was
// paid, what the employer put in on top, and what the worker paid for cover.
type TotalCompensation struct {
	AsOf                   int64
	PlanYear               int16
	GrossPayMinor          int64
	EmployerBenefitMinor   int64
	EmployeeBenefitMinor   int64
	AccruedTimeOffDays     string
	TotalCompensationMinor int64
	Enrollments            []*driverpay.WorkerBenefitEnrollment
}

// TotalCompensationRequest asks what a worker's package is worth.
type TotalCompensationRequest struct {
	TenantInfo pagination.TenantInfo
	WorkerID   pulid.ID
	PlanYear   int16
	// GrossPayMinor is what payroll actually paid, supplied by the caller that
	// has it. The benefits service does not compute pay — it composes what
	// other areas already know, which is what stops two answers to "what did
	// this driver earn".
	GrossPayMinor      int64
	AccruedTimeOffDays string
}

// TotalCompensation composes the statement. Employer contributions are added
// because that is the point of the statement; the employee's own contributions
// are reported but never added, since money the worker paid is not money the
// job gave them.
func (s *Service) TotalCompensation(
	ctx context.Context,
	req *TotalCompensationRequest,
) (*TotalCompensation, error) {
	enrollments, err := s.repo.ListEnrollments(ctx, &repositories.ListBenefitEnrollmentsRequest{
		TenantInfo:  req.TenantInfo,
		WorkerID:    req.WorkerID,
		OpenOnly:    true,
		IncludePlan: true,
	})
	if err != nil {
		return nil, err
	}

	out := &TotalCompensation{
		AsOf:               timeutils.NowUnix(),
		PlanYear:           req.PlanYear,
		GrossPayMinor:      req.GrossPayMinor,
		AccruedTimeOffDays: req.AccruedTimeOffDays,
		Enrollments:        enrollments,
	}
	for _, enrollment := range enrollments {
		if enrollment == nil || !enrollment.Status.Deducts() {
			continue
		}
		out.EmployerBenefitMinor += enrollment.EmployerCostMinor
		out.EmployeeBenefitMinor += enrollment.EmployeeCostMinor
	}
	out.TotalCompensationMinor = out.GrossPayMinor + out.EmployerBenefitMinor

	return out, nil
}
