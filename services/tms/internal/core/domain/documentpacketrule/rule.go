package documentpacketrule

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
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
	_ bun.BeforeAppendModelHook          = (*DocumentPacketRule)(nil)
	_ domaintypes.PostgresSearchable     = (*DocumentPacketRule)(nil)
	_ validationframework.TenantedEntity = (*DocumentPacketRule)(nil)
)

type DocumentPacketRule struct {
	bun.BaseModel `bun:"table:document_packet_rules,alias:dpr" json:"-"`

	ID                    pulid.ID `json:"id"                    bun:"id,type:VARCHAR(100),pk,notnull"`
	OrganizationID        pulid.ID `json:"organizationId"        bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID        pulid.ID `json:"businessUnitId"        bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	ResourceType          string   `json:"resourceType"          bun:"resource_type,type:VARCHAR(100),notnull"`
	DocumentTypeID        pulid.ID `json:"documentTypeId"        bun:"document_type_id,type:VARCHAR(100),notnull"`
	Required              bool     `json:"required"              bun:"required,type:BOOLEAN,notnull,default:false"`
	AllowMultiple         bool     `json:"allowMultiple"         bun:"allow_multiple,type:BOOLEAN,notnull,default:false"`
	DisplayOrder          int      `json:"displayOrder"          bun:"display_order,type:INTEGER,notnull,default:0"`
	ExpirationRequired    bool     `json:"expirationRequired"    bun:"expiration_required,type:BOOLEAN,notnull,default:false"`
	ExpirationWarningDays int      `json:"expirationWarningDays" bun:"expiration_warning_days,type:INTEGER,notnull,default:30"`
	Version               int64    `json:"version"               bun:"version,type:BIGINT"`
	CreatedAt             int64    `json:"createdAt"             bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt             int64    `json:"updatedAt"             bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (r *DocumentPacketRule) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
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
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (r *DocumentPacketRule) GetID() pulid.ID {
	return r.ID
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

func (r *DocumentPacketRule) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("dpr_")
		}

		if r.ExpirationWarningDays == 0 {
			r.ExpirationWarningDays = 30
		}

		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
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
			{
				Field:        "DocumentType",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*documenttype.DocumentType)(nil),
				TargetTable:  "document_types",
				ForeignKey:   "document_type_id",
				ReferenceKey: "id",
				Alias:        "dt",
				Queryable:    true,
			},
		},
	}
}
