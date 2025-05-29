package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billing"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetDocumentTypeByIDRequest struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type DocumentTypeRepository interface {
	List(
		ctx context.Context,
		opts *ports.LimitOffsetQueryOptions,
	) (*ports.ListResult[*billing.DocumentType], error)
	GetByID(ctx context.Context, opts GetDocumentTypeByIDRequest) (*billing.DocumentType, error)
	GetByIDs(ctx context.Context, docIDs []string) ([]*billing.DocumentType, error)
	Create(ctx context.Context, dt *billing.DocumentType) (*billing.DocumentType, error)
	Update(ctx context.Context, dt *billing.DocumentType) (*billing.DocumentType, error)
}
