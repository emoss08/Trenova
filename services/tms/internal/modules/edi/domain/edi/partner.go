package edi

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*EDIPartner)(nil)
	_ domaintypes.PostgresSearchable = (*EDIPartner)(nil)
	_ pagination.CursorEntity        = (*EDIPartner)(nil)
)

type EDIPartner struct {
	bun.BaseModel `json:"-" bun:"table:edi_partners,alias:ep"`
	keel.Table    `keel:"edi_partners"`

	ID                      pulid.ID           `json:"id"                      bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:edip_"`
	BusinessUnitID          pulid.ID           `json:"businessUnitId"          bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID          pulid.ID           `json:"organizationId"          bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	Kind                    PartnerKind        `json:"kind"                    bun:"kind,type:edi_partner_kind_enum,notnull,default:'External'" keel:"kind"`
	Status                  domaintypes.Status `json:"status"                  bun:"status,type:status_enum,notnull,default:'Active'" keel:"status"`
	Code                    string             `json:"code"                    bun:"code,type:VARCHAR(100),notnull" keel:"code"`
	Name                    string             `json:"name"                    bun:"name,type:VARCHAR(200),notnull" keel:"name"`
	Description             string             `json:"description"             bun:"description,type:TEXT,nullzero" keel:"description,nullzero"`
	InternalOrganizationID  pulid.ID           `json:"internalOrganizationId"  bun:"internal_organization_id,type:VARCHAR(100),nullzero" keel:"internal_organization_id,nullzero"`
	EDIConnectionID         pulid.ID           `json:"ediConnectionId"         bun:"edi_connection_id,type:VARCHAR(100),nullzero" keel:"edi_connection_id,nullzero"`
	CustomerID              pulid.ID           `json:"customerId"              bun:"customer_id,type:VARCHAR(100),nullzero" keel:"customer_id,nullzero"`
	DefaultTransportID      pulid.ID           `json:"defaultTransportId"      bun:"default_transport_id,type:VARCHAR(100),nullzero" keel:"default_transport_id,nullzero"`
	DefaultMappingProfileID pulid.ID           `json:"defaultMappingProfileId" bun:"default_mapping_profile_id,type:VARCHAR(100),nullzero" keel:"default_mapping_profile_id,nullzero"`
	Timezone                string             `json:"timezone"                bun:"timezone,type:VARCHAR(100),nullzero" keel:"timezone,nullzero"`
	Country                 string             `json:"country"                 bun:"country,type:VARCHAR(2),notnull,default:'US'" keel:"country"`
	ContactName             string             `json:"contactName"             bun:"contact_name,type:VARCHAR(150),nullzero" keel:"contact_name,nullzero"`
	ContactEmail            string             `json:"contactEmail"            bun:"contact_email,type:VARCHAR(255),nullzero" keel:"contact_email,nullzero"`
	ContactPhone            string             `json:"contactPhone"            bun:"contact_phone,type:VARCHAR(30),nullzero" keel:"contact_phone,nullzero"`
	EnabledForInbound       bool               `json:"enabledForInbound"       bun:"enabled_for_inbound,type:BOOLEAN,notnull" keel:"enabled_for_inbound"`
	EnabledForOutbound      bool               `json:"enabledForOutbound"      bun:"enabled_for_outbound,type:BOOLEAN,notnull" keel:"enabled_for_outbound"`
	Settings                map[string]any     `json:"settings"                bun:"settings,type:JSONB,notnull,default:'{}'" keel:"settings,jsonb"`
	SearchVector            string             `json:"-"                       bun:"search_vector,type:TSVECTOR,scanonly" keel:"search_vector,tsvector,readonly,noselect"`
	Rank                    string             `json:"-"                       bun:"rank,type:VARCHAR(100),scanonly" keel:"rank,computed"`
	Version                 int64              `json:"version"                 bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt               int64              `json:"createdAt"               bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt               int64              `json:"updatedAt"               bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	BusinessUnit         *tenant.BusinessUnit     `json:"businessUnit,omitempty"         bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization         *tenant.Organization     `json:"organization,omitempty"         bun:"rel:belongs-to,join:organization_id=id"`
	InternalOrganization *tenant.Organization     `json:"internalOrganization,omitempty" bun:"rel:belongs-to,join:internal_organization_id=id" keel:"belongs:internal_organization_id"`
	Customer             *customer.Customer       `json:"customer,omitempty"             bun:"rel:belongs-to,join:customer_id=id" keel:"belongs:customer_id"`
	Connection           *EDIConnection           `json:"connection,omitempty"           bun:"rel:belongs-to,join:edi_connection_id=id" keel:"belongs:edi_connection_id"`
	DefaultTransport     *EDICommunicationProfile `json:"defaultTransport,omitempty"     bun:"rel:belongs-to,join:default_transport_id=id" keel:"belongs:default_transport_id"`
	MappingProfile       *EDIMappingProfile       `json:"mappingProfile,omitempty"       bun:"rel:has-one,join:id=edi_partner_id" keel:"one:edi_partner_id"`
	MappingEntries       []*EDIMappingProfileItem `json:"mappingEntries,omitempty"       bun:"rel:has-many,join:id=edi_partner_id" keel:"many:edi_partner_id"`
}

type InternalPartnerPair struct {
	SourcePartner *EDIPartner `json:"sourcePartner"`
	TargetPartner *EDIPartner `json:"targetPartner"`
}

func (p *EDIPartner) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
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
	))

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

func (p *EDIPartner) BeforeSave(_ context.Context) error {
	if p.Settings == nil {
		p.Settings = map[string]any{}
	}
	return nil
}

func (p *EDIPartner) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if err := p.BeforeSave(ctx); err != nil {
		return err
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

func (p *EDIPartner) GetCreatedAt() int64 {
	return p.CreatedAt
}
