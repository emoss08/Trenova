package ratesimulation

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*RateSimulationResult)(nil)

// RateSimulationResult is one shipment, priced two ways.
//
// The rows are kept rather than only the summary because the summary answers
// "should we do this" and the rows answer "who is going to call". A general
// rate increase that adds four percent overall can still double one customer's
// lane, and that is the shipment somebody needs to find before the customer
// does.
type RateSimulationResult struct {
	bun.BaseModel `bun:"table:rate_simulation_results,alias:rsr" json:"-"`

	ID               pulid.ID `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID   pulid.ID `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	RateSimulationID pulid.ID `json:"rateSimulationId" bun:"rate_simulation_id,type:VARCHAR(100),notnull"`

	ShipmentID pulid.ID `json:"shipmentId" bun:"shipment_id,type:VARCHAR(100),notnull"`
	ProNumber  string   `json:"proNumber"  bun:"pro_number,type:VARCHAR(100),nullzero"`

	// CustomerID and LaneKey are copied onto the row so the result grid can be
	// grouped without joining back to shipments that may since have been
	// edited. A simulation is a record of a moment.
	CustomerID *pulid.ID `json:"customerId" bun:"customer_id,type:VARCHAR(100),nullzero"`
	LaneKey    string    `json:"laneKey"    bun:"lane_key,type:VARCHAR(160),nullzero"`

	// EquipmentTypeID is what the load ran on, which is the third dimension
	// pricing changes are usually judged by.
	EquipmentTypeID *pulid.ID `json:"equipmentTypeId" bun:"equipment_type_id,type:VARCHAR(100),nullzero"`

	// BeforeAmount is what the shipment was actually billed. AfterAmount is
	// what the simulated agreement would have charged.
	BeforeAmount decimal.Decimal `json:"beforeAmount" bun:"before_amount,type:NUMERIC(19,4),notnull,default:0"`
	AfterAmount  decimal.Decimal `json:"afterAmount"  bun:"after_amount,type:NUMERIC(19,4),notnull,default:0"`
	Delta        decimal.Decimal `json:"delta"        bun:"delta,type:NUMERIC(19,4),notnull,default:0"`
	DeltaPercent decimal.Decimal `json:"deltaPercent" bun:"delta_percent,type:NUMERIC(9,4),notnull,default:0"`

	Outcome ratequote.Outcome `json:"outcome" bun:"outcome,type:rate_quote_outcome_enum,notnull"`

	// BeforeRuleID and AfterRuleID name the rules that priced each side, which
	// is what turns "this went up" into "this went up because this lane now
	// wins".
	BeforeRuleID *pulid.ID `json:"beforeRuleId" bun:"before_rule_id,type:VARCHAR(100),nullzero"`
	AfterRuleID  *pulid.ID `json:"afterRuleId"  bun:"after_rule_id,type:VARCHAR(100),nullzero"`

	Error string `json:"error" bun:"error,type:TEXT,nullzero"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Simulation *RateSimulation `json:"-" bun:"rel:belongs-to,join:rate_simulation_id=id"`
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

func (r *RateSimulationResult) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("rsr_")
		}
		r.CreatedAt = timeutils.NowUnix()
	}

	return nil
}

func (r *RateSimulationResult) GetID() pulid.ID { return r.ID }

func (r *RateSimulationResult) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *RateSimulationResult) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *RateSimulationResult) GetTableName() string { return "rate_simulation_results" }
