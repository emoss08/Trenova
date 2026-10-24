package distancecalculation

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	keel.Table    `keel:"distance_calculation_runs"`

	ID              pulid.ID       `json:"id"              bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:dcr_"`
	BusinessUnitID  pulid.ID       `json:"businessUnitId"  bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID       `json:"organizationId"  bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	ShipmentID      pulid.ID       `json:"shipmentId"      bun:"shipment_id,type:VARCHAR(100),notnull" keel:"shipment_id"`
	ShipmentMoveID  pulid.ID       `json:"shipmentMoveId"  bun:"shipment_move_id,type:VARCHAR(100),notnull" keel:"shipment_move_id"`
	Provider        string         `json:"provider"        bun:"provider,type:VARCHAR(50),nullzero" keel:"provider,nullzero"`
	Source          string         `json:"source"          bun:"source,type:VARCHAR(50),notnull" keel:"source"`
	RequestSummary  map[string]any `json:"requestSummary"  bun:"request_summary,type:JSONB,nullzero" keel:"request_summary,jsonb,nullzero"`
	ResponseSummary map[string]any `json:"responseSummary" bun:"response_summary,type:JSONB,nullzero" keel:"response_summary,jsonb,nullzero"`
	Status          string         `json:"status"          bun:"status,type:VARCHAR(50),notnull" keel:"status"`
	ErrorCode       string         `json:"errorCode"       bun:"error_code,type:VARCHAR(100),nullzero" keel:"error_code,nullzero"`
	ErrorMessage    string         `json:"errorMessage"    bun:"error_message,type:TEXT,nullzero" keel:"error_message,nullzero"`
	LatencyMillis   int64          `json:"latencyMillis"   bun:"latency_millis,type:BIGINT,notnull" keel:"latency_millis"`
	CreatedAt       int64          `json:"createdAt"       bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
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

func (r *Run) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&r.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&r.ShipmentID, validation.Required.Error("Shipment is required")),
		validation.Field(&r.ShipmentMoveID,
			validation.Required.Error("Shipment move is required"),
		),
		validation.Field(&r.Provider,
			validation.Length(0, maxRunCodeLength).
				Error("Provider cannot be longer than 50 characters"),
		),
		validation.Field(&r.Source,
			validation.Required.Error("Source is required"),
			validation.Length(1, maxRunCodeLength).
				Error("Source cannot be longer than 50 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.Length(1, maxRunCodeLength).
				Error("Status cannot be longer than 50 characters"),
		),
		validation.Field(&r.ErrorCode,
			validation.Length(0, maxRunErrorCodeLength).
				Error("Error code cannot be longer than 100 characters"),
		),
		validation.Field(&r.LatencyMillis,
			validation.Min(int64(0)).Error("Latency cannot be negative"),
		),
	))
}

const (
	maxRunCodeLength      = 50
	maxRunErrorCodeLength = 100
)
