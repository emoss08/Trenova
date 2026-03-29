package documentpacketrule

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Rule)(nil)
	_ validationframework.TenantedEntity = (*Rule)(nil)
)

type ResourceType string

const (
	ResourceTypeShipment ResourceType = "shipment"
	ResourceTypeTrailer  ResourceType = "trailer"
	ResourceTypeTractor  ResourceType = "tractor"
	ResourceTypeWorker   ResourceType = "worker"
)

type Rule struct {
	bun.BaseModel `bun:"table:document_packet_rules,alias:dpr" json:"-"`

	ID                    pulid.ID     `json:"id"                    bun:"id,type:VARCHAR(100),pk,notnull"`
	OrganizationID        pulid.ID     `json:"organizationId"        bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID        pulid.ID     `json:"businessUnitId"        bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	ResourceType          ResourceType `json:"resourceType"          bun:"resource_type,type:VARCHAR(100),notnull"`
	DocumentTypeID        pulid.ID     `json:"documentTypeId"        bun:"document_type_id,type:VARCHAR(100),notnull"`
	Required              bool         `json:"required"              bun:"required,type:BOOLEAN,notnull,default:false"`
	AllowMultiple         bool         `json:"allowMultiple"         bun:"allow_multiple,type:BOOLEAN,notnull,default:false"`
	DisplayOrder          int          `json:"displayOrder"          bun:"display_order,type:INTEGER,notnull,default:0"`
	ExpirationRequired    bool         `json:"expirationRequired"    bun:"expiration_required,type:BOOLEAN,notnull,default:false"`
	ExpirationWarningDays int          `json:"expirationWarningDays" bun:"expiration_warning_days,type:INTEGER,notnull,default:30"`
	Version               int64        `json:"version"               bun:"version,type:BIGINT"`
	CreatedAt             int64        `json:"createdAt"             bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt             int64        `json:"updatedAt"             bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (r *Rule) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		r,
		validation.Field(&r.ResourceType,
			validation.Required.Error("Resource type is required"),
			validation.In(ResourceTypeShipment, ResourceTypeTrailer, ResourceTypeTractor, ResourceTypeWorker).Error("Resource type must be valid"),
		),
		validation.Field(&r.DocumentTypeID, validation.Required.Error("Document type is required")),
		validation.Field(&r.ExpirationWarningDays,
			validation.Min(0).Error("Expiration warning days must be zero or greater"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (r *Rule) GetID() pulid.ID {
	return r.ID
}

func (r *Rule) GetOrganizationID() pulid.ID {
	return r.OrganizationID
}

func (r *Rule) GetBusinessUnitID() pulid.ID {
	return r.BusinessUnitID
}

func (r *Rule) GetTableName() string {
	return "document_packet_rules"
}

func (r *Rule) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("dpr_")
		}
		if r.ExpirationWarningDays == 0 {
			r.ExpirationWarningDays = 30
		}
		r.ResourceType = ResourceType(strings.ToLower(strings.TrimSpace(string(r.ResourceType))))
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.ResourceType = ResourceType(strings.ToLower(strings.TrimSpace(string(r.ResourceType))))
		r.UpdatedAt = now
	}

	return nil
}

type ItemStatus string

const (
	ItemStatusMissing      ItemStatus = "Missing"
	ItemStatusComplete     ItemStatus = "Complete"
	ItemStatusExpiringSoon ItemStatus = "ExpiringSoon"
	ItemStatusExpired      ItemStatus = "Expired"
	ItemStatusNeedsReview  ItemStatus = "NeedsReview"
)

type PacketStatus string

const (
	PacketStatusComplete     PacketStatus = "Complete"
	PacketStatusIncomplete   PacketStatus = "Incomplete"
	PacketStatusExpiringSoon PacketStatus = "ExpiringSoon"
	PacketStatusExpired      PacketStatus = "Expired"
	PacketStatusNeedsReview  PacketStatus = "NeedsReview"
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
	ResourceID       string              `json:"resourceId"`
	ResourceType     ResourceType        `json:"resourceType"`
	Status           PacketStatus        `json:"status"`
	TotalRules       int                 `json:"totalRules"`
	SatisfiedRules   int                 `json:"satisfiedRules"`
	MissingRequired  int                 `json:"missingRequired"`
	ExpiringSoon     int                 `json:"expiringSoon"`
	Expired          int                 `json:"expired"`
	NeedsReview      int                 `json:"needsReview"`
	Items            []PacketItemSummary `json:"items"`
}
