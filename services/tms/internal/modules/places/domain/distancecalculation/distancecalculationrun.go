package distancecalculation

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	SourceOverride      = "Override"
	SourceStoredMileage = "StoredMileage"
	SourcePCMiler       = "PCMiler"
	SourceManual        = "Manual"
)

type Run struct {
	keel.Table `keel:"distance_calculation_runs"`

	ID              pulid.ID       `json:"id"              keel:"id,pk,autoid:dcr_"`
	BusinessUnitID  pulid.ID       `json:"businessUnitId"  keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID       `json:"organizationId"  keel:"organization_id,pk,tenant"`
	ShipmentID      pulid.ID       `json:"shipmentId"      keel:"shipment_id"`
	ShipmentMoveID  pulid.ID       `json:"shipmentMoveId"  keel:"shipment_move_id"`
	Provider        string         `json:"provider"        keel:"provider,nullzero"`
	Source          string         `json:"source"          keel:"source"`
	RequestSummary  map[string]any `json:"requestSummary"  keel:"request_summary,jsonb,nullzero"`
	ResponseSummary map[string]any `json:"responseSummary" keel:"response_summary,jsonb,nullzero"`
	Status          string         `json:"status"          keel:"status"`
	ErrorCode       string         `json:"errorCode"       keel:"error_code,nullzero"`
	ErrorMessage    string         `json:"errorMessage"    keel:"error_message,nullzero"`
	LatencyMillis   int64          `json:"latencyMillis"   keel:"latency_millis"`
	CreatedAt       int64          `json:"createdAt"       keel:"created_at,autocreate"`
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
