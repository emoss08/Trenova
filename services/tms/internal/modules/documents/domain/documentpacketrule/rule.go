package documentpacketrule

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type PacketItemSummary struct {
	DocumentTypeID        pulid.ID   `json:"documentTypeId"`
	DocumentTypeCode      string     `json:"documentTypeCode"`
	DocumentTypeName      string     `json:"documentTypeName"`
	Required              bool       `json:"required"`
	AllowMultiple         bool       `json:"allowMultiple"`
	DisplayOrder          int        `json:"displayOrder"`
	ExpirationRequired    bool       `json:"expirationRequired"`
	ExpirationWarningDays int        `json:"expirationWarningDays"`
	Status                ItemStatus `json:"status"`
	DocumentCount         int        `json:"documentCount"`
	CurrentDocumentIDs    []pulid.ID `json:"currentDocumentIds"`
}

type PacketSummary struct {
	ResourceID      string              `json:"resourceId"`
	ResourceType    string              `json:"resourceType"`
	Status          PacketStatus        `json:"status"`
	TotalRules      int                 `json:"totalRules"`
	SatisfiedRules  int                 `json:"satisfiedRules"`
	MissingRequired int                 `json:"missingRequired"`
	ExpiringSoon    int                 `json:"expiringSoon"`
	Expired         int                 `json:"expired"`
	NeedsReview     int                 `json:"needsReview"`
	Items           []PacketItemSummary `json:"items"`
}

var (
	_ domaintypes.PostgresSearchable     = (*DocumentPacketRule)(nil)
	_ validationframework.TenantedEntity = (*DocumentPacketRule)(nil)
)

const defaultExpirationWarningDays = 30

type DocumentPacketRule struct {
	keel.Table `keel:"document_packet_rules"`

	ID                    pulid.ID `json:"id"                    keel:"id,pk,autoid:dpr_"`
	OrganizationID        pulid.ID `json:"organizationId"        keel:"organization_id,pk,tenant"`
	BusinessUnitID        pulid.ID `json:"businessUnitId"        keel:"business_unit_id,pk,tenant"`
	ResourceType          string   `json:"resourceType"          keel:"resource_type"`
	DocumentTypeID        pulid.ID `json:"documentTypeId"        keel:"document_type_id"`
	Required              bool     `json:"required"              keel:"required"`
	AllowMultiple         bool     `json:"allowMultiple"         keel:"allow_multiple"`
	DisplayOrder          int      `json:"displayOrder"          keel:"display_order"`
	ExpirationRequired    bool     `json:"expirationRequired"    keel:"expiration_required"`
	ExpirationWarningDays int      `json:"expirationWarningDays" keel:"expiration_warning_days"`
	Version               int64    `json:"version"               keel:"version,version"`
	CreatedAt             int64    `json:"createdAt"             keel:"created_at,autocreate"`
	UpdatedAt             int64    `json:"updatedAt"             keel:"updated_at,autoupdate"`
}

func (r *DocumentPacketRule) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		r,
		validation.Field(
			&r.ResourceType,
			validation.Required.Error("Resource type is required"),
			validation.In("Shipment", "Trailer", "Tractor", "Worker").
				Error("Resource type must be valid"),
		),
		validation.Field(&r.DocumentTypeID, validation.Required.Error("Document type is required")),
		validation.Field(&r.ExpirationWarningDays,
			validation.Min(0).Error("Expiration warning days must be zero or greater"),
		),
	))
}

func (r *DocumentPacketRule) GetID() pulid.ID {
	return r.ID
}

func (r *DocumentPacketRule) GetCreatedAt() int64 {
	return r.CreatedAt
}

func (r *DocumentPacketRule) GetOrganizationID() pulid.ID {
	return r.OrganizationID
}

func (r *DocumentPacketRule) GetBusinessUnitID() pulid.ID {
	return r.BusinessUnitID
}

func (r *DocumentPacketRule) GetTableName() string {
	return "document_packet_rules"
}

func (r *DocumentPacketRule) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "dpr",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "resource_type",
				Type:   domaintypes.FieldTypeEnum,
				Weight: domaintypes.SearchWeightB,
			},
		},
		Relationships: []*domaintypes.RelationshipDefintion{
			{Field: "DocumentType", Queryable: true},
		},
	}
}

func (r *DocumentPacketRule) BeforeCreate(_ context.Context) error {
	r.applyDefaults()
	if r.ExpirationWarningDays == 0 {
		r.ExpirationWarningDays = 30
	}
	r.applyDefaults()

	return nil
}

func (r *DocumentPacketRule) applyDefaults() {
	if r.ExpirationWarningDays == 0 {
		r.ExpirationWarningDays = defaultExpirationWarningDays
	}
}
