package edi

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var _ domaintypes.PostgresSearchable = (*EDICommunicationProfile)(nil)

type CommunicationProfileSecretState struct {
	Key      string `json:"key"`
	HasValue bool   `json:"hasValue"`
}

type EDICommunicationProfile struct {
	keel.Table `keel:"edi_communication_profiles"`

	ID                pulid.ID                          `json:"id" keel:"id,pk,autoid:edicp_"`
	BusinessUnitID    pulid.ID                          `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID                          `json:"organizationId" keel:"organization_id,pk,tenant"`
	EDIConnectionID   pulid.ID                          `json:"ediConnectionId" keel:"edi_connection_id,nullzero"`
	EDIPartnerID      pulid.ID                          `json:"ediPartnerId" keel:"edi_partner_id,nullzero"`
	Method            ConnectionMethod                  `json:"method" keel:"method"`
	Status            domaintypes.Status                `json:"status" keel:"status"`
	Name              string                            `json:"name" keel:"name"`
	Description       string                            `json:"description" keel:"description,nullzero"`
	Config            map[string]any                    `json:"config" keel:"config,jsonb"`
	EncryptedSecrets  map[string]string                 `json:"-" keel:"encrypted_secrets,jsonb"`
	SecretState       []CommunicationProfileSecretState `json:"secretState"`
	LastPollAttemptAt *int64                            `json:"lastPollAttemptAt" keel:"last_poll_attempt_at"`
	LastPollSuccessAt *int64                            `json:"lastPollSuccessAt" keel:"last_poll_success_at"`
	LastPollError     string                            `json:"lastPollError" keel:"last_poll_error,nullzero"`
	SearchVector      string                            `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Rank              string                            `json:"-" keel:"rank,computed"`
	Version           int64                             `json:"version" keel:"version,version"`
	CreatedAt         int64                             `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt         int64                             `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization *tenant.Organization `json:"organization,omitempty"`
	Connection   *EDIConnection       `json:"connection,omitempty" keel:"belongs:edi_connection_id"`
	Partner      *EDIPartner          `json:"partner,omitempty" keel:"belongs:edi_partner_id"`
}

func (p *EDICommunicationProfile) BeforeUpdate(ctx context.Context) error {
	if err := p.BeforeSave(ctx); err != nil {
		return err
	}

	return nil
}

func (p *EDICommunicationProfile) BeforeCreate(ctx context.Context) error {
	if err := p.BeforeSave(ctx); err != nil {
		return err
	}
	if p.Status == "" {
		p.Status = domaintypes.StatusActive
	}

	return nil
}

func (p *EDICommunicationProfile) BeforeSave(_ context.Context) error {
	if p.Config == nil {
		p.Config = map[string]any{}
	}
	if p.EncryptedSecrets == nil {
		p.EncryptedSecrets = map[string]string{}
	}
	return nil
}

func (p *EDICommunicationProfile) GetID() pulid.ID {
	return p.ID
}

func (p *EDICommunicationProfile) GetOrganizationID() pulid.ID {
	return p.OrganizationID
}

func (p *EDICommunicationProfile) GetBusinessUnitID() pulid.ID {
	return p.BusinessUnitID
}

func (p *EDICommunicationProfile) GetTableName() string {
	return "edi_communication_profiles"
}

func (p *EDICommunicationProfile) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "ecp",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{Name: "method", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightC},
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightC},
		},
	}
}

func (p *EDICommunicationProfile) GetCreatedAt() int64 {
	return p.CreatedAt
}

func (p *EDICommunicationProfile) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&p.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&p.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxEDINameLength).
				Error("Name cannot be longer than 200 characters"),
		),
		validation.Field(&p.Method,
			validation.Required.Error("Method is required"),
			domainvalidation.ValidEnum[ConnectionMethod]("Method is invalid"),
		),
		validation.Field(&p.Status,
			validation.Required.Error("Status is required"),
			validation.In(domaintypes.StatusActive, domaintypes.StatusInactive).
				Error("Status is invalid"),
		),
		// A profile that names neither a connection nor a partner has nothing to
		// poll on behalf of.
		validation.Field(&p.EDIPartnerID,
			validation.When(p.EDIConnectionID.IsNil(), validation.Required.Error(
				"A communication profile must name a connection or a partner",
			)),
		),
	))
}
