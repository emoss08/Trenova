package edi

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	EDIPartnerID   pulid.ID `json:"ediPartnerId"   bun:"edi_partner_id,type:VARCHAR(100),notnull"`
	Name           string   `json:"name"           bun:"name,type:VARCHAR(200),notnull"`
	Description    string   `json:"description"    bun:"description,type:TEXT,nullzero"`
	Version        int64    `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Partner *EDIPartner              `json:"partner,omitempty" bun:"rel:belongs-to,join:edi_partner_id=id"`
	Entries []*EDIMappingProfileItem `json:"entries,omitempty" bun:"rel:has-many,join:id=mapping_profile_id"`
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

	ID               pulid.ID          `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID          `json:"businessUnitId"   bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID   pulid.ID          `json:"organizationId"   bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	EDIPartnerID     pulid.ID          `json:"ediPartnerId"     bun:"edi_partner_id,type:VARCHAR(100),notnull"`
	MappingProfileID pulid.ID          `json:"mappingProfileId" bun:"mapping_profile_id,type:VARCHAR(100),notnull"`
	EntityType       MappingEntityType `json:"entityType"       bun:"entity_type,type:edi_mapping_entity_type_enum,notnull"`
	SourceID         pulid.ID          `json:"sourceId"         bun:"source_id,type:VARCHAR(100),notnull"`
	SourceLabel      string            `json:"sourceLabel"      bun:"source_label,type:VARCHAR(255),nullzero"`
	TargetID         pulid.ID          `json:"targetId"         bun:"target_id,type:VARCHAR(100),notnull"`
	TargetLabel      string            `json:"targetLabel"      bun:"target_label,type:VARCHAR(255),nullzero"`
	CreatedByID      pulid.ID          `json:"createdById"      bun:"created_by_id,type:VARCHAR(100),nullzero"`
	UpdatedByID      pulid.ID          `json:"updatedById"      bun:"updated_by_id,type:VARCHAR(100),nullzero"`
	Version          int64             `json:"version"          bun:"version,type:BIGINT"`
	CreatedAt        int64             `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64             `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Partner *EDIPartner        `json:"partner,omitempty" bun:"rel:belongs-to,join:edi_partner_id=id"`
	Profile *EDIMappingProfile `json:"profile,omitempty" bun:"rel:belongs-to,join:mapping_profile_id=id"`
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
