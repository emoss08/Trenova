package shipment

import (
	"context"
	"regexp"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/modules/places/domain/storedmileage"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	_ validationframework.TenantedEntity = (*ShipmentMoveJurisdictionMile)(nil)
)

type ShipmentMoveJurisdictionMile struct {
	keel.Table `keel:"shipment_move_jurisdiction_miles"`

	ID                pulid.ID               `json:"id"                keel:"id,pk,autoid:smjm_"`
	BusinessUnitID    pulid.ID               `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID               `json:"organizationId"    keel:"organization_id,pk,tenant"`
	ShipmentMoveID    pulid.ID               `json:"shipmentMoveId"    keel:"shipment_move_id"`
	ShipmentID        pulid.ID               `json:"shipmentId"        keel:"shipment_id"`
	CountryCode       string                 `json:"countryCode"       keel:"country_code"`
	JurisdictionCode  string                 `json:"jurisdictionCode"  keel:"jurisdiction_code"`
	Sequence          int                    `json:"sequence"          keel:"sequence"`
	Distance          float64                `json:"distance"          keel:"distance"`
	DistanceUnits     string                 `json:"distanceUnits"     keel:"distance_units"`
	TollDistance      *float64               `json:"tollDistance"      keel:"toll_distance"`
	FerryDistance     *float64               `json:"ferryDistance"     keel:"ferry_distance"`
	Loaded            bool                   `json:"loaded"            keel:"loaded"`
	Source            JurisdictionMileSource `json:"source"            keel:"source"`
	Provider          string                 `json:"provider"          keel:"provider,nullzero"`
	DataVersion       string                 `json:"dataVersion"       keel:"data_version,nullzero"`
	DistanceProfileID pulid.ID               `json:"distanceProfileId" keel:"distance_profile_id,nullzero"`
	CalculatedAt      int64                  `json:"calculatedAt"      keel:"calculated_at"`
	Version           int64                  `json:"version"           keel:"version,version"`
	CreatedAt         int64                  `json:"createdAt"         keel:"created_at,autocreate"`
	UpdatedAt         int64                  `json:"updatedAt"         keel:"updated_at,autoupdate"`
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

func (m *ShipmentMoveJurisdictionMile) BeforeCreate(ctx context.Context) error {
	now := timeutils.NowUnix()
	if m.Source == "" {
		m.Source = JurisdictionMileSourceRouteCalculation
	}
	if m.DistanceUnits == "" {
		m.DistanceUnits = JurisdictionDistanceUnitsMiles
	}
	if m.CalculatedAt == 0 {
		m.CalculatedAt = now
	}

	return nil
}
