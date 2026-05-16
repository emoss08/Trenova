package edi

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ domaintypes.PostgresSearchable = (*EDIConnection)(nil)

type ConnectionCapabilities struct {
	LoadTenderOutbound bool `json:"loadTenderOutbound"`
	LoadTenderInbound  bool `json:"loadTenderInbound"`
	ShipmentStatus     bool `json:"shipmentStatus"`
	Invoice            bool `json:"invoice"`
}

type ConnectionPartnerConfig struct {
	Code               string         `json:"code"`
	Name               string         `json:"name"`
	Description        string         `json:"description"`
	ContactName        string         `json:"contactName"`
	ContactEmail       string         `json:"contactEmail"`
	ContactPhone       string         `json:"contactPhone"`
	EnabledForInbound  bool           `json:"enabledForInbound"`
	EnabledForOutbound bool           `json:"enabledForOutbound"`
	Settings           map[string]any `json:"settings"`
}

type EDIConnection struct {
	bun.BaseModel `json:"-" bun:"table:edi_connections,alias:ec"`

	ID                   pulid.ID                `json:"id"                       bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID                `json:"businessUnitId"           bun:"business_unit_id,type:VARCHAR(100),notnull"`
	SourceOrganizationID pulid.ID                `json:"sourceOrganizationId"     bun:"source_organization_id,type:VARCHAR(100),notnull"`
	TargetOrganizationID pulid.ID                `json:"targetOrganizationId"     bun:"target_organization_id,type:VARCHAR(100),notnull"`
	SourcePartnerID      pulid.ID                `json:"sourcePartnerId"          bun:"source_partner_id,type:VARCHAR(100),nullzero"`
	TargetPartnerID      pulid.ID                `json:"targetPartnerId"          bun:"target_partner_id,type:VARCHAR(100),nullzero"`
	Method               ConnectionMethod        `json:"method"                   bun:"method,type:edi_connection_method_enum,notnull"`
	Status               ConnectionStatus        `json:"status"                   bun:"status,type:edi_connection_status_enum,notnull,default:'PendingAcceptance'"`
	Capabilities         ConnectionCapabilities  `json:"capabilities"             bun:"capabilities,type:JSONB,notnull,default:'{}'::jsonb"`
	SourcePartnerConfig  ConnectionPartnerConfig `json:"sourcePartnerConfig"      bun:"source_partner_config,type:JSONB,notnull,default:'{}'::jsonb"`
	TargetPartnerConfig  ConnectionPartnerConfig `json:"targetPartnerConfig"      bun:"target_partner_config,type:JSONB,notnull,default:'{}'::jsonb"`
	RequestedByID        pulid.ID                `json:"requestedById"            bun:"requested_by_id,type:VARCHAR(100),nullzero"`
	RequestedAt          int64                   `json:"requestedAt"              bun:"requested_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	AcceptedByID         pulid.ID                `json:"acceptedById"             bun:"accepted_by_id,type:VARCHAR(100),nullzero"`
	AcceptedAt           *int64                  `json:"acceptedAt"               bun:"accepted_at,type:BIGINT,nullzero"`
	RejectedByID         pulid.ID                `json:"rejectedById"             bun:"rejected_by_id,type:VARCHAR(100),nullzero"`
	RejectedAt           *int64                  `json:"rejectedAt"               bun:"rejected_at,type:BIGINT,nullzero"`
	RejectionReason      string                  `json:"rejectionReason"          bun:"rejection_reason,type:TEXT,nullzero"`
	SuspendedByID        pulid.ID                `json:"suspendedById"            bun:"suspended_by_id,type:VARCHAR(100),nullzero"`
	SuspendedAt          *int64                  `json:"suspendedAt"              bun:"suspended_at,type:BIGINT,nullzero"`
	RevokedByID          pulid.ID                `json:"revokedById"              bun:"revoked_by_id,type:VARCHAR(100),nullzero"`
	RevokedAt            *int64                  `json:"revokedAt"                bun:"revoked_at,type:BIGINT,nullzero"`
	Version              int64                   `json:"version"                  bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt            int64                   `json:"createdAt"                bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64                   `json:"updatedAt"                bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit       *tenant.BusinessUnit `json:"businessUnit,omitempty"       bun:"rel:belongs-to,join:business_unit_id=id"`
	SourceOrganization *tenant.Organization `json:"sourceOrganization,omitempty" bun:"rel:belongs-to,join:source_organization_id=id"`
	TargetOrganization *tenant.Organization `json:"targetOrganization,omitempty" bun:"rel:belongs-to,join:target_organization_id=id"`
	SourcePartner      *EDIPartner          `json:"sourcePartner,omitempty"      bun:"rel:belongs-to,join:source_partner_id=id"`
	TargetPartner      *EDIPartner          `json:"targetPartner,omitempty"      bun:"rel:belongs-to,join:target_partner_id=id"`
}

func (c *EDIConnection) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if c.Capabilities == (ConnectionCapabilities{}) {
		c.Capabilities = ConnectionCapabilities{
			LoadTenderOutbound: true,
			LoadTenderInbound:  true,
		}
	}
	if c.SourcePartnerConfig.Settings == nil {
		c.SourcePartnerConfig.Settings = map[string]any{}
	}
	if c.TargetPartnerConfig.Settings == nil {
		c.TargetPartnerConfig.Settings = map[string]any{}
	}

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("edic_")
		}
		c.CreatedAt = now
		if c.RequestedAt == 0 {
			c.RequestedAt = now
		}
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}

func (c *EDIConnection) GetID() pulid.ID {
	return c.ID
}

func (c *EDIConnection) GetOrganizationID() pulid.ID {
	return c.SourceOrganizationID
}

func (c *EDIConnection) GetBusinessUnitID() pulid.ID {
	return c.BusinessUnitID
}

func (c *EDIConnection) GetTableName() string {
	return "edi_connections"
}

func (c *EDIConnection) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "ec",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "method", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightA},
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightB},
		},
	}
}
