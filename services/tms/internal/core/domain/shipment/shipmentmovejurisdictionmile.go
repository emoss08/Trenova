package shipment

import (
	"context"
	"regexp"

	"github.com/emoss08/trenova/internal/core/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/core/domain/storedmileage"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	JurisdictionDistanceUnitsMiles      = distanceprofile.DefaultDistanceUnits
	JurisdictionDistanceUnitsKilometers = "Kilometers"
)

var (
	jurisdictionCountryCodeRE = regexp.MustCompile(`^[A-Z]{2}$`)
	jurisdictionCodeRE        = regexp.MustCompile(`^[A-Z0-9]{1,5}$`)
)

type JurisdictionMileSource string

const (
	JurisdictionMileSourceRouteCalculation = JurisdictionMileSource("RouteCalculation")
	JurisdictionMileSourceManual           = JurisdictionMileSource("Manual")
)

func (s JurisdictionMileSource) String() string { return string(s) }

func (s JurisdictionMileSource) IsValid() bool {
	return s == JurisdictionMileSourceRouteCalculation || s == JurisdictionMileSourceManual
}

var (
	_ bun.BeforeAppendModelHook          = (*ShipmentMoveJurisdictionMile)(nil)
	_ validationframework.TenantedEntity = (*ShipmentMoveJurisdictionMile)(nil)
)

type ShipmentMoveJurisdictionMile struct {
	bun.BaseModel `json:"-" bun:"table:shipment_move_jurisdiction_miles,alias:smjm"`

	ID                pulid.ID               `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID               `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID    pulid.ID               `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ShipmentMoveID    pulid.ID               `json:"shipmentMoveId"    bun:"shipment_move_id,type:VARCHAR(100),notnull"`
	ShipmentID        pulid.ID               `json:"shipmentId"        bun:"shipment_id,type:VARCHAR(100),notnull"`
	CountryCode       string                 `json:"countryCode"       bun:"country_code,type:VARCHAR(2),notnull"`
	JurisdictionCode  string                 `json:"jurisdictionCode"  bun:"jurisdiction_code,type:VARCHAR(5),notnull"`
	Sequence          int                    `json:"sequence"          bun:"sequence,type:INTEGER,notnull"`
	Distance          float64                `json:"distance"          bun:"distance,type:FLOAT,notnull"`
	DistanceUnits     string                 `json:"distanceUnits"     bun:"distance_units,type:VARCHAR(50),notnull"`
	TollDistance      *float64               `json:"tollDistance"      bun:"toll_distance,type:FLOAT,nullzero"`
	FerryDistance     *float64               `json:"ferryDistance"     bun:"ferry_distance,type:FLOAT,nullzero"`
	Loaded            bool                   `json:"loaded"            bun:"loaded,type:BOOLEAN,notnull"`
	Source            JurisdictionMileSource `json:"source"            bun:"source,type:VARCHAR(50),notnull"`
	Provider          string                 `json:"provider"          bun:"provider,type:VARCHAR(50),nullzero"`
	DataVersion       string                 `json:"dataVersion"       bun:"data_version,type:VARCHAR(50),nullzero"`
	DistanceProfileID pulid.ID               `json:"distanceProfileId" bun:"distance_profile_id,type:VARCHAR(100),nullzero"`
	CalculatedAt      int64                  `json:"calculatedAt"      bun:"calculated_at,type:BIGINT,notnull"`
	Version           int64                  `json:"version"           bun:"version,type:BIGINT"`
	CreatedAt         int64                  `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64                  `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (m *ShipmentMoveJurisdictionMile) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		m,
		validation.Field(
			&m.ShipmentMoveID,
			validation.Required.Error("Shipment move is required"),
		),
		validation.Field(
			&m.ShipmentID,
			validation.Required.Error("Shipment is required"),
		),
		validation.Field(
			&m.CountryCode,
			validation.Required.Error("Country code is required"),
			validation.Match(jurisdictionCountryCodeRE).
				Error("Country code must be two uppercase letters"),
		),
		validation.Field(
			&m.JurisdictionCode,
			validation.Required.Error("Jurisdiction code is required"),
			validation.Match(jurisdictionCodeRE).
				Error("Jurisdiction code must be one to five uppercase letters or digits"),
		),
		validation.Field(
			&m.Sequence,
			validation.Min(0).Error("Sequence cannot be negative"),
		),
		validation.Field(
			&m.Distance,
			validation.Min(0.0).Error("Distance cannot be negative"),
		),
		validation.Field(
			&m.DistanceUnits,
			validation.Required.Error("Distance units are required"),
			validation.In(JurisdictionDistanceUnitsMiles, JurisdictionDistanceUnitsKilometers).
				Error("Distance units must be Miles or Kilometers"),
		),
		validation.Field(
			&m.TollDistance,
			validation.Min(0.0).Error("Toll distance cannot be negative"),
		),
		validation.Field(
			&m.FerryDistance,
			validation.Min(0.0).Error("Ferry distance cannot be negative"),
		),
		validation.Field(
			&m.Source,
			validation.Required.Error("Source is required"),
			domainvalidation.ValidEnum[JurisdictionMileSource]("Source is invalid"),
		),
		validation.Field(
			&m.CalculatedAt,
			validation.Required.Error("Calculated at is required"),
			validation.Min(int64(1)).Error("Calculated at must be a positive timestamp"),
		),
	))
}

func (m *ShipmentMoveJurisdictionMile) Key() string {
	return m.CountryCode + "_" + m.JurisdictionCode
}

func (m *ShipmentMoveJurisdictionMile) DistanceInMiles() float64 {
	return storedmileage.ConvertDistance(
		m.Distance,
		m.DistanceUnits,
		JurisdictionDistanceUnitsMiles,
	)
}

func (m *ShipmentMoveJurisdictionMile) GetID() pulid.ID { return m.ID }

func (m *ShipmentMoveJurisdictionMile) GetCreatedAt() int64 { return m.CreatedAt }

func (m *ShipmentMoveJurisdictionMile) GetOrganizationID() pulid.ID { return m.OrganizationID }

func (m *ShipmentMoveJurisdictionMile) GetBusinessUnitID() pulid.ID { return m.BusinessUnitID }

func (m *ShipmentMoveJurisdictionMile) GetTableName() string {
	return "shipment_move_jurisdiction_miles"
}

func (m *ShipmentMoveJurisdictionMile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if m.ID.IsNil() {
			m.ID = pulid.MustNew("smjm_")
		}
		if m.Source == "" {
			m.Source = JurisdictionMileSourceRouteCalculation
		}
		if m.DistanceUnits == "" {
			m.DistanceUnits = JurisdictionDistanceUnitsMiles
		}
		if m.CalculatedAt == 0 {
			m.CalculatedAt = now
		}
		m.CreatedAt = now
		m.UpdatedAt = now
	case *bun.UpdateQuery:
		m.UpdatedAt = now
	}

	return nil
}
