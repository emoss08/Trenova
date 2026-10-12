package tender

import (
	"context"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	SpecificityLocationPair = int16(3)
	SpecificityCityPair     = int16(2)
	SpecificityStatePair    = int16(1)
)

var _ bun.BeforeAppendModelHook = (*RoutingGuide)(nil)

type RoutingGuide struct {
	bun.BaseModel `bun:"table:routing_guides,alias:rgd" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`

	Name                  string             `json:"name"                  bun:"name,type:VARCHAR(255),notnull"`
	Description           string             `json:"description"           bun:"description,type:TEXT,nullzero"`
	Status                domaintypes.Status `json:"status"                bun:"status,type:VARCHAR(50),notnull,default:'Active'"`
	OriginLocationID      *pulid.ID          `json:"originLocationId"      bun:"origin_location_id,type:VARCHAR(100),nullzero"`
	DestinationLocationID *pulid.ID          `json:"destinationLocationId" bun:"destination_location_id,type:VARCHAR(100),nullzero"`
	OriginCity            string             `json:"originCity"            bun:"origin_city,type:VARCHAR(100),nullzero"`
	OriginState           string             `json:"originState"           bun:"origin_state,type:VARCHAR(2),nullzero"`
	DestinationCity       string             `json:"destinationCity"       bun:"destination_city,type:VARCHAR(100),nullzero"`
	DestinationState      string             `json:"destinationState"      bun:"destination_state,type:VARCHAR(2),nullzero"`
	Specificity           int16              `json:"specificity"           bun:"specificity,type:SMALLINT,notnull"`
	Version               int64              `json:"version"               bun:"version,type:BIGINT"`
	CreatedAt             int64              `json:"createdAt"             bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt             int64              `json:"updatedAt"             bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Entries []*RoutingGuideEntry `json:"entries,omitempty" bun:"rel:has-many,join:id=routing_guide_id"`
}

func (rg *RoutingGuide) originSpecificity() int16 {
	switch {
	case rg.OriginLocationID != nil && !rg.OriginLocationID.IsNil():
		return SpecificityLocationPair
	case rg.OriginCity != "" && rg.OriginState != "":
		return SpecificityCityPair
	case rg.OriginState != "":
		return SpecificityStatePair
	}
	return 0
}

func (rg *RoutingGuide) destinationSpecificity() int16 {
	switch {
	case rg.DestinationLocationID != nil && !rg.DestinationLocationID.IsNil():
		return SpecificityLocationPair
	case rg.DestinationCity != "" && rg.DestinationState != "":
		return SpecificityCityPair
	case rg.DestinationState != "":
		return SpecificityStatePair
	}
	return 0
}

func (rg *RoutingGuide) ComputeSpecificity() int16 {
	origin := rg.originSpecificity()
	dest := rg.destinationSpecificity()
	if origin == 0 || dest == 0 || origin != dest {
		return 0
	}
	return origin
}

func (rg *RoutingGuide) applyDefaults() {
	if rg.Status == "" {
		rg.Status = domaintypes.StatusActive
	}
	rg.Specificity = rg.ComputeSpecificity()
}

func (rg *RoutingGuide) Validate(multiErr *errortypes.MultiError) {
	rg.applyDefaults()

	multiErr.AddOzzoError(validation.ValidateStruct(rg,
		validation.Field(&rg.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
		validation.Field(&rg.Status,
			validation.Required.Error("Status is required"),
			validation.In(domaintypes.StatusActive, domaintypes.StatusInactive).
				Error("Status is invalid"),
		),
	))

	origin := rg.originSpecificity()
	dest := rg.destinationSpecificity()

	if origin == 0 {
		multiErr.Add(
			"originLocationId",
			errortypes.ErrInvalid,
			"Origin requires a location, a city and state, or a state",
		)
	}
	if dest == 0 {
		multiErr.Add(
			"destinationLocationId",
			errortypes.ErrInvalid,
			"Destination requires a location, a city and state, or a state",
		)
	}
	if origin != 0 && dest != 0 && origin != dest {
		multiErr.Add(
			"originLocationId",
			errortypes.ErrInvalid,
			"Origin and destination must be defined at the same level of detail",
		)
	}

	hasOriginLocation := rg.OriginLocationID != nil && !rg.OriginLocationID.IsNil()
	if hasOriginLocation && (rg.OriginCity != "" || rg.OriginState != "") {
		multiErr.Add(
			"originCity",
			errortypes.ErrInvalid,
			"An origin location cannot be combined with city or state criteria",
		)
	}
	hasDestLocation := rg.DestinationLocationID != nil && !rg.DestinationLocationID.IsNil()
	if hasDestLocation && (rg.DestinationCity != "" || rg.DestinationState != "") {
		multiErr.Add(
			"destinationCity",
			errortypes.ErrInvalid,
			"A destination location cannot be combined with city or state criteria",
		)
	}

	if len(rg.Entries) == 0 {
		multiErr.Add("entries", errortypes.ErrRequired, "At least one carrier entry is required")
	}

	ranks := make(map[int16]struct{}, len(rg.Entries))
	for idx, entry := range rg.Entries {
		if entry == nil {
			continue
		}
		entryErr := multiErr.WithIndex("entries", idx)
		entry.Validate(entryErr)
		if _, dup := ranks[entry.Rank]; dup {
			entryErr.Add("rank", errortypes.ErrInvalid, "Rank must be unique within the guide")
		}
		ranks[entry.Rank] = struct{}{}
	}
}

func (rg *RoutingGuide) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "rgd",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "origin_city",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "destination_city",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "origin_state",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightC,
			},
			{
				Name:   "destination_state",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightC,
			},
		},
	}
}

func (rg *RoutingGuide) GetID() pulid.ID {
	return rg.ID
}

func (rg *RoutingGuide) GetCreatedAt() int64 {
	return rg.CreatedAt
}

func (rg *RoutingGuide) GetTableName() string {
	return "routing_guides"
}

func (rg *RoutingGuide) GetOrganizationID() pulid.ID {
	return rg.OrganizationID
}

func (rg *RoutingGuide) GetBusinessUnitID() pulid.ID {
	return rg.BusinessUnitID
}

func (rg *RoutingGuide) GetVersion() int64 {
	return rg.Version
}

func (rg *RoutingGuide) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if rg.ID.IsNil() {
			rg.ID = pulid.MustNew("rg_")
		}
		rg.CreatedAt = now
	case *bun.UpdateQuery:
		rg.UpdatedAt = now
	}

	return nil
}
