package distanceprofile

import (
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pcmiler"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*DistanceProfile)(nil)
	_ domaintypes.PostgresSearchable     = (*DistanceProfile)(nil)
	_ pagination.CursorEntity            = (*DistanceProfile)(nil)
)

const (
	StatusActive   = "Active"
	StatusInactive = "Inactive"

	RegionNorthAmerica = "NA"

	DefaultName                = "Default PC*Miler"
	DefaultDataVersion         = "Current"
	DefaultRegion              = RegionNorthAmerica
	DefaultRoutingType         = "Practical"
	DefaultDistanceUnits       = "Miles"
	DefaultLocationGranularity = "PostalCode"
)

type DistanceProfile struct {
	keel.Table          `keel:"distance_profiles"`
	ID                  pulid.ID         `json:"id" keel:"id,pk,autoid:dp_"`
	BusinessUnitID      pulid.ID         `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID      pulid.ID         `json:"organizationId" keel:"organization_id,pk,tenant"`
	Name                string           `json:"name" keel:"name"`
	Description         string           `json:"description" keel:"description,nullzero"`
	Status              string           `json:"status" keel:"status"`
	IsDefault           bool             `json:"isDefault" keel:"is_default"`
	Provider            integration.Type `json:"provider" keel:"provider"`
	DataVersion         string           `json:"dataVersion" keel:"data_version"`
	Region              string           `json:"region" keel:"region"`
	RoutingType         string           `json:"routingType" keel:"routing_type"`
	DistanceUnits       string           `json:"distanceUnits" keel:"distance_units"`
	LocationGranularity string           `json:"locationGranularity" keel:"location_granularity"`
	ProfileName         string           `json:"profileName" keel:"profile_name,nullzero"`
	HighwayOnly         bool             `json:"highwayOnly" keel:"highway_only"`
	TollRoads           bool             `json:"tollRoads" keel:"toll_roads"`
	BordersOpen         bool             `json:"bordersOpen" keel:"borders_open"`
	IncludeTollData     bool             `json:"includeTollData" keel:"include_toll_data"`
	Version             int64            `json:"version" keel:"version,version"`
	CreatedAt           int64            `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt           int64            `json:"updatedAt" keel:"updated_at,autoupdate"`
	SearchVector        string           `json:"-"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func NewDefault(orgID, buID pulid.ID) *DistanceProfile {
	return &DistanceProfile{
		OrganizationID:      orgID,
		BusinessUnitID:      buID,
		Name:                DefaultName,
		Description:         "Default PC*Miler routing profile",
		Status:              StatusActive,
		IsDefault:           true,
		Provider:            integration.TypePCMiler,
		DataVersion:         DefaultDataVersion,
		Region:              DefaultRegion,
		RoutingType:         DefaultRoutingType,
		DistanceUnits:       DefaultDistanceUnits,
		LocationGranularity: DefaultLocationGranularity,
		TollRoads:           true,
		BordersOpen:         true,
	}
}

func (d *DistanceProfile) ApplyDefaults() {
	d.Name = strings.TrimSpace(d.Name)
	d.Description = strings.TrimSpace(d.Description)
	d.ProfileName = strings.TrimSpace(d.ProfileName)
	if d.Status == "" {
		d.Status = StatusActive
	}
	if d.Provider == "" {
		d.Provider = integration.TypePCMiler
	}
	if strings.TrimSpace(d.DataVersion) == "" {
		d.DataVersion = DefaultDataVersion
	}
	if strings.TrimSpace(d.Region) == "" {
		d.Region = DefaultRegion
	}
	if strings.TrimSpace(d.RoutingType) == "" {
		d.RoutingType = DefaultRoutingType
	}
	if strings.TrimSpace(d.DistanceUnits) == "" {
		d.DistanceUnits = DefaultDistanceUnits
	}
	if strings.TrimSpace(d.LocationGranularity) == "" {
		d.LocationGranularity = DefaultLocationGranularity
	}
}

func (d *DistanceProfile) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		d,
		validation.Field(&d.Name, validation.Required.Error("Name is required")),
		validation.Field(&d.Provider, validation.Required.Error("Provider is required")),
		validation.Field(&d.RoutingType, validation.Required.Error("Routing type is required")),
		validation.Field(
			&d.DistanceUnits,
			validation.Required.Error("Distance units are required"),
		),
		validation.Field(
			&d.LocationGranularity,
			validation.Required.Error("Location granularity is required"),
		),
		validation.Field(&d.Region, validation.Required.Error("Region is required")),
	))
	if d.Provider != "" && d.Provider != integration.TypePCMiler {
		multiErr.Add("provider", errortypes.ErrInvalid, "Provider must be PCMiler")
	}
	if d.Status != StatusActive && d.Status != StatusInactive {
		multiErr.Add("status", errortypes.ErrInvalid, "Status must be Active or Inactive")
	}
	if d.Region != "" && d.Region != RegionNorthAmerica {
		multiErr.Add("region", errortypes.ErrInvalid, "Region must be NA")
	}
	if d.IsDefault && d.Status == StatusInactive {
		multiErr.Add("isDefault", errortypes.ErrInvalid, "Default profile must be active")
	}
}

func (d *DistanceProfile) RouteOptions() pcmiler.RouteOptions {
	return pcmiler.RouteOptions{
		DataVersion:         d.DataVersion,
		Region:              d.Region,
		RoutingType:         d.RoutingType,
		DistanceUnits:       d.DistanceUnits,
		VehicleType:         "Truck",
		LocationGranularity: d.LocationGranularity,
		ProfileName:         d.ProfileName,
		HighwayOnly:         d.HighwayOnly,
		TollRoads:           d.TollRoads,
		BordersOpen:         d.BordersOpen,
		IncludeTollData:     d.IncludeTollData,
	}
}

func (d *DistanceProfile) GetID() pulid.ID { return d.ID }

func (d *DistanceProfile) GetCreatedAt() int64 { return d.CreatedAt }

func (d *DistanceProfile) GetTableName() string { return "distance_profiles" }

func (d *DistanceProfile) GetOrganizationID() pulid.ID { return d.OrganizationID }

func (d *DistanceProfile) GetBusinessUnitID() pulid.ID { return d.BusinessUnitID }

func (d *DistanceProfile) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "dp",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{Name: "provider", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightC},
		},
	}
}
