package benefitsservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/benefitsservice"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBenefitRepo struct {
	repositories.BenefitRepository
	plan        *driverpay.BenefitPlan
	enrollments []*driverpay.WorkerBenefitEnrollment
}

func (r *fakeBenefitRepo) GetPlanByID(
	_ context.Context,
	_ *repositories.GetBenefitPlanByIDRequest,
) (*driverpay.BenefitPlan, error) {
	return r.plan, nil
}

func (r *fakeBenefitRepo) CreateEnrollment(
	_ context.Context,
	entity *driverpay.WorkerBenefitEnrollment,
) (*driverpay.WorkerBenefitEnrollment, error) {
	entity.ID = pulid.MustNew("wben_")
	r.enrollments = append(r.enrollments, entity)
	return entity, nil
}

func (r *fakeBenefitRepo) GetEnrollmentByID(
	_ context.Context,
	req *repositories.GetBenefitEnrollmentByIDRequest,
) (*driverpay.WorkerBenefitEnrollment, error) {
	for _, row := range r.enrollments {
		if row.ID == req.ID {
			return row, nil
		}
	}
	return nil, assertNotFound{}
}

func (r *fakeBenefitRepo) UpdateEnrollment(
	_ context.Context,
	entity *driverpay.WorkerBenefitEnrollment,
) (*driverpay.WorkerBenefitEnrollment, error) {
	return entity, nil
}

func (r *fakeBenefitRepo) ListEnrollments(
	_ context.Context,
	_ *repositories.ListBenefitEnrollmentsRequest,
) ([]*driverpay.WorkerBenefitEnrollment, error) {
	return r.enrollments, nil
}

type assertNotFound struct{}

func (assertNotFound) Error() string { return "not found" }

type fakeDeductionRepo struct {
	repositories.RecurringDeductionRepository
	created []*driverpay.RecurringDeduction
	updated []*driverpay.RecurringDeduction
}

func (r *fakeDeductionRepo) Create(
	_ context.Context,
	entity *driverpay.RecurringDeduction,
) (*driverpay.RecurringDeduction, error) {
	entity.ID = pulid.MustNew("rded_")
	r.created = append(r.created, entity)
	return entity, nil
}

func (r *fakeDeductionRepo) GetByID(
	_ context.Context,
	req repositories.GetRecurringDeductionByIDRequest,
) (*driverpay.RecurringDeduction, error) {
	for _, row := range r.created {
		if row.ID == req.ID {
			return row, nil
		}
	}
	return nil, assertNotFound{}
}

func (r *fakeDeductionRepo) Update(
	_ context.Context,
	entity *driverpay.RecurringDeduction,
) (*driverpay.RecurringDeduction, error) {
	r.updated = append(r.updated, entity)
	return entity, nil
}

func benefitTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func newBenefitService(
	plan *driverpay.BenefitPlan,
) (*benefitsservice.Service, *fakeBenefitRepo, *fakeDeductionRepo) {
	repo := &fakeBenefitRepo{plan: plan}
	deductions := &fakeDeductionRepo{}
	svc := benefitsservice.NewWithDeps(benefitsservice.Deps{
		Repo:          repo,
		DeductionRepo: deductions,
	})
	return svc, repo, deductions
}

func testPlan() *driverpay.BenefitPlan {
	return &driverpay.BenefitPlan{
		ID:                "bplan_1",
		Status:            domaintypes.StatusActive,
		Code:              "MED-PPO",
		Name:              "Medical PPO",
		PlanType:          driverpay.BenefitPlanMedical,
		PayCodeID:         "pcod_1",
		PlanYear:          2026,
		EmployeeCostMinor: 12_000,
		EmployerCostMinor: 48_000,
		CurrencyCode:      "USD",
	}
}

// The whole point of the design: an employee contribution is taken through an
// ordinary recurring deduction, so it inherits the cap, the pause and the
// reversal on a voided settlement that the deduction path already has.
func TestEnroll_OpensAnOrdinaryDeduction(t *testing.T) {
	t.Parallel()

	svc, _, deductions := newBenefitService(testPlan())

	enrollment, err := svc.Enroll(t.Context(), &benefitsservice.EnrollRequest{
		TenantInfo:   benefitTenant(),
		WorkerID:     "wrk_1",
		PlanID:       "bplan_1",
		CoverageTier: driverpay.TierEmployee,
		UserID:       "usr_1",
	})
	require.NoError(t, err)

	require.Len(t, deductions.created, 1)
	deduction := deductions.created[0]
	assert.Equal(t, driverpay.DeductionKindBenefit, deduction.Kind)
	assert.Equal(t, int64(12_000), deduction.AmountMinor)
	assert.Equal(t, "pcod_1", deduction.PayCodeID.String())
	// A benefit is taken after a garnishment; the priority is what says so.
	assert.Equal(t, driverpay.DeductionKindBenefit.DefaultPriority(), deduction.Priority)
	assert.Equal(t, deduction.ID, enrollment.RecurringDeductionID)
	// No cap: a contribution runs for as long as the cover does, and the
	// enrollment ending is what stops it.
	assert.Nil(t, deduction.TotalCapMinor)
}

func TestEnroll_ScalesTheCostToTheTier(t *testing.T) {
	t.Parallel()

	svc, _, deductions := newBenefitService(testPlan())

	enrollment, err := svc.Enroll(t.Context(), &benefitsservice.EnrollRequest{
		TenantInfo:   benefitTenant(),
		WorkerID:     "wrk_1",
		PlanID:       "bplan_1",
		CoverageTier: driverpay.TierFamily,
		UserID:       "usr_1",
	})
	require.NoError(t, err)

	assert.Equal(t, int64(31_200), enrollment.EmployeeCostMinor)
	require.Len(t, deductions.created, 1)
	assert.Equal(t, int64(31_200), deductions.created[0].AmountMinor)
}

// A carrier that prices each tier separately overrides the arithmetic, and the
// override has to win.
func TestEnroll_HonoursAnExplicitPrice(t *testing.T) {
	t.Parallel()

	svc, _, deductions := newBenefitService(testPlan())

	enrollment, err := svc.Enroll(t.Context(), &benefitsservice.EnrollRequest{
		TenantInfo:        benefitTenant(),
		WorkerID:          "wrk_1",
		PlanID:            "bplan_1",
		CoverageTier:      driverpay.TierFamily,
		EmployeeCostMinor: 28_500,
		UserID:            "usr_1",
	})
	require.NoError(t, err)

	assert.Equal(t, int64(28_500), enrollment.EmployeeCostMinor)
	assert.Equal(t, int64(28_500), deductions.created[0].AmountMinor)
}

// A waiver is a record that somebody declined. It must never take money, and
// it must not carry the employer's cost either — that would show up in a total
// as cover nobody has.
func TestEnroll_WaiverTakesNothingAndCostsNothing(t *testing.T) {
	t.Parallel()

	svc, _, deductions := newBenefitService(testPlan())

	enrollment, err := svc.Enroll(t.Context(), &benefitsservice.EnrollRequest{
		TenantInfo:   benefitTenant(),
		WorkerID:     "wrk_1",
		PlanID:       "bplan_1",
		Waive:        true,
		WaivedReason: "Covered by a spouse's plan",
		UserID:       "usr_1",
	})
	require.NoError(t, err)

	assert.Equal(t, driverpay.EnrollmentWaived, enrollment.Status)
	assert.Equal(t, int64(0), enrollment.EmployeeCostMinor)
	assert.Equal(t, int64(0), enrollment.EmployerCostMinor)
	assert.True(t, enrollment.RecurringDeductionID.IsNil())
	assert.Empty(t, deductions.created, "a waiver opens no deduction")
}

func TestEnroll_RefusesAnArchivedPlan(t *testing.T) {
	t.Parallel()

	plan := testPlan()
	plan.Status = domaintypes.StatusInactive
	svc, _, _ := newBenefitService(plan)

	_, err := svc.Enroll(t.Context(), &benefitsservice.EnrollRequest{
		TenantInfo: benefitTenant(),
		WorkerID:   "wrk_1",
		PlanID:     "bplan_1",
		UserID:     "usr_1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "archived")
}

// Ending cover has to stop the money with it, or a settlement keeps taking a
// contribution for a plan the worker is no longer on.
func TestEndEnrollment_StopsTheContribution(t *testing.T) {
	t.Parallel()

	svc, _, deductions := newBenefitService(testPlan())
	tenant := benefitTenant()

	enrollment, err := svc.Enroll(t.Context(), &benefitsservice.EnrollRequest{
		TenantInfo: tenant,
		WorkerID:   "wrk_1",
		PlanID:     "bplan_1",
		UserID:     "usr_1",
	})
	require.NoError(t, err)

	endsAt := enrollment.EffectiveFrom + 30*86400
	ended, err := svc.EndEnrollment(t.Context(), &benefitsservice.EndEnrollmentRequest{
		TenantInfo:  tenant,
		ID:          enrollment.ID,
		EffectiveTo: endsAt,
		UserID:      "usr_1",
	})
	require.NoError(t, err)

	assert.Equal(t, driverpay.EnrollmentEnded, ended.Status)
	require.NotNil(t, ended.EffectiveTo)
	assert.Equal(t, endsAt, *ended.EffectiveTo)

	require.Len(t, deductions.updated, 1)
	assert.Equal(t, driverpay.DeductionStatusCompleted, deductions.updated[0].Status)
	// The end date is set as well as the status, so a settlement generated for
	// an earlier period still sees it as live for that period.
	require.NotNil(t, deductions.updated[0].EndDate)
	assert.Equal(t, endsAt, *deductions.updated[0].EndDate)
}

func TestEndEnrollment_RefusesToEndBeforeItBegan(t *testing.T) {
	t.Parallel()

	svc, _, _ := newBenefitService(testPlan())
	tenant := benefitTenant()

	enrollment, err := svc.Enroll(t.Context(), &benefitsservice.EnrollRequest{
		TenantInfo: tenant,
		WorkerID:   "wrk_1",
		PlanID:     "bplan_1",
		UserID:     "usr_1",
	})
	require.NoError(t, err)

	_, err = svc.EndEnrollment(t.Context(), &benefitsservice.EndEnrollmentRequest{
		TenantInfo:  tenant,
		ID:          enrollment.ID,
		EffectiveTo: enrollment.EffectiveFrom - 1,
		UserID:      "usr_1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot end before it begins")
}

// Money the worker paid is not money the job gave them, so the employee's own
// contributions are reported but never added to the total.
func TestTotalCompensation_AddsTheEmployerShareOnly(t *testing.T) {
	t.Parallel()

	svc, _, _ := newBenefitService(testPlan())
	tenant := benefitTenant()

	_, err := svc.Enroll(t.Context(), &benefitsservice.EnrollRequest{
		TenantInfo: tenant,
		WorkerID:   "wrk_1",
		PlanID:     "bplan_1",
		UserID:     "usr_1",
	})
	require.NoError(t, err)

	total, err := svc.TotalCompensation(t.Context(), &benefitsservice.TotalCompensationRequest{
		TenantInfo:    tenant,
		WorkerID:      "wrk_1",
		PlanYear:      2026,
		GrossPayMinor: 9_000_000,
	})
	require.NoError(t, err)

	assert.Equal(t, int64(48_000), total.EmployerBenefitMinor)
	assert.Equal(t, int64(12_000), total.EmployeeBenefitMinor)
	assert.Equal(t, int64(9_048_000), total.TotalCompensationMinor)
}

// A waived plan contributes nothing on either side of the statement.
func TestTotalCompensation_IgnoresAWaiver(t *testing.T) {
	t.Parallel()

	svc, _, _ := newBenefitService(testPlan())
	tenant := benefitTenant()

	_, err := svc.Enroll(t.Context(), &benefitsservice.EnrollRequest{
		TenantInfo:   tenant,
		WorkerID:     "wrk_1",
		PlanID:       "bplan_1",
		Waive:        true,
		WaivedReason: "Covered elsewhere",
		UserID:       "usr_1",
	})
	require.NoError(t, err)

	total, err := svc.TotalCompensation(t.Context(), &benefitsservice.TotalCompensationRequest{
		TenantInfo:    tenant,
		WorkerID:      "wrk_1",
		GrossPayMinor: 9_000_000,
	})
	require.NoError(t, err)

	assert.Equal(t, int64(0), total.EmployerBenefitMinor)
	assert.Equal(t, int64(9_000_000), total.TotalCompensationMinor)
}
