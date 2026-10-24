package edi

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	keel.Table `keel:"edi_connections"`

	ID                   pulid.ID                `json:"id" keel:"id,pk,autoid:edic_"`
	BusinessUnitID       pulid.ID                `json:"businessUnitId" keel:"business_unit_id,tenant"`
	SourceOrganizationID pulid.ID                `json:"sourceOrganizationId" keel:"source_organization_id"`
	TargetOrganizationID pulid.ID                `json:"targetOrganizationId" keel:"target_organization_id"`
	SourcePartnerID      pulid.ID                `json:"sourcePartnerId" keel:"source_partner_id,nullzero"`
	TargetPartnerID      pulid.ID                `json:"targetPartnerId" keel:"target_partner_id,nullzero"`
	Method               ConnectionMethod        `json:"method" keel:"method"`
	Status               ConnectionStatus        `json:"status" keel:"status"`
	Capabilities         ConnectionCapabilities  `json:"capabilities" keel:"capabilities,jsonb"`
	SourcePartnerConfig  ConnectionPartnerConfig `json:"sourcePartnerConfig" keel:"source_partner_config,jsonb"`
	TargetPartnerConfig  ConnectionPartnerConfig `json:"targetPartnerConfig" keel:"target_partner_config,jsonb"`
	RequestedByID        pulid.ID                `json:"requestedById" keel:"requested_by_id,nullzero"`
	RequestedAt          int64                   `json:"requestedAt" keel:"requested_at"`
	AcceptedByID         pulid.ID                `json:"acceptedById" keel:"accepted_by_id,nullzero"`
	AcceptedAt           *int64                  `json:"acceptedAt" keel:"accepted_at"`
	RejectedByID         pulid.ID                `json:"rejectedById" keel:"rejected_by_id,nullzero"`
	RejectedAt           *int64                  `json:"rejectedAt" keel:"rejected_at"`
	RejectionReason      string                  `json:"rejectionReason" keel:"rejection_reason,nullzero"`
	SuspendedByID        pulid.ID                `json:"suspendedById" keel:"suspended_by_id,nullzero"`
	SuspendedAt          *int64                  `json:"suspendedAt" keel:"suspended_at"`
	RevokedByID          pulid.ID                `json:"revokedById" keel:"revoked_by_id,nullzero"`
	RevokedAt            *int64                  `json:"revokedAt" keel:"revoked_at"`
	Version              int64                   `json:"version" keel:"version,version"`
	CreatedAt            int64                   `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt            int64                   `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit       *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	SourceOrganization *tenant.Organization `json:"sourceOrganization,omitempty" keel:"belongs:source_organization_id"`
	TargetOrganization *tenant.Organization `json:"targetOrganization,omitempty" keel:"belongs:target_organization_id"`
	SourcePartner      *EDIPartner          `json:"sourcePartner,omitempty" keel:"belongs:source_partner_id"`
	TargetPartner      *EDIPartner          `json:"targetPartner,omitempty" keel:"belongs:target_partner_id"`
}

func (c *EDIConnection) BeforeUpdate(ctx context.Context) error {
	if err := c.BeforeSave(ctx); err != nil {
		return err
	}

	return nil
}

func (c *EDIConnection) BeforeSave(_ context.Context) error {
	if c.Capabilities == (ConnectionCapabilities{}) {
		c.Capabilities = ConnectionCapabilities{
			LoadTenderOutbound: true,
			LoadTenderInbound:  true,
			ShipmentStatus:     true,
		}
	}
	if c.SourcePartnerConfig.Settings == nil {
		c.SourcePartnerConfig.Settings = map[string]any{}
	}
	if c.TargetPartnerConfig.Settings == nil {
		c.TargetPartnerConfig.Settings = map[string]any{}
	}
	return nil
}

func (c *EDIConnection) BeforeCreate(ctx context.Context) error {
	now := timeutils.NowUnix()
	if err := c.BeforeSave(ctx); err != nil {
		return err
	}
	if c.Status == "" {
		c.Status = ConnectionStatusPendingAcceptance
	}
	if c.RequestedAt == 0 {
		c.RequestedAt = timeutils.NowUnix()
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

func (c *EDIConnection) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		// A connection joins two organizations; without both ends it addresses
		// nobody, and pointing both ends at one organization would have an
		// organization trading with itself.
		validation.Field(&c.SourceOrganizationID,
			validation.Required.Error("Source organization is required"),
		),
		validation.Field(&c.TargetOrganizationID,
			validation.Required.Error("Target organization is required"),
			validation.NotIn(c.SourceOrganizationID).
				Error("A connection cannot join an organization to itself"),
		),
		validation.Field(&c.Method,
			validation.Required.Error("Method is required"),
			domainvalidation.ValidEnum[ConnectionMethod]("Method is invalid"),
		),
		validation.Field(&c.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ConnectionStatus]("Status is invalid"),
		),
		// Each lifecycle transition is an authorization record, so who acted and
		// when travel together or the trail cannot be audited.
		validation.Field(&c.AcceptedAt,
			validation.When(c.AcceptedByID.IsNotNil(), validation.Required.Error(
				"An accepted connection must record when it was accepted",
			)),
		),
		validation.Field(&c.AcceptedByID,
			validation.When(c.AcceptedAt != nil, validation.Required.Error(
				"An accepted connection must record who accepted it",
			)),
		),
		validation.Field(&c.RejectedAt,
			validation.When(c.RejectedByID.IsNotNil(), validation.Required.Error(
				"A rejected connection must record when it was rejected",
			)),
		),
		validation.Field(&c.RejectionReason,
			validation.When(
				c.Status == ConnectionStatusRejected,
				validation.Required.Error("A rejected connection must record why"),
			),
		),
		validation.Field(&c.SuspendedAt,
			validation.When(c.SuspendedByID.IsNotNil(), validation.Required.Error(
				"A suspended connection must record when it was suspended",
			)),
		),
		validation.Field(&c.RevokedAt,
			validation.When(c.RevokedByID.IsNotNil(), validation.Required.Error(
				"A revoked connection must record when it was revoked",
			)),
		),
	))
}
