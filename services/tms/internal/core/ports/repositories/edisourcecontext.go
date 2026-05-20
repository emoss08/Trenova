package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDISourceContextSchemasRequest struct {
	Filter         *pagination.QueryOptions     `json:"filter"`
	Standard       edi.EDIStandard              `json:"standard"`
	TransactionSet edi.TransactionSet           `json:"transactionSet"`
	Direction      edi.DocumentDirection        `json:"direction"`
	X12Version     string                       `json:"x12Version"`
	ContextKey     string                       `json:"contextKey"`
	SchemaVersion  int64                        `json:"schemaVersion"`
	Status         edi.SourceContextFieldStatus `json:"status"`
}

type GetEDISourceContextSchemaRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetActiveEDISourceContextSchemaRequest struct {
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	Standard       edi.EDIStandard       `json:"standard"`
	TransactionSet edi.TransactionSet    `json:"transactionSet"`
	Direction      edi.DocumentDirection `json:"direction"`
	X12Version     string                `json:"x12Version"`
	ContextKey     string                `json:"contextKey"`
	SchemaVersion  int64                 `json:"schemaVersion"`
}

type ListEDISourceContextFieldsRequest struct {
	Filter         *pagination.QueryOptions     `json:"filter"`
	SchemaID       pulid.ID                     `json:"schemaId"`
	Standard       edi.EDIStandard              `json:"standard"`
	TransactionSet edi.TransactionSet           `json:"transactionSet"`
	Direction      edi.DocumentDirection        `json:"direction"`
	Status         edi.SourceContextFieldStatus `json:"status"`
	SourceKind     edi.SourceContextKind        `json:"sourceKind"`
	Repeated       *bool                        `json:"repeated"`
	PathPrefix     string                       `json:"pathPrefix"`
}

type EDISourceContextRepository interface {
	ListSourceContextSchemas(
		ctx context.Context,
		req *ListEDISourceContextSchemasRequest,
	) (*pagination.ListResult[*edi.EDISourceContextSchema], error)
	GetSourceContextSchema(
		ctx context.Context,
		req GetEDISourceContextSchemaRequest,
	) (*edi.EDISourceContextSchema, error)
	GetActiveSourceContextSchema(
		ctx context.Context,
		req GetActiveEDISourceContextSchemaRequest,
	) (*edi.EDISourceContextSchema, error)
	ListSourceContextFields(
		ctx context.Context,
		req *ListEDISourceContextFieldsRequest,
	) (*pagination.ListResult[*edi.EDISourceContextField], error)
	SearchSourceContextFields(
		ctx context.Context,
		req *ListEDISourceContextFieldsRequest,
	) (*pagination.ListResult[*edi.EDISourceContextField], error)
	SelectSourceContextFieldOptions(
		ctx context.Context,
		req *ListEDISourceContextFieldsRequest,
	) (*pagination.ListResult[*edi.EDISourceContextField], error)
}
