package edi

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ domaintypes.PostgresSearchable = (*EDIPartner)(nil)

type EDIPartner struct {
	bun.BaseModel `json:"-" bun:"table:edi_partners,alias:ep"`

	ID                         pulid.ID           `json:"id"                         bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID             pulid.ID           `json:"businessUnitId"             bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID             pulid.ID           `json:"organizationId"             bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Kind                       PartnerKind        `json:"kind"                       bun:"kind,type:edi_partner_kind_enum,notnull,default:'External'"`
	Status                     domaintypes.Status `json:"status"                     bun:"status,type:status_enum,notnull,default:'Active'"`
	Code                       string             `json:"code"                       bun:"code,type:VARCHAR(100),notnull"`
	Name                       string             `json:"name"                       bun:"name,type:VARCHAR(200),notnull"`
	Description                string             `json:"description"                bun:"description,type:TEXT,nullzero"`
	InternalOrganizationID     pulid.ID           `json:"internalOrganizationId"     bun:"internal_organization_id,type:VARCHAR(100),nullzero"`
	CustomerID                 pulid.ID           `json:"customerId"                 bun:"customer_id,type:VARCHAR(100),nullzero"`
	DefaultTransportID         pulid.ID           `json:"defaultTransportId"         bun:"default_transport_id,type:VARCHAR(100),nullzero"`
	DefaultMappingProfileID    pulid.ID           `json:"defaultMappingProfileId"    bun:"default_mapping_profile_id,type:VARCHAR(100),nullzero"`
	DefaultValidationProfileID pulid.ID           `json:"defaultValidationProfileId" bun:"default_validation_profile_id,type:VARCHAR(100),nullzero"`
	Timezone                   string             `json:"timezone"                   bun:"timezone,type:VARCHAR(100),nullzero"`
	Country                    string             `json:"country"                    bun:"country,type:VARCHAR(2),notnull,default:'US'"`
	ContactName                string             `json:"contactName"                bun:"contact_name,type:VARCHAR(150),nullzero"`
	ContactEmail               string             `json:"contactEmail"               bun:"contact_email,type:VARCHAR(255),nullzero"`
	ContactPhone               string             `json:"contactPhone"               bun:"contact_phone,type:VARCHAR(30),nullzero"`
	EnabledForInbound          bool               `json:"enabledForInbound"          bun:"enabled_for_inbound,type:BOOLEAN,notnull,default:true"`
	EnabledForOutbound         bool               `json:"enabledForOutbound"         bun:"enabled_for_outbound,type:BOOLEAN,notnull,default:true"`
	Settings                   map[string]any     `json:"settings"                   bun:"settings,type:JSONB,notnull,default:'{}'::jsonb"`
	SearchVector               string             `json:"-"                          bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                       string             `json:"-"                          bun:"rank,type:VARCHAR(100),scanonly"`
	Version                    int64              `json:"version"                    bun:"version,type:BIGINT"`
	CreatedAt                  int64              `json:"createdAt"                  bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                  int64              `json:"updatedAt"                  bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit         *tenant.BusinessUnit     `json:"businessUnit,omitempty"         bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization         *tenant.Organization     `json:"organization,omitempty"         bun:"rel:belongs-to,join:organization_id=id"`
	InternalOrganization *tenant.Organization     `json:"internalOrganization,omitempty" bun:"rel:belongs-to,join:internal_organization_id=id"`
	Customer             *customer.Customer       `json:"customer,omitempty"             bun:"rel:belongs-to,join:customer_id=id"`
	MappingProfile       *EDIMappingProfile       `json:"mappingProfile,omitempty"       bun:"rel:has-one,join:id=edi_partner_id"`
	MappingEntries       []*EDIMappingProfileItem `json:"mappingEntries,omitempty"       bun:"rel:has-many,join:id=edi_partner_id"`
}

type InternalPartnerPair struct {
	SourcePartner *EDIPartner `json:"sourcePartner"`
	TargetPartner *EDIPartner `json:"targetPartner"`
}

func (p *EDIPartner) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		p,
		validation.Field(&p.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&p.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(
			&p.Kind,
			validation.Required.Error("Kind is required"),
			validation.In(PartnerKindInternal, PartnerKindExternal),
		),
		validation.Field(
			&p.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 100),
		),
		validation.Field(
			&p.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 200),
		),
		validation.Field(
			&p.Country,
			validation.Required.Error("Country is required"),
			validation.Length(2, 2),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if p.Kind == PartnerKindInternal && p.InternalOrganizationID.IsNil() {
		multiErr.Add(
			"internalOrganizationId",
			errortypes.ErrRequired,
			"Internal organization is required for internal EDI partners",
		)
	}
}

func (p *EDIPartner) GetID() pulid.ID {
	return p.ID
}

func (p *EDIPartner) GetTableName() string {
	return "edi_partners"
}

func (p *EDIPartner) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "ep",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
			{Name: "kind", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightC},
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightC},
		},
	}
}

func (p *EDIPartner) GetOrganizationID() pulid.ID {
	return p.OrganizationID
}

func (p *EDIPartner) GetBusinessUnitID() pulid.ID {
	return p.BusinessUnitID
}

func (p *EDIPartner) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if p.Settings == nil {
		p.Settings = map[string]any{}
	}

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("edip_")
		}
		p.CreatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}
