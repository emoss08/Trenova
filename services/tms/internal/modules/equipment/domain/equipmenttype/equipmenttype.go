package equipmenttype

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	InchesPerFoot       = 12.0
	MaxDeckHeightInches = 120.0
	MaxAxleCount        = int16(20)
)

var (
	_ validationframework.TenantedEntity = (*EquipmentType)(nil)
	_ domaintypes.PostgresSearchable     = (*EquipmentType)(nil)
	_ pagination.CursorEntity            = (*EquipmentType)(nil)
)

// Equipment classification
type EquipmentType struct {
	keel.Table `keel:"equipment_types,report=Equipment"`

	ID             pulid.ID           `json:"id"             keel:"id,pk,autoid:et_"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	Status         domaintypes.Status `json:"status"         keel:"status"`
	Code           string             `json:"code"           keel:"code"`
	Description    string             `json:"description"    keel:"description,nullzero"`
	Class          Class              `json:"class"          keel:"class"`
	Color          string             `json:"color"          keel:"color,nullzero"`
	InteriorLength *float64           `json:"interiorLength" keel:"interior_length"`

	DeckType         DeckType `json:"deckType"         keel:"deck_type,nullzero"`
	DeckHeightInches *float64 `json:"deckHeightInches" keel:"deck_height_inches"`
	WellLengthFeet   *float64 `json:"wellLengthFeet"   keel:"well_length_feet"`
	AxleCount        *int16   `json:"axleCount"        keel:"axle_count,dimension"`
	SearchVector     string   `json:"-"                keel:"search_vector,tsvector,readonly,noselect"`
	Rank             string   `json:"-"                keel:"rank,computed"`
	Version          int64    `json:"version"          keel:"version,version"`
	CreatedAt        int64    `json:"createdAt"        keel:"created_at,autocreate"`
	UpdatedAt        int64    `json:"updatedAt"        keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" keel:"belongs:business_unit_id,noreport"`
	Organization *tenant.Organization `json:"organization,omitempty" keel:"belongs:organization_id,noreport"`
}

func (et *EquipmentType) BeforeCreate(_ context.Context) error {
	if et.Status == "" {
		et.Status = domaintypes.StatusActive
	}

	return nil
}

func (et *EquipmentType) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		et,
		validation.Field(&et.Code, validation.Required),
		validation.Field(
			&et.Code,
			validation.Length(1, 10).Error("Code must be between 1 and 10 characters"),
		),
		validation.Field(
			&et.Class,
			validation.Required,
			domainvalidation.ValidEnum[Class](
				"Class must be one of: Tractor, Trailer, Container, Other",
			),
		),
	))

	et.validateDeck(multiErr)
}

func (et *EquipmentType) validateDeck(multiErr *errortypes.MultiError) {
	if et.DeckType != "" && !et.DeckType.IsValid() {
		multiErr.Add("deckType", errortypes.ErrInvalid,
			"Deck type must be one of: Flatbed, StepDeck, DoubleDrop, RGN, Lowboy, Conestoga")
	}

	if et.DeckHeightInches != nil &&
		(*et.DeckHeightInches <= 0 || *et.DeckHeightInches > MaxDeckHeightInches) {
		multiErr.Add("deckHeightInches", errortypes.ErrInvalid,
			"Deck height must be between 0 and 120 inches")
	}

	if et.WellLengthFeet != nil && *et.WellLengthFeet <= 0 {
		multiErr.Add("wellLengthFeet", errortypes.ErrInvalid,
			"Well length must be greater than zero")
	}

	if et.WellLengthFeet != nil && et.DeckType != "" && !et.DeckType.HasWell() {
		multiErr.Add("wellLengthFeet", errortypes.ErrInvalid,
			"Well length only applies to double drop, RGN, and lowboy decks")
	}

	if et.AxleCount != nil && (*et.AxleCount < 1 || *et.AxleCount > MaxAxleCount) {
		multiErr.Add("axleCount", errortypes.ErrInvalid,
			"Axle count must be between 1 and 20")
	}
}

func (et *EquipmentType) DeckHeightFeet() float64 {
	if et == nil || et.DeckHeightInches == nil {
		return 0
	}

	return *et.DeckHeightInches / InchesPerFoot
}

func (et *EquipmentType) GetID() pulid.ID {
	return et.ID
}

func (et *EquipmentType) GetCreatedAt() int64 {
	return et.CreatedAt
}

func (et *EquipmentType) GetOrganizationID() pulid.ID {
	return et.OrganizationID
}

func (et *EquipmentType) GetBusinessUnitID() pulid.ID {
	return et.BusinessUnitID
}

func (et *EquipmentType) GetTableName() string {
	return "equipment_types"
}

func (et *EquipmentType) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "et",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText},
			{Name: "description", Type: domaintypes.FieldTypeText},
		},
	}
}
