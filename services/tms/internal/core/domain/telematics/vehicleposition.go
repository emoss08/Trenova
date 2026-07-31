package telematics

import (
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type VehiclePosition struct {
	bun.BaseModel `bun:"table:telematics_vehicle_positions,alias:tvp" json:"-"`

	OrganizationID    pulid.ID    `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID    `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	TractorID         pulid.ID    `json:"tractorId"         bun:"tractor_id,pk,type:VARCHAR(100),notnull"`
	Provider          string      `json:"provider"          bun:"provider,type:VARCHAR(32),notnull,default:'Samsara'"`
	ProviderVehicleID string      `json:"providerVehicleId" bun:"provider_vehicle_id,type:TEXT,notnull"`
	Latitude          float64     `json:"latitude"          bun:"latitude,type:DOUBLE PRECISION,notnull"`
	Longitude         float64     `json:"longitude"         bun:"longitude,type:DOUBLE PRECISION,notnull"`
	HeadingDegrees    float64     `json:"headingDegrees"    bun:"heading_degrees,type:DOUBLE PRECISION,notnull,default:0"`
	SpeedMph          float64     `json:"speedMph"          bun:"speed_mph,type:DOUBLE PRECISION,notnull,default:0"`
	EngineState       EngineState `json:"engineState"       bun:"engine_state,type:VARCHAR(16),nullzero"`
	FuelPercent       *float64    `json:"fuelPercent"       bun:"fuel_percent,type:DOUBLE PRECISION,nullzero"`
	OdometerMeters    *int64      `json:"odometerMeters"    bun:"odometer_meters,type:BIGINT,nullzero"`
	FormattedLocation string      `json:"formattedLocation" bun:"formatted_location,type:TEXT,nullzero"`
	RecordedAt        int64       `json:"recordedAt"        bun:"recorded_at,type:BIGINT,notnull"`
	ReceivedAt        int64       `json:"receivedAt"        bun:"received_at,type:BIGINT,notnull"`

	Tractor *tractor.Tractor `json:"tractor,omitempty" bun:"rel:belongs-to,join:tractor_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (p *VehiclePosition) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&p.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&p.TractorID, validation.Required.Error("Tractor is required")),
		validation.Field(&p.Provider,
			validation.Required.Error("Provider is required"),
			validation.Length(1, maxProviderLength).
				Error("Provider cannot be longer than 32 characters"),
		),
		validation.Field(&p.ProviderVehicleID,
			validation.Required.Error("Provider vehicle id is required"),
		),
		// A position outside the coordinate system would place the tractor off
		// the map, which is what dispatch plans the next load from.
		validation.Field(&p.Latitude,
			validation.Min(minLatitude).Error("Latitude must be between -90 and 90"),
			validation.Max(maxLatitude).Error("Latitude must be between -90 and 90"),
		),
		validation.Field(&p.Longitude,
			validation.Min(minLongitude).Error("Longitude must be between -180 and 180"),
			validation.Max(maxLongitude).Error("Longitude must be between -180 and 180"),
		),
		validation.Field(&p.HeadingDegrees,
			validation.Min(0.0).Error("Heading must be between 0 and 360"),
			validation.Max(maxHeadingDegrees).Error("Heading must be between 0 and 360"),
		),
		validation.Field(&p.SpeedMph, validation.Min(0.0).Error("Speed cannot be negative")),
		validation.Field(&p.EngineState,
			domainvalidation.ValidEnum[EngineState]("Engine state is invalid"),
		),
		validation.Field(&p.FuelPercent,
			validation.Min(0.0).Error("Fuel percent must be between 0 and 100"),
			validation.Max(maxFuelPercent).Error("Fuel percent must be between 0 and 100"),
		),
		validation.Field(&p.OdometerMeters,
			validation.Min(int64(0)).Error("Odometer cannot be negative"),
		),
		validation.Field(&p.RecordedAt,
			validation.Required.Error("Recorded at is required"),
			validation.Min(int64(1)).Error("Recorded at must be a valid timestamp"),
		),
		validation.Field(&p.ReceivedAt,
			validation.Required.Error("Received at is required"),
			validation.Min(int64(1)).Error("Received at must be a valid timestamp"),
		),
	))
}

const (
	minLatitude       = -90.0
	maxLatitude       = 90.0
	minLongitude      = -180.0
	maxLongitude      = 180.0
	maxHeadingDegrees = 360.0
	maxFuelPercent    = 100.0
)
