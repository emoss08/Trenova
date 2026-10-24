package edi

import (
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var _ domaintypes.PostgresSearchable = (*EDIMappingProfile)(nil)

const maxMappingSourceIDLength = 100

func MappingSourceID(value string) pulid.ID {
	normalized := strings.ToUpper(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
	if len(normalized) > maxMappingSourceIDLength {
		normalized = normalized[:maxMappingSourceIDLength]
	}
	return pulid.ID(normalized)
}

type EDIMappingProfile struct {
	keel.Table `keel:"edi_mapping_profiles"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:edimp_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	EDIPartnerID   pulid.ID `json:"ediPartnerId" keel:"edi_partner_id"`
	Name           string   `json:"name" keel:"name"`
	Description    string   `json:"description" keel:"description,nullzero"`
	SearchVector   string   `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Rank           string   `json:"-" keel:"rank,computed"`
	Version        int64    `json:"version" keel:"version,version"`
	CreatedAt      int64    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt" keel:"updated_at,autoupdate"`

	Partner *EDIPartner              `json:"partner,omitempty" keel:"belongs:edi_partner_id"`
	Entries []*EDIMappingProfileItem `json:"entries,omitempty" keel:"many:mapping_profile_id"`
}

func (p *EDIMappingProfile) GetID() pulid.ID {
	return p.ID
}

func (p *EDIMappingProfile) GetCreatedAt() int64 {
	return p.CreatedAt
}

func (p *EDIMappingProfile) GetTableName() string {
	return "edi_mapping_profiles"
}

func (p *EDIMappingProfile) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "emp",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "description", Type: domaintypes.FieldTypeText},
		},
	}
}

type EDIMappingProfileItem struct {
	keel.Table `keel:"edi_mapping_profile_items"`

	ID               pulid.ID          `json:"id" keel:"id,pk,autoid:edimi_"`
	BusinessUnitID   pulid.ID          `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID   pulid.ID          `json:"organizationId" keel:"organization_id,pk,tenant"`
	EDIPartnerID     pulid.ID          `json:"ediPartnerId" keel:"edi_partner_id"`
	MappingProfileID pulid.ID          `json:"mappingProfileId" keel:"mapping_profile_id"`
	EntityType       MappingEntityType `json:"entityType" keel:"entity_type"`
	SourceID         pulid.ID          `json:"sourceId" keel:"source_id"`
	SourceLabel      string            `json:"sourceLabel" keel:"source_label,nullzero"`
	TargetID         pulid.ID          `json:"targetId" keel:"target_id"`
	TargetLabel      string            `json:"targetLabel" keel:"target_label,nullzero"`
	CreatedByID      pulid.ID          `json:"createdById" keel:"created_by_id,nullzero"`
	UpdatedByID      pulid.ID          `json:"updatedById" keel:"updated_by_id,nullzero"`
	Version          int64             `json:"version" keel:"version,version"`
	CreatedAt        int64             `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt        int64             `json:"updatedAt" keel:"updated_at,autoupdate"`

	Partner *EDIPartner        `json:"partner,omitempty" keel:"belongs:edi_partner_id"`
	Profile *EDIMappingProfile `json:"profile,omitempty" keel:"belongs:mapping_profile_id"`
}

func (p *EDIMappingProfile) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&p.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&p.EDIPartnerID, validation.Required.Error("Partner is required")),
		validation.Field(&p.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxEDINameLength).
				Error("Name cannot be longer than 200 characters"),
		),
	))

	for i, entry := range p.Entries {
		if entry == nil {
			continue
		}
		entry.Validate(multiErr.WithIndex("entries", i))
	}
}

func (i *EDIMappingProfileItem) Validate(multiErr *errortypes.MultiError) {
	// Tenancy and the owning profile are stamped when the parent is written, so
	// they are not the caller's to supply.
	multiErr.AddOzzoError(validation.ValidateStruct(i,
		validation.Field(&i.EntityType,
			validation.Required.Error("Entity type is required"),
			domainvalidation.ValidEnum[MappingEntityType]("Entity type is invalid"),
		),
		// A mapping with either end missing translates nothing, and an inbound
		// document that hits it would fail with no usable explanation.
		validation.Field(&i.SourceID, validation.Required.Error("Source is required")),
		validation.Field(&i.TargetID, validation.Required.Error("Target is required")),
		validation.Field(&i.SourceLabel,
			validation.Length(0, maxMappingLabelLength).
				Error("Source label cannot be longer than 255 characters"),
		),
		validation.Field(&i.TargetLabel,
			validation.Length(0, maxMappingLabelLength).
				Error("Target label cannot be longer than 255 characters"),
		),
	))
}

const maxMappingLabelLength = 255
