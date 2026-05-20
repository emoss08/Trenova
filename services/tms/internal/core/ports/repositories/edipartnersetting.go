package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDIPartnerSettingSchemasRequest struct {
	Filter         *pagination.QueryOptions `json:"filter"`
	Standard       edi.EDIStandard          `json:"standard"`
	TransactionSet edi.TransactionSet       `json:"transactionSet"`
	Direction      edi.DocumentDirection    `json:"direction"`
	X12Version     string                   `json:"x12Version"`
	DocumentTypeID pulid.ID                 `json:"documentTypeId"`
	SchemaVersion  int64                    `json:"schemaVersion"`
	Status         edi.PartnerSettingStatus `json:"status"`
}

type GetEDIPartnerSettingSchemaRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetActiveEDIPartnerSettingSchemaRequest struct {
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	DocumentTypeID pulid.ID              `json:"documentTypeId"`
	Standard       edi.EDIStandard       `json:"standard"`
	TransactionSet edi.TransactionSet    `json:"transactionSet"`
	Direction      edi.DocumentDirection `json:"direction"`
	X12Version     string                `json:"x12Version"`
	SchemaVersion  int64                 `json:"schemaVersion"`
}

type ListEDIPartnerSettingFieldsRequest struct {
	Filter         *pagination.QueryOptions `json:"filter"`
	SchemaID       pulid.ID                 `json:"schemaId"`
	Standard       edi.EDIStandard          `json:"standard"`
	TransactionSet edi.TransactionSet       `json:"transactionSet"`
	Direction      edi.DocumentDirection    `json:"direction"`
	Status         edi.PartnerSettingStatus `json:"status"`
	PathPrefix     string                   `json:"pathPrefix"`
	GroupKey       string                   `json:"groupKey"`
	Required       *bool                    `json:"required"`
	Secret         *bool                    `json:"secret"`
}

type EDIPartnerSettingRepository interface {
	ListPartnerSettingSchemas(
		ctx context.Context,
		req *ListEDIPartnerSettingSchemasRequest,
	) (*pagination.ListResult[*edi.EDIPartnerSettingSchema], error)
	GetPartnerSettingSchema(
		ctx context.Context,
		req GetEDIPartnerSettingSchemaRequest,
	) (*edi.EDIPartnerSettingSchema, error)
	GetActivePartnerSettingSchema(
		ctx context.Context,
		req GetActiveEDIPartnerSettingSchemaRequest,
	) (*edi.EDIPartnerSettingSchema, error)
	ListPartnerSettingFields(
		ctx context.Context,
		req *ListEDIPartnerSettingFieldsRequest,
	) (*pagination.ListResult[*edi.EDIPartnerSettingField], error)
	SearchPartnerSettingFields(
		ctx context.Context,
		req *ListEDIPartnerSettingFieldsRequest,
	) (*pagination.ListResult[*edi.EDIPartnerSettingField], error)
	SelectPartnerSettingFieldOptions(
		ctx context.Context,
		req *ListEDIPartnerSettingFieldsRequest,
	) (*pagination.ListResult[*edi.EDIPartnerSettingField], error)
}
