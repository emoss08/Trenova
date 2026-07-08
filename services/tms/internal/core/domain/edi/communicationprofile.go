package edi

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ domaintypes.PostgresSearchable = (*EDICommunicationProfile)(nil)

type CommunicationProfileSecretState struct {
	Key      string `json:"key"`
	HasValue bool   `json:"hasValue"`
}

type EDICommunicationProfile struct {
	bun.BaseModel             `json:"-" bun:"table:edi_communication_profiles,alias:ecp"`
	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID                pulid.ID                          `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID                          `json:"businessUnitId"    bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID    pulid.ID                          `json:"organizationId"    bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	EDIConnectionID   pulid.ID                          `json:"ediConnectionId"   bun:"edi_connection_id,type:VARCHAR(100),nullzero"`
	EDIPartnerID      pulid.ID                          `json:"ediPartnerId"      bun:"edi_partner_id,type:VARCHAR(100),nullzero"`
	Method            ConnectionMethod                  `json:"method"            bun:"method,type:edi_connection_method_enum,notnull"`
	Status            domaintypes.Status                `json:"status"            bun:"status,type:status_enum,notnull,default:'Active'"`
	Name              string                            `json:"name"              bun:"name,type:VARCHAR(200),notnull"`
	Description       string                            `json:"description"       bun:"description,type:TEXT,nullzero"`
	Config            map[string]any                    `json:"config"            bun:"config,type:JSONB,notnull,default:'{}'::jsonb"`
	EncryptedSecrets  map[string]string                 `json:"-"                 bun:"encrypted_secrets,type:JSONB,notnull,default:'{}'::jsonb"`
	SecretState       []CommunicationProfileSecretState `json:"secretState"       bun:"-"`
	LastPollAttemptAt *int64                            `json:"lastPollAttemptAt" bun:"last_poll_attempt_at,type:BIGINT,nullzero"`
	LastPollSuccessAt *int64                            `json:"lastPollSuccessAt" bun:"last_poll_success_at,type:BIGINT,nullzero"`
	LastPollError     string                            `json:"lastPollError"     bun:"last_poll_error,type:TEXT,nullzero"`
	SearchVector      string                            `json:"-"                 bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank              string                            `json:"-"                 bun:"rank,type:VARCHAR(100),scanonly"`
	Version           int64                             `json:"version"           bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt         int64                             `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64                             `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Connection   *EDIConnection       `json:"connection,omitempty"   bun:"rel:belongs-to,join:edi_connection_id=id"`
	Partner      *EDIPartner          `json:"partner,omitempty"      bun:"rel:belongs-to,join:edi_partner_id=id"`
}

func (p *EDICommunicationProfile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if p.Config == nil {
		p.Config = map[string]any{}
	}
	if p.EncryptedSecrets == nil {
		p.EncryptedSecrets = map[string]string{}
	}

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("edicp_")
		}
		p.CreatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
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
