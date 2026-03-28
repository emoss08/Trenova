package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListDocumentTypesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetDocumentTypeByIDRequest struct {
	ID         pulid.ID              `json:"id" form:"id"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type GetDocumentTypeByCodeRequest struct {
	Code       string                `json:"code" form:"code"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type DocumentTypeRepository interface {
	List(
		ctx context.Context,
		req *ListDocumentTypesRequest,
	) (*pagination.ListResult[*documenttype.DocumentType], error)
	GetByID(
		ctx context.Context,
		req GetDocumentTypeByIDRequest,
	) (*documenttype.DocumentType, error)
	GetByCode(
		ctx context.Context,
		req GetDocumentTypeByCodeRequest,
	) (*documenttype.DocumentType, error)
	Create(
		ctx context.Context,
		entity *documenttype.DocumentType,
	) (*documenttype.DocumentType, error)
	Update(
		ctx context.Context,
		entity *documenttype.DocumentType,
	) (*documenttype.DocumentType, error)
	SelectOptions(
		ctx context.Context,
		req *pagination.SelectQueryRequest,
	) (*pagination.ListResult[*documenttype.DocumentType], error)
}
