package distanceprofile

import (
	"context"
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
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*DistanceProfile)(nil)
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
	bun.BaseModel             `bun:"table:distance_profiles,alias:dp" json:"-"`
	keel.Table                `keel:"distance_profiles"`
	pagination.CursorValueSet `bun:",embed"                           json:"-"`

	ID                  pulid.ID         `json:"id"                  bun:"id,type:VARCHAR(100),pk,notnull" keel:"id,pk,autoid:dp_"`
	BusinessUnitID      pulid.ID         `json:"businessUnitId"      bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID      pulid.ID         `json:"organizationId"      bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	Name                string           `json:"name"                bun:"name,type:VARCHAR(100),notnull" keel:"name"`
	Description         string           `json:"description"         bun:"description,type:TEXT,nullzero" keel:"description,nullzero"`
	Status              string           `json:"status"              bun:"status,type:VARCHAR(20),notnull" keel:"status"`
	IsDefault           bool             `json:"isDefault"           bun:"is_default,type:BOOLEAN,notnull" keel:"is_default"`
	Provider            integration.Type `json:"provider"            bun:"provider,type:VARCHAR(50),notnull" keel:"provider"`
	DataVersion         string           `json:"dataVersion"         bun:"data_version,type:VARCHAR(50),notnull" keel:"data_version"`
	Region              string           `json:"region"              bun:"region,type:VARCHAR(20),notnull" keel:"region"`
	RoutingType         string           `json:"routingType"         bun:"routing_type,type:VARCHAR(50),notnull" keel:"routing_type"`
	DistanceUnits       string           `json:"distanceUnits"       bun:"distance_units,type:VARCHAR(50),notnull" keel:"distance_units"`
	LocationGranularity string           `json:"locationGranularity" bun:"location_granularity,type:VARCHAR(50),notnull" keel:"location_granularity"`
	ProfileName         string           `json:"profileName"         bun:"profile_name,type:VARCHAR(100),nullzero" keel:"profile_name,nullzero"`
	HighwayOnly         bool             `json:"highwayOnly"         bun:"highway_only,type:BOOLEAN,notnull" keel:"highway_only"`
	TollRoads           bool             `json:"tollRoads"           bun:"toll_roads,type:BOOLEAN,notnull" keel:"toll_roads"`
	BordersOpen         bool             `json:"bordersOpen"         bun:"borders_open,type:BOOLEAN,notnull" keel:"borders_open"`
	IncludeTollData     bool             `json:"includeTollData"     bun:"include_toll_data,type:BOOLEAN,notnull" keel:"include_toll_data"`
	Version             int64            `json:"version"             bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt           int64            `json:"createdAt"           bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt           int64            `json:"updatedAt"           bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
	SearchVector        string           `json:"-"                   bun:"search_vector,type:TSVECTOR,scanonly"`

	BusinessUnit *tenant.BusinessUnit `json:"-" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-" bun:"rel:belongs-to,join:organization_id=id"`
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

func (d *DistanceProfile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("dp_")
		}
		d.CreatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}
	return nil
}
