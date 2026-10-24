package stopeta

import (
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type StopETA struct {
	bun.BaseModel `bun:"table:stop_etas,alias:seta" json:"-"`
	keel.Table    `keel:"stop_etas"`

	ID                 pulid.ID    `json:"id"                 bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk"`
	OrganizationID     pulid.ID    `json:"organizationId"     bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID     pulid.ID    `json:"businessUnitId"     bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	ShipmentID         pulid.ID    `json:"shipmentId"         bun:"shipment_id,type:VARCHAR(100),notnull" keel:"shipment_id"`
	ShipmentMoveID     pulid.ID    `json:"shipmentMoveId"     bun:"shipment_move_id,type:VARCHAR(100),notnull" keel:"shipment_move_id"`
	StopID             pulid.ID    `json:"stopId"             bun:"stop_id,type:VARCHAR(100),notnull" keel:"stop_id"`
	MoveSequence       int64       `json:"moveSequence"       bun:"move_sequence,type:INTEGER,notnull" keel:"move_sequence"`
	StopSequence       int64       `json:"stopSequence"       bun:"stop_sequence,type:INTEGER,notnull" keel:"stop_sequence"`
	Arrived            bool        `json:"arrived"            bun:"arrived,type:BOOLEAN,notnull,default:false" keel:"arrived"`
	EstimatedArrival   *int64      `json:"estimatedArrival"   bun:"estimated_arrival,type:BIGINT,nullzero" keel:"estimated_arrival"`
	EstimatedDeparture *int64      `json:"estimatedDeparture" bun:"estimated_departure,type:BIGINT,nullzero" keel:"estimated_departure"`
	WindowEnd          *int64      `json:"windowEnd"          bun:"window_end,type:BIGINT,nullzero" keel:"window_end"`
	SlackMinutes       *int64      `json:"slackMinutes"       bun:"slack_minutes,type:INTEGER" keel:"slack_minutes"`
	Verdict            Verdict     `json:"verdict"            bun:"verdict,type:VARCHAR(20),notnull" keel:"verdict"`
	Basis              Basis       `json:"basis"              bun:"basis,type:VARCHAR(20),notnull" keel:"basis"`
	DriveMethod        DriveMethod `json:"driveMethod"        bun:"drive_method,type:VARCHAR(20),notnull" keel:"drive_method"`
	MilesRemaining     float64     `json:"milesRemaining"     bun:"miles_remaining,type:DOUBLE PRECISION,notnull,default:0" keel:"miles_remaining"`
	DriveMinutes       int64       `json:"driveMinutes"       bun:"drive_minutes,type:INTEGER,notnull,default:0" keel:"drive_minutes"`
	RestMinutes        int64       `json:"restMinutes"        bun:"rest_minutes,type:INTEGER,notnull,default:0" keel:"rest_minutes"`
	InputAt            *int64      `json:"inputAt"            bun:"input_at,type:BIGINT,nullzero" keel:"input_at"`
	InputStale         bool        `json:"inputStale"         bun:"input_stale,type:BOOLEAN,notnull,default:false" keel:"input_stale"`
	ThresholdMinutes   int64       `json:"thresholdMinutes"   bun:"threshold_minutes,type:INTEGER,notnull" keel:"threshold_minutes"`
	Reason             string      `json:"reason"             bun:"reason,type:VARCHAR(500),nullzero" keel:"reason,nullzero"`
	ComputedAt         int64       `json:"computedAt"         bun:"computed_at,type:BIGINT,notnull" keel:"computed_at"`
}

func NewID() pulid.ID {
	return pulid.MustNew("seta_")
}

func (e *StopETA) NeedsAttention() bool {
	return e != nil && !e.Arrived && e.Verdict.NeedsAttention()
}

func (e *StopETA) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&e.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&e.ShipmentID, validation.Required.Error("Shipment is required")),
		validation.Field(&e.ShipmentMoveID, validation.Required.Error("Move is required")),
		validation.Field(&e.StopID, validation.Required.Error("Stop is required")),
		validation.Field(&e.Verdict,
			validation.Required.Error("Verdict is required"),
			domainvalidation.ValidEnum[Verdict]("Verdict is invalid"),
		),
		validation.Field(&e.Basis,
			validation.Required.Error("Basis is required"),
			domainvalidation.ValidEnum[Basis]("Basis is invalid"),
		),
		validation.Field(&e.DriveMethod,
			validation.Required.Error("Drive method is required"),
			domainvalidation.ValidEnum[DriveMethod]("Drive method is invalid"),
		),
		validation.Field(&e.MilesRemaining,
			validation.Min(0.0).Error("Miles remaining cannot be negative"),
		),
		validation.Field(&e.DriveMinutes,
			validation.Min(int64(0)).Error("Drive minutes cannot be negative"),
		),
		validation.Field(&e.RestMinutes,
			validation.Min(int64(0)).Error("Rest minutes cannot be negative"),
		),
		validation.Field(&e.ThresholdMinutes,
			validation.Min(int64(MinRiskThresholdMinutes)).
				Error("Risk threshold must be between 0 and 1440 minutes"),
			validation.Max(int64(MaxRiskThresholdMinutes)).
				Error("Risk threshold must be between 0 and 1440 minutes"),
		),
		validation.Field(&e.Reason,
			validation.RuneLength(0, MaxReasonLength).
				Error("Reason cannot be longer than 500 characters"),
		),
		validation.Field(&e.ComputedAt,
			validation.Required.Error("Computed time is required"),
		),
	))
}

func TrackedShipmentStatuses() []shipment.Status {
	return []shipment.Status{
		shipment.StatusPartiallyAssigned,
		shipment.StatusAssigned,
		shipment.StatusInTransit,
		shipment.StatusDelayed,
		shipment.StatusPartiallyCompleted,
	}
}

func TrackedShipment(status shipment.Status) bool {
	return slices.Contains(TrackedShipmentStatuses(), status)
}

func Governing(rows []*StopETA) *StopETA {
	var next, worst *StopETA
	for _, row := range rows {
		if row == nil || row.Arrived {
			continue
		}
		if next == nil {
			next = row
		}
		if row.NeedsAttention() && (worst == nil || row.Verdict.Rank() > worst.Verdict.Rank()) {
			worst = row
		}
	}

	switch {
	case worst != nil:
		return worst
	case next != nil:
		return next
	case len(rows) > 0:
		return rows[0]
	default:
		return nil
	}
}

func IndexByStop(rows []*StopETA) map[pulid.ID]*StopETA {
	out := make(map[pulid.ID]*StopETA, len(rows))
	for _, row := range rows {
		if row != nil {
			out[row.StopID] = row
		}
	}
	return out
}
