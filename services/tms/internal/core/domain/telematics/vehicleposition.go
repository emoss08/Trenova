package telematics

import (
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/shared/pulid"
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
