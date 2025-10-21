package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type ListDocumentTypeRequest struct {
	Filter *pagination.QueryOptions `json:"filter" form:"filter"`
}

type GetDocumentTypeByIDRequest struct {
	ID     pulid.ID `json:"id"     form:"id"`
	OrgID  pulid.ID `json:"orgId"  form:"orgId"`
	BuID   pulid.ID `json:"buId"   form:"buId"`
	UserID pulid.ID `json:"userId" form:"userId"`
}

type DocumentTypeRepository interface {
	List(
		ctx context.Context,
		req *ListDocumentTypeRequest,
	) (*pagination.ListResult[*documenttype.DocumentType], error)
	GetByID(
		ctx context.Context,
		req GetDocumentTypeByIDRequest,
	) (*documenttype.DocumentType, error)
	Create(
		ctx context.Context,
		dt *documenttype.DocumentType,
	) (*documenttype.DocumentType, error)
	Update(
		ctx context.Context,
		dt *documenttype.DocumentType,
	) (*documenttype.DocumentType, error)
}
