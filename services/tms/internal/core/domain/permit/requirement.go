package permit

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Requirement)(nil)
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
	bun.BaseModel             `bun:"table:permit_requirements,alias:pmr" json:"-"`
	pagination.CursorValueSet `bun:",embed"                              json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ShipmentID     pulid.ID `json:"shipmentId"     bun:"shipment_id,type:VARCHAR(100),notnull"`
	StateID        pulid.ID `json:"stateId"        bun:"state_id,type:VARCHAR(100),notnull"`

	Status        RequirementStatus `json:"status"        bun:"status,type:permit_requirement_status_enum,notnull,default:'Open'"`
	RouteSequence int16             `json:"routeSequence" bun:"route_sequence,type:SMALLINT,notnull,default:0"`

	Exceedances  []jurisdictionrule.Exceedance        `json:"exceedances"  bun:"exceedances,type:JSONB,nullzero"`
	Escorts      []jurisdictionrule.EscortRequirement `json:"escorts"      bun:"escorts,type:JSONB,nullzero"`
	Restrictions []jurisdictionrule.RestrictionKind   `json:"restrictions" bun:"restrictions,type:JSONB,nullzero"`

	LeadTimeDays int16               `json:"leadTimeDays" bun:"lead_time_days,type:SMALLINT,notnull,default:0"`
	ValidityDays int16               `json:"validityDays" bun:"validity_days,type:SMALLINT,notnull,default:0"`
	EstimatedFee decimal.NullDecimal `json:"estimatedFee" bun:"estimated_fee,type:NUMERIC(19,4),nullzero"`
	IsSuperload  bool                `json:"isSuperload"  bun:"is_superload,type:BOOLEAN,notnull,default:false"`

	SatisfiedByPermitID *pulid.ID `json:"satisfiedByPermitId" bun:"satisfied_by_permit_id,type:VARCHAR(100),nullzero"`
	WaivedByID          *pulid.ID `json:"waivedById"          bun:"waived_by_id,type:VARCHAR(100),nullzero"`
	WaiverReason        string    `json:"waiverReason"        bun:"waiver_reason,type:TEXT,nullzero"`

	Provenance *RuleProvenance `json:"provenance" bun:"provenance,type:JSONB,nullzero"`

	DerivedAt int64 `json:"derivedAt" bun:"derived_at,type:BIGINT,notnull"`
	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit      *tenant.BusinessUnit `json:"-"                           bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization      *tenant.Organization `json:"-"                           bun:"rel:belongs-to,join:organization_id=id"`
	State             *usstate.UsState     `json:"state,omitempty"             bun:"rel:belongs-to,join:state_id=id"`
	SatisfiedByPermit *Permit              `json:"satisfiedByPermit,omitempty" bun:"rel:belongs-to,join:satisfied_by_permit_id=id"`
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

func (r *Requirement) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	if r.Status == "" {
		r.Status = RequirementOpen
	}
	if r.DerivedAt == 0 {
		r.DerivedAt = now
	}

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("pmr_")
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}
