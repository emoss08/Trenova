package dispatchplanprice

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/shared/pulid"
)

type UncoveredReason string

const (
	UncoveredNoCandidate     = UncoveredReason("NoCandidate")
	UncoveredDisqualified    = UncoveredReason("Disqualified")
	UncoveredBeyondDeadhead  = UncoveredReason("BeyondDeadhead")
	UncoveredLocationUnknown = UncoveredReason("LocationUnknown")
	UncoveredNotWorthIt      = UncoveredReason("NotWorthIt")
	UncoveredOutbid          = UncoveredReason("Outbid")
)

type Alternative struct {
	WorkerID   pulid.ID                 `json:"workerId"`
	WorkerName string                   `json:"workerName"`
	ExtraCents int64                    `json:"extraCents"`
	CostCents  int64                    `json:"costCents"`
	Lines      []dispatcheconomics.Line `json:"lines"`
}

type DispatchPlanPrice struct {
	keel.Table `keel:"dispatch_plan_prices"`

	ID                   pulid.ID                 `json:"id"                   keel:"id,pk,autoid:dpp_"`
	BusinessUnitID       pulid.ID                 `json:"businessUnitId"       keel:"business_unit_id,pk,tenant"`
	OrganizationID       pulid.ID                 `json:"organizationId"       keel:"organization_id,pk,tenant"`
	MoveID               pulid.ID                 `json:"moveId"               keel:"move_id"`
	WorkerID             pulid.ID                 `json:"workerId"             keel:"worker_id,nullzero"`
	WorkerName           string                   `json:"workerName"           keel:"worker_name"`
	CostCents            int64                    `json:"costCents"            keel:"cost_cents"`
	CostLines            []dispatcheconomics.Line `json:"costLines"            keel:"cost_lines,jsonb"`
	PreferenceFloorCents int64                    `json:"preferenceFloorCents" keel:"preference_floor_cents"`
	CoveragePriceCents   int64                    `json:"coveragePriceCents"   keel:"coverage_price_cents"`
	DriverValueCents     int64                    `json:"driverValueCents"     keel:"driver_value_cents"`
	UncoveredReason      UncoveredReason          `json:"uncoveredReason"      keel:"uncovered_reason"`
	Alternatives         []Alternative            `json:"alternatives"         keel:"alternatives,jsonb"`
	GeneratedAt          int64                    `json:"generatedAt"          keel:"generated_at"`
	Version              int64                    `json:"version"              keel:"version,version"`
	CreatedAt            int64                    `json:"createdAt"            keel:"created_at,autocreate"`
	UpdatedAt            int64                    `json:"updatedAt"            keel:"updated_at,autoupdate"`
}

func (p *DispatchPlanPrice) Covered() bool {
	return !p.WorkerID.IsNil()
}
