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

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/ratesimulation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
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
	keel.Table `keel:"rate_simulations"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:rsim_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	// RateAgreementID is the contract being replayed. It is usually a draft:
	// the point is to see what it would do before anybody signs it.
	RateAgreementID pulid.ID `json:"rateAgreementId" keel:"rate_agreement_id"`

	Name        string `json:"name" keel:"name"`
	Description string `json:"description" keel:"description,nullzero"`

	Status Status `json:"status" keel:"status"`

	// PartyType says which side is being simulated. A carrier contract replay
	// answers "what would this cost us", which is the same machinery pointed
	// the other way.
	PartyType rateagreement.PartyType `json:"partyType" keel:"party_type"`

	// SampleFrom and SampleTo bound the shipments replayed, by their own ship
	// dates. Half open, matching every other window in the rating system: a
	// shipment on SampleTo belongs to the next period.
	SampleFrom int64 `json:"sampleFrom" keel:"sample_from"`
	SampleTo   int64 `json:"sampleTo" keel:"sample_to"`

	// SampleLimit caps how many shipments are replayed. Zero means every
	// shipment in the window.
	SampleLimit int `json:"sampleLimit" keel:"sample_limit"`

	// Summary is what the run came to, written once at the end.
	Summary *ratesimulation.Summary `json:"summary" keel:"summary,jsonb"`

	// RuleCoverage says what happened to each of the agreement's rules. It is
	// the half of the answer the revenue total cannot give: which lanes never
	// fired, and which fired but always lost.
	RuleCoverage []*RuleCoverage `json:"ruleCoverage" keel:"rule_coverage,jsonb,nullzero"`

	Error string `json:"error" keel:"error,nullzero"`

	StartedAt   *int64    `json:"startedAt" keel:"started_at"`
	CompletedAt *int64    `json:"completedAt" keel:"completed_at"`
	RequestedBy *pulid.ID `json:"requestedBy" keel:"requested_by_id"`

	// WorkflowID is the Temporal run driving this simulation, so a stuck run can
	// be found and cancelled.
	WorkflowID string `json:"workflowId" keel:"workflow_id,nullzero"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit         `json:"-"`
	Organization *tenant.Organization         `json:"-"`
	Agreement    *rateagreement.RateAgreement `json:"agreement,omitempty" keel:"belongs:rate_agreement_id"`
	Results      []*RateSimulationResult      `json:"results,omitempty" keel:"many:rate_simulation_id"`
}

func (rs *RateSimulation) BeforeCreate(_ context.Context) error {
	if rs.Status == "" {
		rs.Status = StatusPending
	}
	if rs.PartyType == "" {
		rs.PartyType = rateagreement.PartyTypeCustomer
	}

	return nil
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

func (rs *RateSimulation) GetID() pulid.ID { return rs.ID }

func (rs *RateSimulation) GetCreatedAt() int64 { return rs.CreatedAt }

func (rs *RateSimulation) GetOrganizationID() pulid.ID { return rs.OrganizationID }

func (rs *RateSimulation) GetBusinessUnitID() pulid.ID { return rs.BusinessUnitID }

func (rs *RateSimulation) GetTableName() string { return "rate_simulations" }
