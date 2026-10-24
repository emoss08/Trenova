package permit

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/places/domain/jurisdictionrule"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*Requirement)(nil)
)

// RuleProvenance is a snapshot of the jurisdiction rule as it stood when the
// requirement was derived. It carries VerificationState deliberately: a
// requirement computed from unverified seed data must say so on its face rather
// than presenting itself with the same authority as a confirmed rule.
type RuleProvenance struct {
	RuleID            pulid.ID                           `json:"ruleId"`
	StateCode         string                             `json:"stateCode"`
	StateName         string                             `json:"stateName"`
	VerificationState jurisdictionrule.VerificationState `json:"verificationState"`
	VerifiedAt        *int64                             `json:"verifiedAt,omitempty"`
	SourceNote        string                             `json:"sourceNote,omitempty"`
	SourceURL         string                             `json:"sourceUrl,omitempty"`
	MaxWidthFeet      float64                            `json:"maxWidthFeet"`
	MaxHeightFeet     float64                            `json:"maxHeightFeet"`
	MaxLengthFeet     float64                            `json:"maxLengthFeet"`
	MaxWeightPounds   int64                              `json:"maxWeightPounds"`
	OverriddenFields  []string                           `json:"overriddenFields,omitempty"`
	OverrideReason    string                             `json:"overrideReason,omitempty"`
}

func (p *RuleProvenance) IsTrusted() bool {
	return p.VerificationState.IsTrusted()
}

type Requirement struct {
	keel.Table     `                                                   keel:"permit_requirements"`
	ID             pulid.ID `json:"id" keel:"id,pk,autoid:pmr_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	ShipmentID     pulid.ID `json:"shipmentId" keel:"shipment_id"`
	StateID        pulid.ID `json:"stateId" keel:"state_id"`

	Status        RequirementStatus `json:"status" keel:"status"`
	RouteSequence int16             `json:"routeSequence" keel:"route_sequence"`

	Exceedances  []jurisdictionrule.Exceedance        `json:"exceedances" keel:"exceedances,jsonb,nullzero"`
	Escorts      []jurisdictionrule.EscortRequirement `json:"escorts" keel:"escorts,jsonb,nullzero"`
	Restrictions []jurisdictionrule.RestrictionKind   `json:"restrictions" keel:"restrictions,jsonb,nullzero"`

	LeadTimeDays int16               `json:"leadTimeDays" keel:"lead_time_days"`
	ValidityDays int16               `json:"validityDays" keel:"validity_days"`
	EstimatedFee decimal.NullDecimal `json:"estimatedFee" keel:"estimated_fee"`
	IsSuperload  bool                `json:"isSuperload" keel:"is_superload"`

	SatisfiedByPermitID *pulid.ID `json:"satisfiedByPermitId" keel:"satisfied_by_permit_id"`
	WaivedByID          *pulid.ID `json:"waivedById" keel:"waived_by_id"`
	WaiverReason        string    `json:"waiverReason" keel:"waiver_reason,nullzero"`

	Provenance *RuleProvenance `json:"provenance" keel:"provenance,jsonb"`

	DerivedAt int64 `json:"derivedAt" keel:"derived_at"`
	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit      *tenant.BusinessUnit `json:"-"`
	Organization      *tenant.Organization `json:"-"`
	State             *usstate.UsState     `json:"state,omitempty" keel:"belongs:state_id"`
	SatisfiedByPermit *Permit              `json:"satisfiedByPermit,omitempty" keel:"belongs:satisfied_by_permit_id"`
}

func (r *Requirement) BeforeUpdate(_ context.Context) error {
	now := timeutils.NowUnix()
	if r.Status == "" {
		r.Status = RequirementOpen
	}
	if r.DerivedAt == 0 {
		r.DerivedAt = now
	}

	return nil
}

func (r *Requirement) BeforeCreate(_ context.Context) error {
	now := timeutils.NowUnix()
	if r.Status == "" {
		r.Status = RequirementOpen
	}
	if r.DerivedAt == 0 {
		r.DerivedAt = now
	}

	return nil
}

func (r *Requirement) GetID() pulid.ID { return r.ID }

func (r *Requirement) GetCreatedAt() int64 { return r.CreatedAt }

func (r *Requirement) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *Requirement) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *Requirement) GetTableName() string { return "permit_requirements" }

func (r *Requirement) StateCode() string {
	if r.Provenance != nil && r.Provenance.StateCode != "" {
		return r.Provenance.StateCode
	}
	if r.State != nil {
		return r.State.Abbreviation
	}
	return ""
}

func (r *Requirement) RequiresEscorts() bool {
	return len(r.Escorts) > 0
}

func (r *Requirement) EscortRoles() []jurisdictionrule.EscortRole {
	roles := make([]jurisdictionrule.EscortRole, 0, len(r.Escorts))
	for _, escort := range r.Escorts {
		roles = append(roles, escort.Role)
	}
	return roles
}

// Describe renders the requirement as an operator would read it: which state,
// which dimension pushed it over, and by how much.
func (r *Requirement) Describe() string {
	code := r.StateCode()

	reasons := make([]string, 0, len(r.Exceedances))
	for _, exceedance := range r.Exceedances {
		reasons = append(reasons, fmt.Sprintf(
			"%s %.2f%s over",
			strings.ToLower(exceedance.Trigger.String()),
			exceedance.OverBy,
			exceedance.Trigger.Unit(),
		))
	}

	if len(reasons) == 0 {
		return fmt.Sprintf("%s permit required", code)
	}

	return fmt.Sprintf("%s permit required (%s)", code, strings.Join(reasons, ", "))
}

func (r *Requirement) Validate(multiErr *errortypes.MultiError) {
	if r.ShipmentID.IsNil() {
		multiErr.Add("shipmentId", errortypes.ErrRequired, "Shipment is required")
	}
	if r.StateID.IsNil() {
		multiErr.Add("stateId", errortypes.ErrRequired, "State is required")
	}
	if !r.Status.IsValid() {
		multiErr.Add("status", errortypes.ErrInvalid, "Status is invalid")
	}

	if r.Status == RequirementSatisfied &&
		(r.SatisfiedByPermitID == nil || r.SatisfiedByPermitID.IsNil()) {
		multiErr.Add("satisfiedByPermitId", errortypes.ErrRequired,
			"A satisfied requirement must name the permit that satisfies it")
	}

	if r.Status == RequirementWaived && len(r.WaiverReason) < MinWaiverReasonLength {
		multiErr.Add("waiverReason", errortypes.ErrInvalid,
			"Provide a reason of at least 10 characters explaining the waiver")
	}
}

const MinWaiverReasonLength = 10

func (r *Requirement) BeforeSave(_ context.Context) error {
	now := timeutils.NowUnix()
	if r.Status == "" {
		r.Status = RequirementOpen
	}
	if r.DerivedAt == 0 {
		r.DerivedAt = now
	}

	return nil
}
