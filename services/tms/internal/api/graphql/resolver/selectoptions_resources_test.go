package resolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/generated"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every value the schema offers has to reach a resolver: an enum value with no
// registry entry is accepted by the schema and then refused at runtime with a
// validation error, which reads to the client as a broken picker.
func TestSelectOptionRegistry_CoversEverySchemaResource(t *testing.T) {
	t.Parallel()

	schema := generated.NewExecutableSchema(generated.Config{}).Schema()
	definition, ok := schema.Types["SelectOptionResource"]
	require.True(t, ok)
	require.NotEmpty(t, definition.EnumValues)

	registry := (&Resolver{}).selectOptionRegistry()
	for _, value := range definition.EnumValues {
		t.Run(value.Name, func(t *testing.T) {
			entry, registered := registry[gqlmodel.SelectOptionResource(value.Name)]
			require.Truef(t, registered, "SelectOptionResource %q has no registry entry", value.Name)
			require.NotNilf(t, entry.resolve, "SelectOptionResource %q has a nil resolver", value.Name)
		})
	}
}

func TestPayCodeSelectOptionItem(t *testing.T) {
	t.Parallel()

	entity := &driverpay.PayCode{
		ID:                    pulid.MustNew("payc_"),
		Code:                  "LH",
		Name:                  "Linehaul",
		Direction:             driverpay.PayCodeDirectionEarning,
		Taxable:               true,
		CountsTowardGuarantee: true,
		CreatedAt:             1780415883,
	}

	item := payCodeSelectOptionItem(entity)

	assert.Equal(t, "LH", item.option.Label)
	require.NotNil(t, item.option.Description)
	assert.Equal(t, "Linehaul", *item.option.Description)
	assert.Equal(t, "Linehaul", item.option.Meta["name"])
	assert.Equal(t, string(driverpay.PayCodeDirectionEarning), item.option.Meta["direction"])
	assert.Equal(t, true, item.option.Meta["taxable"])
	assert.Equal(t, int64(1780415883), item.cursor.CreatedAt)
}

func TestPayProfileSelectOptionItem(t *testing.T) {
	t.Parallel()

	item := payProfileSelectOptionItem(&driverpay.PayProfile{
		ID:             pulid.MustNew("dpp_"),
		Name:           "Owner Operator 2026",
		Description:    "Percentage of linehaul",
		Classification: driverpay.PayeeClassificationOwnerOperator,
		CurrencyCode:   "USD",
		CreatedAt:      1780415884,
	})

	assert.Equal(t, "Owner Operator 2026", item.option.Label)
	assert.Equal(
		t,
		string(driverpay.PayeeClassificationOwnerOperator),
		item.option.Meta["classification"],
	)
	assert.Equal(t, "USD", item.option.Meta["currencyCode"])
}

func TestWorkerCredentialTypeSelectOptionItem_CarriesFormDrivingMeta(t *testing.T) {
	t.Parallel()

	validityMonths := int32(48)
	item := workerCredentialTypeSelectOptionItem(&worker.WorkerCredentialType{
		ID:               pulid.MustNew("wct_"),
		Code:             "CDL",
		Name:             "Commercial Driver License",
		Description:      "State issued CDL",
		RequiresNumber:   true,
		RequiresDocument: true,
		ProfileField:     worker.CredentialProfileFieldLicenseExpiry,
		ValidityMonths:   &validityMonths,
		CreatedAt:        1780415885,
	})

	assert.Equal(t, "Commercial Driver License", item.option.Label)
	assert.Equal(t, true, item.option.Meta["requiresNumber"])
	assert.Equal(t, true, item.option.Meta["requiresDocument"])
	assert.Equal(
		t,
		string(worker.CredentialProfileFieldLicenseExpiry),
		item.option.Meta["profileField"],
	)
	assert.Equal(t, int32(48), item.option.Meta["validityMonths"])
}

// A credential type with no expiry has to send a null, not a zero: the form
// reads the value back to suggest an expiry date, and 0 months would suggest
// the day it was issued.
func TestWorkerCredentialTypeSelectOptionItem_NilValidityStaysNull(t *testing.T) {
	t.Parallel()

	item := workerCredentialTypeSelectOptionItem(&worker.WorkerCredentialType{
		ID:   pulid.MustNew("wct_"),
		Code: "TWIC",
		Name: "TWIC Card",
	})

	require.Contains(t, item.option.Meta, "validityMonths")
	assert.Nil(t, item.option.Meta["validityMonths"])
}

func TestTrainingCourseSelectOptionItem_ScoredCourse(t *testing.T) {
	t.Parallel()

	item := trainingCourseSelectOptionItem(&worker.TrainingCourse{
		ID:                     pulid.MustNew("trnc_"),
		Code:                   "HAZMAT",
		Name:                   "Hazmat Refresher",
		Description:            "Annual hazmat handling",
		DurationMinutes:        90,
		DueDaysAfterAssignment: 14,
		PassingScore:           decimal.NewNullDecimal(decimal.RequireFromString("80.00")),
		CreatedAt:              1780415886,
	})

	assert.Equal(t, "Hazmat Refresher", item.option.Label)
	assert.Equal(t, int32(14), item.option.Meta["dueDaysAfterAssignment"])
	assert.Equal(t, "80", item.option.Meta["passingScore"])
}

// An unscored course must report a null passing score; the completion dialog
// decides whether to ask for a score from exactly this value.
func TestTrainingCourseSelectOptionItem_UnscoredCourseSendsNull(t *testing.T) {
	t.Parallel()

	item := trainingCourseSelectOptionItem(&worker.TrainingCourse{
		ID:   pulid.MustNew("trnc_"),
		Code: "ORIENT",
		Name: "Orientation",
	})

	require.Contains(t, item.option.Meta, "passingScore")
	assert.Nil(t, item.option.Meta["passingScore"])
}

func TestPerformanceReviewTemplateSelectOptionItem(t *testing.T) {
	t.Parallel()

	cadence := int32(12)
	item := performanceReviewTemplateSelectOptionItem(&worker.PerformanceReviewTemplate{
		ID:            pulid.MustNew("prt_"),
		Code:          "ANNUAL",
		Name:          "Annual Driver Review",
		IsDefault:     true,
		CadenceMonths: &cadence,
		Items:         []worker.ReviewItem{{Label: "Safety"}, {Label: "Service"}},
		CreatedAt:     1780415887,
	})

	assert.Equal(t, "Annual Driver Review", item.option.Label)
	assert.Equal(t, 2, item.option.Meta["itemCount"])
	assert.Equal(t, int32(12), item.option.Meta["cadenceMonths"])
	assert.Equal(t, true, item.option.Meta["isDefault"])
}

func TestPTOPolicySelectOptionItem(t *testing.T) {
	t.Parallel()

	item := ptoPolicySelectOptionItem(&worker.PTOPolicy{
		ID:               pulid.MustNew("ptop_"),
		Code:             "STD",
		Name:             "Standard PTO",
		Description:      "Accrues per period",
		IsDefault:        true,
		YearBasis:        worker.PTOYearBasisCalendarYear,
		RequiresApproval: true,
		CreatedAt:        1780415888,
	})

	assert.Equal(t, "Standard PTO", item.option.Label)
	assert.Equal(t, string(worker.PTOYearBasisCalendarYear), item.option.Meta["yearBasis"])
	assert.Equal(t, true, item.option.Meta["requiresApproval"])
}

func TestBenefitPlanSelectOptionItem(t *testing.T) {
	t.Parallel()

	item := benefitPlanSelectOptionItem(&driverpay.BenefitPlan{
		ID:                pulid.MustNew("bplan_"),
		Code:              "MED-HD",
		Name:              "Medical High Deductible",
		PlanType:          driverpay.BenefitPlanMedical,
		Carrier:           "Blue Cross",
		PlanYear:          2026,
		EmployeeCostMinor: 12500,
		EmployerCostMinor: 40000,
		CurrencyCode:      "USD",
		WaitingPeriodDays: 30,
		CreatedAt:         1780415889,
	})

	assert.Equal(t, "Medical High Deductible", item.option.Label)
	assert.Equal(t, string(driverpay.BenefitPlanMedical), item.option.Meta["planType"])
	assert.Equal(t, int16(2026), item.option.Meta["planYear"])
	assert.Equal(t, int64(12500), item.option.Meta["employeeCostMinor"])
}

func TestSelectOptionInt16Filter(t *testing.T) {
	t.Parallel()

	// GraphQL JSON scalars decode numbers as float64, so that is the shape the
	// planYear filter actually arrives in.
	assert.Equal(t, int16(2026), selectOptionInt16Filter(map[string]any{"planYear": 2026.0}, "planYear"))
	assert.Equal(t, int16(2026), selectOptionInt16Filter(map[string]any{"planYear": "2026"}, "planYear"))
	assert.Equal(t, int16(0), selectOptionInt16Filter(map[string]any{"planYear": true}, "planYear"))
	assert.Equal(t, int16(0), selectOptionInt16Filter(map[string]any{}, "planYear"))
}
