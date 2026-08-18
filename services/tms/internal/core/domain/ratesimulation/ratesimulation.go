// Package ratesimulation replays a proposed pricing change against shipments
// that already happened.
//
// The question it answers is the one that stops a general rate increase from
// shipping: what would this contract have charged for the freight we actually
// moved. Every shipment is re-rated against its own historical facts — the
// weight it had, the lane it ran, the day it shipped — so the answer is what
// would have been invoiced rather than what today's traffic would produce.
//
// Nothing a simulation writes ever touches a shipment. The quotes it produces
// carry Purpose=Simulation and are never applied.
package ratesimulation

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratesimulation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*RateSimulation)(nil)
	_ validationframework.TenantedEntity = (*RateSimulation)(nil)
)

const (
	maxNameLength        = 150
	maxDescriptionLength = 500
	// maxShipmentSample bounds one run. A year of a mid-sized carrier's freight
	// is well under this; anything above it is somebody asking for the whole
	// history, which is a report rather than a simulation.
	maxShipmentSample = 100_000
)

// RateSimulation is one replay of an agreement against historical shipments.
type RateSimulation struct {
	bun.BaseModel             `bun:"table:rate_simulations,alias:rsim" json:"-"`
	pagination.CursorValueSet `bun:",embed"                            json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	// RateAgreementID is the contract being replayed. It is usually a draft:
	// the point is to see what it would do before anybody signs it.
	RateAgreementID pulid.ID `json:"rateAgreementId" bun:"rate_agreement_id,type:VARCHAR(100),notnull"`

	Name        string `json:"name"        bun:"name,type:VARCHAR(150),notnull"`
	Description string `json:"description" bun:"description,type:TEXT,nullzero"`

	Status Status `json:"status" bun:"status,type:rate_simulation_status_enum,notnull,default:'Pending'"`

	// PartyType says which side is being simulated. A carrier contract replay
	// answers "what would this cost us", which is the same machinery pointed
	// the other way.
	PartyType rateagreement.PartyType `json:"partyType" bun:"party_type,type:rate_agreement_party_type_enum,notnull,default:'Customer'"`

	// SampleFrom and SampleTo bound the shipments replayed, by their own ship
	// dates. Half open, matching every other window in the rating system: a
	// shipment on SampleTo belongs to the next period.
	SampleFrom int64 `json:"sampleFrom" bun:"sample_from,type:BIGINT,notnull"`
	SampleTo   int64 `json:"sampleTo"   bun:"sample_to,type:BIGINT,notnull"`

	// SampleLimit caps how many shipments are replayed. Zero means every
	// shipment in the window.
	SampleLimit int `json:"sampleLimit" bun:"sample_limit,type:INTEGER,notnull,default:0"`

	// Summary is what the run came to, written once at the end.
	Summary *ratesimulation.Summary `json:"summary" bun:"summary,type:JSONB,nullzero"`

	// RuleCoverage says what happened to each of the agreement's rules. It is
	// the half of the answer the revenue total cannot give: which lanes never
	// fired, and which fired but always lost.
	RuleCoverage []*RuleCoverage `json:"ruleCoverage" bun:"rule_coverage,type:JSONB,nullzero"`

	Error string `json:"error" bun:"error,type:TEXT,nullzero"`

	StartedAt   *int64    `json:"startedAt"   bun:"started_at,type:BIGINT,nullzero"`
	CompletedAt *int64    `json:"completedAt" bun:"completed_at,type:BIGINT,nullzero"`
	RequestedBy *pulid.ID `json:"requestedBy" bun:"requested_by_id,type:VARCHAR(100),nullzero"`

	// WorkflowID is the Temporal run driving this simulation, so a stuck run can
	// be found and cancelled.
	WorkflowID string `json:"workflowId" bun:"workflow_id,type:VARCHAR(255),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit         `json:"-"                   bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization         `json:"-"                   bun:"rel:belongs-to,join:organization_id=id"`
	Agreement    *rateagreement.RateAgreement `json:"agreement,omitempty" bun:"rel:belongs-to,join:rate_agreement_id=id"`
	Results      []*RateSimulationResult      `json:"results,omitempty"   bun:"rel:has-many,join:id=rate_simulation_id"`
}

// RuleCoverage is what happened to one rule across a whole simulation.
type RuleCoverage struct {
	RuleID  pulid.ID `json:"ruleId"`
	Label   string   `json:"label"`
	LaneKey string   `json:"laneKey"`

	Outcome RuleOutcome `json:"outcome"`

	// WonCount and LostCount are how many shipments this rule priced and how
	// many it matched but was outranked on.
	WonCount  int `json:"wonCount"`
	LostCount int `json:"lostCount"`

	// LostTo names the rule that most often beat this one, which is the single
	// most useful thing to know about a lane that never wins.
	LostTo      *pulid.ID `json:"lostTo,omitempty"`
	LostToLabel string    `json:"lostToLabel,omitempty"`
}

func (rs *RateSimulation) applyDefaults() {
	if rs.Status == "" {
		rs.Status = StatusPending
	}

	if rs.PartyType == "" {
		rs.PartyType = rateagreement.PartyTypeCustomer
	}
}

func (rs *RateSimulation) Validate(multiErr *errortypes.MultiError) {
	rs.applyDefaults()

	multiErr.AddOzzoError(validation.ValidateStruct(rs,
		validation.Field(&rs.RateAgreementID,
			validation.Required.Error("An agreement to simulate is required"),
		),
		validation.Field(&rs.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxNameLength).
				Error("Name must be between 1 and 150 characters"),
		),
		validation.Field(&rs.Description,
			validation.Length(0, maxDescriptionLength).
				Error("Description cannot be longer than 500 characters"),
		),
		validation.Field(&rs.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("Status is invalid"),
		),
		validation.Field(&rs.PartyType,
			validation.Required.Error("Party type is required"),
			domainvalidation.ValidEnum[rateagreement.PartyType]("Party type is invalid"),
		),
		validation.Field(&rs.SampleFrom,
			validation.Required.Error("A start date is required"),
		),
		validation.Field(&rs.SampleTo,
			validation.Required.Error("An end date is required"),
		),
	))

	rs.validateWindow(multiErr)
}

// validateWindow enforces a sample that can actually contain shipments.
//
// The window is half open, so a start equal to the end contains nothing. That
// is worth refusing rather than running: a simulation over zero shipments
// reports a zero delta, which reads as "this change costs nothing".
func (rs *RateSimulation) validateWindow(multiErr *errortypes.MultiError) {
	if rs.SampleFrom > 0 && rs.SampleTo > 0 && rs.SampleTo <= rs.SampleFrom {
		multiErr.Add(
			"sampleTo",
			errortypes.ErrInvalid,
			"The end of the sample must be after its start",
		)
	}

	if rs.SampleLimit < 0 {
		multiErr.Add("sampleLimit", errortypes.ErrInvalid, "The sample limit cannot be negative")
	}

	if rs.SampleLimit > maxShipmentSample {
		multiErr.Add(
			"sampleLimit",
			errortypes.ErrInvalid,
			"A simulation cannot replay more than 100,000 shipments",
		)
	}
}

// CanCancel reports whether there is still something to stop.
func (rs *RateSimulation) CanCancel() bool {
	return rs != nil && !rs.Status.IsTerminal()
}

func (rs *RateSimulation) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if rs.ID.IsNil() {
			rs.ID = pulid.MustNew("rsim_")
		}
		rs.CreatedAt = now
		rs.UpdatedAt = now
	case *bun.UpdateQuery:
		rs.UpdatedAt = now
	}

	return nil
}

func (rs *RateSimulation) GetID() pulid.ID { return rs.ID }

func (rs *RateSimulation) GetCreatedAt() int64 { return rs.CreatedAt }

func (rs *RateSimulation) GetOrganizationID() pulid.ID { return rs.OrganizationID }

func (rs *RateSimulation) GetBusinessUnitID() pulid.ID { return rs.BusinessUnitID }

func (rs *RateSimulation) GetTableName() string { return "rate_simulations" }
