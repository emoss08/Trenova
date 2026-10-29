package workforceselectoptions

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/paybenefit"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerCredentialTypeSelectOptionItem_CarriesFormDrivingMeta(t *testing.T) {
	t.Parallel()

	validityMonths := int32(48)
	item := workerCredentialTypeSelectOptionItem(&workercredential.WorkerCredentialType{
		ID:               pulid.MustNew("wct_"),
		Code:             "CDL",
		Name:             "Commercial Driver License",
		Description:      "State issued CDL",
		RequiresNumber:   true,
		RequiresDocument: true,
		ProfileField:     workercredential.CredentialProfileFieldLicenseExpiry,
		ValidityMonths:   &validityMonths,
		CreatedAt:        1780415885,
	})

	assert.Equal(t, "Commercial Driver License", item.Option.Label)
	assert.Equal(t, true, item.Option.Meta["requiresNumber"])
	assert.Equal(t, true, item.Option.Meta["requiresDocument"])
	assert.Equal(
		t,
		string(workercredential.CredentialProfileFieldLicenseExpiry),
		item.Option.Meta["profileField"],
	)
	assert.Equal(t, int32(48), item.Option.Meta["validityMonths"])
}

// A credential type with no expiry has to send a null, not a zero: the form
// reads the value back to suggest an expiry date, and 0 months would suggest
// the day it was issued.
func TestWorkerCredentialTypeSelectOptionItem_NilValidityStaysNull(t *testing.T) {
	t.Parallel()

	item := workerCredentialTypeSelectOptionItem(&workercredential.WorkerCredentialType{
		ID:   pulid.MustNew("wct_"),
		Code: "TWIC",
		Name: "TWIC Card",
	})

	require.Contains(t, item.Option.Meta, "validityMonths")
	assert.Nil(t, item.Option.Meta["validityMonths"])
}

func TestTrainingCourseSelectOptionItem_ScoredCourse(t *testing.T) {
	t.Parallel()

	item := trainingCourseSelectOptionItem(&workertraining.TrainingCourse{
		ID:                     pulid.MustNew("trnc_"),
		Code:                   "HAZMAT",
		Name:                   "Hazmat Refresher",
		Description:            "Annual hazmat handling",
		DurationMinutes:        90,
		DueDaysAfterAssignment: 14,
		PassingScore:           decimal.NewNullDecimal(decimal.RequireFromString("80.00")),
		CreatedAt:              1780415886,
	})

	assert.Equal(t, "Hazmat Refresher", item.Option.Label)
	assert.Equal(t, int32(14), item.Option.Meta["dueDaysAfterAssignment"])
	assert.Equal(t, "80", item.Option.Meta["passingScore"])
}

// An unscored course must report a null passing score; the completion dialog
// decides whether to ask for a score from exactly this value.
func TestTrainingCourseSelectOptionItem_UnscoredCourseSendsNull(t *testing.T) {
	t.Parallel()

	item := trainingCourseSelectOptionItem(&workertraining.TrainingCourse{
		ID:   pulid.MustNew("trnc_"),
		Code: "ORIENT",
		Name: "Orientation",
	})

	require.Contains(t, item.Option.Meta, "passingScore")
	assert.Nil(t, item.Option.Meta["passingScore"])
}

func TestPerformanceReviewTemplateSelectOptionItem(t *testing.T) {
	t.Parallel()

	cadence := int32(12)
	item := performanceReviewTemplateSelectOptionItem(&workerreview.PerformanceReviewTemplate{
		ID:            pulid.MustNew("prt_"),
		Code:          "ANNUAL",
		Name:          "Annual Driver Review",
		IsDefault:     true,
		CadenceMonths: &cadence,
		Items:         []workerreview.ReviewItem{{Label: "Safety"}, {Label: "Service"}},
		CreatedAt:     1780415887,
	})

	assert.Equal(t, "Annual Driver Review", item.Option.Label)
	assert.Equal(t, 2, item.Option.Meta["itemCount"])
	assert.Equal(t, int32(12), item.Option.Meta["cadenceMonths"])
	assert.Equal(t, true, item.Option.Meta["isDefault"])
}

func TestPTOPolicySelectOptionItem(t *testing.T) {
	t.Parallel()

	item := ptoPolicySelectOptionItem(&pto.PTOPolicy{
		ID:               pulid.MustNew("ptop_"),
		Code:             "STD",
		Name:             "Standard PTO",
		Description:      "Accrues per period",
		IsDefault:        true,
		YearBasis:        pto.PTOYearBasisCalendarYear,
		RequiresApproval: true,
		CreatedAt:        1780415888,
	})

	assert.Equal(t, "Standard PTO", item.Option.Label)
	assert.Equal(t, string(pto.PTOYearBasisCalendarYear), item.Option.Meta["yearBasis"])
	assert.Equal(t, true, item.Option.Meta["requiresApproval"])
}

func TestBenefitPlanSelectOptionItem(t *testing.T) {
	t.Parallel()

	item := benefitPlanSelectOptionItem(&paybenefit.BenefitPlan{
		ID:                pulid.MustNew("bplan_"),
		Code:              "MED-HD",
		Name:              "Medical High Deductible",
		PlanType:          paybenefit.BenefitPlanMedical,
		Carrier:           "Blue Cross",
		PlanYear:          2026,
		EmployeeCostMinor: 12500,
		EmployerCostMinor: 40000,
		CurrencyCode:      "USD",
		WaitingPeriodDays: 30,
		CreatedAt:         1780415889,
	})

	assert.Equal(t, "Medical High Deductible", item.Option.Label)
	assert.Equal(t, string(paybenefit.BenefitPlanMedical), item.Option.Meta["planType"])
	assert.Equal(t, int16(2026), item.Option.Meta["planYear"])
	assert.Equal(t, int64(12500), item.Option.Meta["employeeCostMinor"])
}
