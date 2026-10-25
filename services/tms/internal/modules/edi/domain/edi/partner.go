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
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ domaintypes.PostgresSearchable = (*EDIPartner)(nil)
	_ pagination.CursorEntity        = (*EDIPartner)(nil)
)

type EDIPartner struct {
	keel.Table `keel:"edi_partners"`

	ID                      pulid.ID           `json:"id"                      keel:"id,pk,autoid:edip_"`
	BusinessUnitID          pulid.ID           `json:"businessUnitId"          keel:"business_unit_id,pk,tenant"`
	OrganizationID          pulid.ID           `json:"organizationId"          keel:"organization_id,pk,tenant"`
	Kind                    PartnerKind        `json:"kind"                    keel:"kind"`
	Status                  domaintypes.Status `json:"status"                  keel:"status"`
	Code                    string             `json:"code"                    keel:"code"`
	Name                    string             `json:"name"                    keel:"name"`
	Description             string             `json:"description"             keel:"description,nullzero"`
	InternalOrganizationID  pulid.ID           `json:"internalOrganizationId"  keel:"internal_organization_id,nullzero"`
	EDIConnectionID         pulid.ID           `json:"ediConnectionId"         keel:"edi_connection_id,nullzero"`
	CustomerID              pulid.ID           `json:"customerId"              keel:"customer_id,nullzero"`
	DefaultTransportID      pulid.ID           `json:"defaultTransportId"      keel:"default_transport_id,nullzero"`
	DefaultMappingProfileID pulid.ID           `json:"defaultMappingProfileId" keel:"default_mapping_profile_id,nullzero"`
	Timezone                string             `json:"timezone"                keel:"timezone,nullzero"`
	Country                 string             `json:"country"                 keel:"country"`
	ContactName             string             `json:"contactName"             keel:"contact_name,nullzero"`
	ContactEmail            string             `json:"contactEmail"            keel:"contact_email,nullzero"`
	ContactPhone            string             `json:"contactPhone"            keel:"contact_phone,nullzero"`
	EnabledForInbound       bool               `json:"enabledForInbound"       keel:"enabled_for_inbound"`
	EnabledForOutbound      bool               `json:"enabledForOutbound"      keel:"enabled_for_outbound"`
	Settings                map[string]any     `json:"settings"                keel:"settings,jsonb"`
	SearchVector            string             `json:"-"                       keel:"search_vector,tsvector,readonly,noselect"`
	Rank                    string             `json:"-"                       keel:"rank,computed"`
	Version                 int64              `json:"version"                 keel:"version,version"`
	CreatedAt               int64              `json:"createdAt"               keel:"created_at,autocreate"`
	UpdatedAt               int64              `json:"updatedAt"               keel:"updated_at,autoupdate"`

	BusinessUnit         *tenant.BusinessUnit     `json:"businessUnit,omitempty"`
	Organization         *tenant.Organization     `json:"organization,omitempty"`
	InternalOrganization *tenant.Organization     `json:"internalOrganization,omitempty" keel:"belongs:internal_organization_id"`
	Customer             *customer.Customer       `json:"customer,omitempty"             keel:"belongs:customer_id"`
	Connection           *EDIConnection           `json:"connection,omitempty"           keel:"belongs:edi_connection_id"`
	DefaultTransport     *EDICommunicationProfile `json:"defaultTransport,omitempty"     keel:"belongs:default_transport_id"`
	MappingProfile       *EDIMappingProfile       `json:"mappingProfile,omitempty"       keel:"one:edi_partner_id"`
	MappingEntries       []*EDIMappingProfileItem `json:"mappingEntries,omitempty"       keel:"many:edi_partner_id"`
}

func (p *EDIPartner) BeforeUpdate(ctx context.Context) error {
	if err := p.BeforeSave(ctx); err != nil {
		return err
	}

	return nil
}

func (p *EDIPartner) BeforeCreate(ctx context.Context) error {
	if err := p.BeforeSave(ctx); err != nil {
		return err
	}
	if p.Kind == "" {
		p.Kind = PartnerKindExternal
	}
	if p.Status == "" {
		p.Status = domaintypes.StatusActive
	}
	if p.Country == "" {
		p.Country = "US"
	}

	return nil
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

func (p *EDIPartner) GetCreatedAt() int64 {
	return p.CreatedAt
}

func (EDIPartner) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "idx_edi_partners_code_org",
			Message: "EDI partner with this code already exists",
		},
		{
			Name:    "idx_edi_partners_name_org",
			Message: "EDI partner with this name already exists",
		},
		{
			Name:    "idx_edi_partners_internal_relationship_org_bu",
			Message: "An internal EDI partner already exists for this target organization",
		},
	}
}
