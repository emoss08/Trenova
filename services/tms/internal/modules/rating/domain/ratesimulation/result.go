package ratesimulation

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// RateSimulationResult is one shipment, priced two ways.
//
// The rows are kept rather than only the summary because the summary answers
// "should we do this" and the rows answer "who is going to call". A general
// rate increase that adds four percent overall can still double one customer's
// lane, and that is the shipment somebody needs to find before the customer
// does.
type RateSimulationResult struct {
	keel.Table `keel:"rate_simulation_results"`

	ID               pulid.ID `json:"id"               keel:"id,pk,autoid:rsr_"`
	BusinessUnitID   pulid.ID `json:"businessUnitId"   keel:"business_unit_id,pk,tenant"`
	OrganizationID   pulid.ID `json:"organizationId"   keel:"organization_id,pk,tenant"`
	RateSimulationID pulid.ID `json:"rateSimulationId" keel:"rate_simulation_id"`

	ShipmentID pulid.ID `json:"shipmentId" keel:"shipment_id"`
	ProNumber  string   `json:"proNumber"  keel:"pro_number,nullzero"`

	// CustomerID and LaneKey are copied onto the row so the result grid can be
	// grouped without joining back to shipments that may since have been
	// edited. A simulation is a record of a moment.
	CustomerID *pulid.ID `json:"customerId" keel:"customer_id"`
	LaneKey    string    `json:"laneKey"    keel:"lane_key,nullzero"`

	// EquipmentTypeID is what the load ran on, which is the third dimension
	// pricing changes are usually judged by.
	EquipmentTypeID *pulid.ID `json:"equipmentTypeId" keel:"equipment_type_id"`

	// BeforeAmount is what the shipment was actually billed. AfterAmount is
	// what the simulated agreement would have charged.
	BeforeAmount decimal.Decimal `json:"beforeAmount" keel:"before_amount"`
	AfterAmount  decimal.Decimal `json:"afterAmount"  keel:"after_amount"`
	Delta        decimal.Decimal `json:"delta"        keel:"delta"`
	DeltaPercent decimal.Decimal `json:"deltaPercent" keel:"delta_percent"`

	Outcome ratequote.Outcome `json:"outcome" keel:"outcome"`

	// BeforeRuleID and AfterRuleID name the rules that priced each side, which
	// is what turns "this went up" into "this went up because this lane now
	// wins".
	BeforeRuleID *pulid.ID `json:"beforeRuleId" keel:"before_rule_id"`
	AfterRuleID  *pulid.ID `json:"afterRuleId"  keel:"after_rule_id"`

	Error string `json:"error" keel:"error,nullzero"`

	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`

	Simulation *RateSimulation `json:"-" keel:"belongs:rate_simulation_id"`
}

// Failed reports whether this shipment could be priced at all.
func (r *RateSimulationResult) Failed() bool {
	return r.Error != "" || !r.Outcome.Priced()
}

func (r *RateSimulationResult) Validate(multiErr *errortypes.MultiError) {
	if r.ShipmentID.IsNil() {
		multiErr.Add("shipmentId", errortypes.ErrRequired, "Shipment is required")
	}

	if r.RateSimulationID.IsNil() {
		multiErr.Add("rateSimulationId", errortypes.ErrRequired, "Simulation is required")
	}
}

func (r *RateSimulationResult) GetID() pulid.ID { return r.ID }

func (r *RateSimulationResult) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *RateSimulationResult) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *RateSimulationResult) GetTableName() string { return "rate_simulation_results" }
