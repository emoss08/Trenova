package driverpay

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	ErrInvalidBenefitPlanType  = errors.New("invalid benefit plan type")
	ErrInvalidEnrollmentStatus = errors.New("invalid benefit enrollment status")
	ErrInvalidCoverageTier     = errors.New("invalid coverage tier")
)

type BenefitPlanType string

const (
	BenefitPlanMedical    = BenefitPlanType("Medical")
	BenefitPlanDental     = BenefitPlanType("Dental")
	BenefitPlanVision     = BenefitPlanType("Vision")
	BenefitPlanLife       = BenefitPlanType("Life")
	BenefitPlanDisability = BenefitPlanType("Disability")
	BenefitPlanRetirement = BenefitPlanType("Retirement")
	BenefitPlanOther      = BenefitPlanType("Other")
)

func (t BenefitPlanType) String() string { return string(t) }

func (t BenefitPlanType) IsValid() bool {
	switch t {
	case BenefitPlanMedical, BenefitPlanDental, BenefitPlanVision, BenefitPlanLife,
		BenefitPlanDisability, BenefitPlanRetirement, BenefitPlanOther:
		return true
	default:
		return false
	}
}

func AllBenefitPlanTypes() []BenefitPlanType {
	return []BenefitPlanType{
		BenefitPlanMedical,
		BenefitPlanDental,
		BenefitPlanVision,
		BenefitPlanLife,
		BenefitPlanDisability,
		BenefitPlanRetirement,
		BenefitPlanOther,
	}
}

type BenefitEnrollmentStatus string

const (
	EnrollmentPending = BenefitEnrollmentStatus("Pending")
	EnrollmentActive  = BenefitEnrollmentStatus("Active")
	EnrollmentWaived  = BenefitEnrollmentStatus("Waived")
	EnrollmentEnded   = BenefitEnrollmentStatus("Ended")
)

func (s BenefitEnrollmentStatus) String() string { return string(s) }

func (s BenefitEnrollmentStatus) IsValid() bool {
	switch s {
	case EnrollmentPending, EnrollmentActive, EnrollmentWaived, EnrollmentEnded:
		return true
	default:
		return false
	}
}

// Deducts reports whether an enrollment in this state should be taking money.
// A waiver and an ended enrollment are both records of coverage that is not
// happening, and neither should reach a settlement.
func (s BenefitEnrollmentStatus) Deducts() bool { return s == EnrollmentActive }

// CoverageTier is who the cover extends to. It is on the enrollment rather
// than the plan because the same plan costs different amounts depending on how
// many people it covers.
type CoverageTier string

const (
	TierEmployee         = CoverageTier("Employee")
	TierEmployeeSpouse   = CoverageTier("EmployeeSpouse")
	TierEmployeeChildren = CoverageTier("EmployeeChildren")
	TierFamily           = CoverageTier("Family")
)

func (t CoverageTier) String() string { return string(t) }

func (t CoverageTier) IsValid() bool {
	switch t {
	case TierEmployee, TierEmployeeSpouse, TierEmployeeChildren, TierFamily:
		return true
	default:
		return false
	}
}

// Multiplier is what the plan's employee-only cost is scaled by for a wider
// tier. A carrier that prices each tier separately overrides the amount on the
// enrollment; this is the starting point, not the last word.
func (t CoverageTier) Multiplier() float64 {
	switch t {
	case TierEmployeeSpouse:
		return 1.8
	case TierEmployeeChildren:
		return 1.6
	case TierFamily:
		return 2.6
	default:
		return 1
	}
}

func AllCoverageTiers() []CoverageTier {
	return []CoverageTier{TierEmployee, TierEmployeeSpouse, TierEmployeeChildren, TierFamily}
}

var (
	_ bun.BeforeAppendModelHook          = (*BenefitPlan)(nil)
	_ validationframework.TenantedEntity = (*BenefitPlan)(nil)
)

// BenefitPlan is what a carrier offers, for one plan year.
type BenefitPlan struct {
	bun.BaseModel `bun:"table:benefit_plans,alias:bplan" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Status       domaintypes.Status `json:"status"      bun:"status,type:status_enum,notnull,default:'Active'"`
	Code         string             `json:"code"        bun:"code,type:VARCHAR(20),notnull"`
	Name         string             `json:"name"        bun:"name,type:VARCHAR(100),notnull"`
	Description  string             `json:"description" bun:"description,type:TEXT,nullzero"`
	PlanType     BenefitPlanType    `json:"planType"    bun:"plan_type,type:benefit_plan_type_enum,notnull,default:'Medical'"`
	Carrier      string             `json:"carrier"      bun:"carrier,type:VARCHAR(150),nullzero"`
	PolicyNumber string             `json:"policyNumber" bun:"policy_number,type:VARCHAR(100),nullzero"`
	// PayCodeID is what a contribution shows up as on a settlement. Required
	// rather than optional: a deduction nobody can categorise is a deduction
	// nobody can explain.
	PayCodeID pulid.ID `json:"payCodeId" bun:"pay_code_id,type:VARCHAR(100),notnull"`
	PlanYear  int16    `json:"planYear"  bun:"plan_year,type:SMALLINT,notnull"`

	EmployeeCostMinor int64 `json:"employeeCostMinor" bun:"employee_cost_minor,type:BIGINT,notnull"`
	// EmployerCostMinor is never deducted. It is carried so a
	// total-compensation statement can show what the job is actually worth.
	EmployerCostMinor int64  `json:"employerCostMinor" bun:"employer_cost_minor,type:BIGINT,notnull"`
	CurrencyCode      string `json:"currencyCode"      bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	WaitingPeriodDays int32  `json:"waitingPeriodDays" bun:"waiting_period_days,type:INTEGER,notnull"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	PayCode *PayCode `json:"payCode,omitempty" bun:"rel:belongs-to,join:pay_code_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (p *BenefitPlan) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 20).Error("Code cannot exceed 20 characters"),
		),
		validation.Field(&p.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name cannot exceed 100 characters"),
		),
		validation.Field(&p.PlanType,
			validation.Required.Error("Plan type is required"),
			domainvalidation.ValidEnum[BenefitPlanType]("Plan type is not valid"),
		),
		validation.Field(&p.PayCodeID,
			validation.Required.Error("A pay code is required so a contribution can be categorised"),
		),
		validation.Field(&p.PlanYear,
			validation.Required.Error("Plan year is required"),
			validation.Min(int16(2000)).Error("Plan year is not valid"),
			validation.Max(int16(2200)).Error("Plan year is not valid"),
		),
		validation.Field(&p.WaitingPeriodDays,
			validation.Min(int32(0)).Error("Waiting period cannot be negative"),
			validation.Max(int32(365)).Error("Waiting period cannot exceed a year"),
		),
	))

	if p.EmployeeCostMinor < 0 {
		multiErr.Add("employeeCostMinor", errortypes.ErrInvalid, "Cost cannot be negative")
	}
	if p.EmployerCostMinor < 0 {
		multiErr.Add("employerCostMinor", errortypes.ErrInvalid, "Cost cannot be negative")
	}
	// A plan that costs nobody anything is a plan nobody is really on, and it
	// would produce a zero deduction on every settlement forever.
	if p.EmployeeCostMinor == 0 && p.EmployerCostMinor == 0 {
		multiErr.Add(
			"employeeCostMinor",
			errortypes.ErrInvalid,
			"A plan has to cost somebody something",
		)
	}
}

// CostForTier scales the employee-only price for a wider tier, rounding to the
// minor unit. It is the starting point a carrier can override on the
// enrollment, not the last word.
func (p *BenefitPlan) CostForTier(tier CoverageTier) int64 {
	if p.EmployeeCostMinor <= 0 {
		return 0
	}
	return int64(float64(p.EmployeeCostMinor)*tier.Multiplier() + 0.5)
}

func (p *BenefitPlan) Normalise() {
	p.Code = strings.ToUpper(strings.TrimSpace(p.Code))
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
	p.Carrier = strings.TrimSpace(p.Carrier)
	p.PolicyNumber = strings.TrimSpace(p.PolicyNumber)
}

func (p *BenefitPlan) GetID() pulid.ID { return p.ID }

func (p *BenefitPlan) GetCreatedAt() int64 { return p.CreatedAt }

func (p *BenefitPlan) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *BenefitPlan) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *BenefitPlan) GetTableName() string { return "benefit_plans" }

func (p *BenefitPlan) GetResourceType() string { return "benefit_plan" }

func (p *BenefitPlan) GetResourceID() string { return p.ID.String() }

func (p *BenefitPlan) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("bplan_")
		}
		if p.Status == "" {
			p.Status = domaintypes.StatusActive
		}
		if p.PlanType == "" {
			p.PlanType = BenefitPlanMedical
		}
		if p.CurrencyCode == "" {
			p.CurrencyCode = "USD"
		}
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}

var (
	_ bun.BeforeAppendModelHook          = (*WorkerBenefitEnrollment)(nil)
	_ validationframework.TenantedEntity = (*WorkerBenefitEnrollment)(nil)
)

// WorkerBenefitEnrollment is who is on which plan. The employee contribution
// is taken through a recurring deduction rather than a second mechanism, so it
// inherits the cap, the pause and the reversal on a voided settlement.
type WorkerBenefitEnrollment struct {
	bun.BaseModel `bun:"table:worker_benefit_enrollments,alias:wben" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`
	BenefitPlanID  pulid.ID `json:"benefitPlanId"  bun:"benefit_plan_id,type:VARCHAR(100),notnull"`

	Status       BenefitEnrollmentStatus `json:"status"       bun:"status,type:benefit_enrollment_status_enum,notnull,default:'Pending'"`
	CoverageTier CoverageTier            `json:"coverageTier" bun:"coverage_tier,type:benefit_coverage_tier_enum,notnull,default:'Employee'"`

	EffectiveFrom int64  `json:"effectiveFrom" bun:"effective_from,type:BIGINT,notnull"`
	EffectiveTo   *int64 `json:"effectiveTo"   bun:"effective_to,type:BIGINT,nullzero"`

	// EmployeeCostMinor and EmployerCostMinor are copied from the plan and the
	// tier when the enrollment is made, rather than read through. A plan
	// repriced next year must not silently restate what somebody was charged
	// this year.
	EmployeeCostMinor int64 `json:"employeeCostMinor" bun:"employee_cost_minor,type:BIGINT,notnull"`
	EmployerCostMinor int64 `json:"employerCostMinor" bun:"employer_cost_minor,type:BIGINT,notnull"`

	RecurringDeductionID pulid.ID `json:"recurringDeductionId" bun:"recurring_deduction_id,type:VARCHAR(100),nullzero"`
	WaivedReason         string   `json:"waivedReason"         bun:"waived_reason,type:VARCHAR(255),nullzero"`
	Notes                string   `json:"notes"                bun:"notes,type:TEXT,nullzero"`
	EnrolledByID         pulid.ID `json:"enrolledById"         bun:"enrolled_by_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker      *worker.Worker      `json:"worker,omitempty"      bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	BenefitPlan *BenefitPlan        `json:"benefitPlan,omitempty" bun:"rel:belongs-to,join:benefit_plan_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Deduction   *RecurringDeduction `json:"deduction,omitempty"   bun:"rel:belongs-to,join:recurring_deduction_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (e *WorkerBenefitEnrollment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&e.BenefitPlanID, validation.Required.Error("Plan is required")),
		validation.Field(&e.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[BenefitEnrollmentStatus]("Status is not valid"),
		),
		validation.Field(&e.CoverageTier,
			validation.Required.Error("Coverage is required"),
			domainvalidation.ValidEnum[CoverageTier]("Coverage is not valid"),
		),
		validation.Field(&e.EffectiveFrom,
			validation.Required.Error("An effective date is required"),
		),
		validation.Field(&e.WaivedReason,
			validation.Length(0, 255).Error("Reason cannot exceed 255 characters"),
		),
	))

	if e.EffectiveTo != nil && *e.EffectiveTo < e.EffectiveFrom {
		multiErr.Add(
			"effectiveTo",
			errortypes.ErrInvalid,
			"Coverage cannot end before it begins",
		)
	}
	// A waiver is a decision, and a decision with no reason on it is
	// indistinguishable later from nobody having asked.
	if e.Status == EnrollmentWaived && strings.TrimSpace(e.WaivedReason) == "" {
		multiErr.Add("waivedReason", errortypes.ErrRequired, "Say why the cover was declined")
	}
	if e.EmployeeCostMinor < 0 || e.EmployerCostMinor < 0 {
		multiErr.Add("employeeCostMinor", errortypes.ErrInvalid, "Cost cannot be negative")
	}
}

// IsOpen reports whether the enrollment is still running at an instant.
func (e *WorkerBenefitEnrollment) IsOpen(now int64) bool {
	if e.Status == EnrollmentEnded {
		return false
	}
	if now < e.EffectiveFrom {
		return false
	}
	return e.EffectiveTo == nil || *e.EffectiveTo >= now
}

func (e *WorkerBenefitEnrollment) Normalise() {
	e.WaivedReason = strings.TrimSpace(e.WaivedReason)
	e.Notes = strings.TrimSpace(e.Notes)
}

func (e *WorkerBenefitEnrollment) GetID() pulid.ID { return e.ID }

func (e *WorkerBenefitEnrollment) GetCreatedAt() int64 { return e.CreatedAt }

func (e *WorkerBenefitEnrollment) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *WorkerBenefitEnrollment) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *WorkerBenefitEnrollment) GetTableName() string { return "worker_benefit_enrollments" }

func (e *WorkerBenefitEnrollment) GetResourceType() string { return "worker_benefit_enrollment" }

func (e *WorkerBenefitEnrollment) GetResourceID() string { return e.ID.String() }

func (e *WorkerBenefitEnrollment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("wben_")
		}
		if e.Status == "" {
			e.Status = EnrollmentPending
		}
		if e.CoverageTier == "" {
			e.CoverageTier = TierEmployee
		}
		e.CreatedAt = now
		e.UpdatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}

	return nil
}
