package edi

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
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
	bun.BaseModel `json:"-" bun:"table:edi_mapping_profiles,alias:emp"`
	keel.Table    `keel:"edi_mapping_profiles"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:edimp_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	EDIPartnerID   pulid.ID `json:"ediPartnerId"   bun:"edi_partner_id,type:VARCHAR(100),notnull" keel:"edi_partner_id"`
	Name           string   `json:"name"           bun:"name,type:VARCHAR(200),notnull" keel:"name"`
	Description    string   `json:"description"    bun:"description,type:TEXT,nullzero" keel:"description,nullzero"`
	SearchVector   string   `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly" keel:"search_vector,tsvector,readonly,noselect"`
	Rank           string   `json:"-"              bun:"rank,type:VARCHAR(100),scanonly" keel:"rank,computed"`
	Version        int64    `json:"version"        bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Partner *EDIPartner              `json:"partner,omitempty" bun:"rel:belongs-to,join:edi_partner_id=id" keel:"belongs:edi_partner_id"`
	Entries []*EDIMappingProfileItem `json:"entries,omitempty" bun:"rel:has-many,join:id=mapping_profile_id" keel:"many:mapping_profile_id"`
}

func (p *EDIMappingProfile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("edimp_")
		}
		p.CreatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
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
	bun.BaseModel `json:"-" bun:"table:edi_mapping_profile_items,alias:empi"`
	keel.Table    `keel:"edi_mapping_profile_items"`

	ID               pulid.ID          `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:edimi_"`
	BusinessUnitID   pulid.ID          `json:"businessUnitId"   bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID   pulid.ID          `json:"organizationId"   bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	EDIPartnerID     pulid.ID          `json:"ediPartnerId"     bun:"edi_partner_id,type:VARCHAR(100),notnull" keel:"edi_partner_id"`
	MappingProfileID pulid.ID          `json:"mappingProfileId" bun:"mapping_profile_id,type:VARCHAR(100),notnull" keel:"mapping_profile_id"`
	EntityType       MappingEntityType `json:"entityType"       bun:"entity_type,type:edi_mapping_entity_type_enum,notnull" keel:"entity_type"`
	SourceID         pulid.ID          `json:"sourceId"         bun:"source_id,type:VARCHAR(100),notnull" keel:"source_id"`
	SourceLabel      string            `json:"sourceLabel"      bun:"source_label,type:VARCHAR(255),nullzero" keel:"source_label,nullzero"`
	TargetID         pulid.ID          `json:"targetId"         bun:"target_id,type:VARCHAR(100),notnull" keel:"target_id"`
	TargetLabel      string            `json:"targetLabel"      bun:"target_label,type:VARCHAR(255),nullzero" keel:"target_label,nullzero"`
	CreatedByID      pulid.ID          `json:"createdById"      bun:"created_by_id,type:VARCHAR(100),nullzero" keel:"created_by_id,nullzero"`
	UpdatedByID      pulid.ID          `json:"updatedById"      bun:"updated_by_id,type:VARCHAR(100),nullzero" keel:"updated_by_id,nullzero"`
	Version          int64             `json:"version"          bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt        int64             `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt        int64             `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Partner *EDIPartner        `json:"partner,omitempty" bun:"rel:belongs-to,join:edi_partner_id=id" keel:"belongs:edi_partner_id"`
	Profile *EDIMappingProfile `json:"profile,omitempty" bun:"rel:belongs-to,join:mapping_profile_id=id" keel:"belongs:mapping_profile_id"`
}

func (i *EDIMappingProfileItem) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if i.ID.IsNil() {
			i.ID = pulid.MustNew("edimi_")
		}
		i.CreatedAt = now
	case *bun.UpdateQuery:
		i.UpdatedAt = now
	}

	return nil
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
