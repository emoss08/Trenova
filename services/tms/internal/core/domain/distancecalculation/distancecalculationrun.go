package distancecalculation

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const (
	SourceOverride      = "Override"
	SourceStoredMileage = "StoredMileage"
	SourcePCMiler       = "PCMiler"
	SourceManual        = "Manual"
)

type Run struct {
	bun.BaseModel `bun:"table:distance_calculation_runs,alias:dcr" json:"-"`

	ID              pulid.ID       `json:"id"              bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID  pulid.ID       `json:"businessUnitId"  bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID  pulid.ID       `json:"organizationId"  bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	ShipmentID      pulid.ID       `json:"shipmentId"      bun:"shipment_id,type:VARCHAR(100),notnull"`
	ShipmentMoveID  pulid.ID       `json:"shipmentMoveId"  bun:"shipment_move_id,type:VARCHAR(100),notnull"`
	Provider        string         `json:"provider"        bun:"provider,type:VARCHAR(50),nullzero"`
	Source          string         `json:"source"          bun:"source,type:VARCHAR(50),notnull"`
	RequestSummary  map[string]any `json:"requestSummary"  bun:"request_summary,type:JSONB,nullzero"`
	ResponseSummary map[string]any `json:"responseSummary" bun:"response_summary,type:JSONB,nullzero"`
	Status          string         `json:"status"          bun:"status,type:VARCHAR(50),notnull"`
	ErrorCode       string         `json:"errorCode"       bun:"error_code,type:VARCHAR(100),nullzero"`
	ErrorMessage    string         `json:"errorMessage"    bun:"error_message,type:TEXT,nullzero"`
	LatencyMillis   int64          `json:"latencyMillis"   bun:"latency_millis,type:BIGINT,notnull,default:0"`
	CreatedAt       int64          `json:"createdAt"       bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (r *Run) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); !ok {
		return nil
	}
	if r.ID.IsNil() {
		r.ID = pulid.MustNew("dcr_")
	}
	r.CreatedAt = timeutils.NowUnix()
	return nil
}
