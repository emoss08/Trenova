package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetIntegrationByIDOptions struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type GetIntegrationByTypeRequest struct {
	Type  integration.Type
	OrgID pulid.ID
	BuID  pulid.ID
}

type IntegrationRepository interface {
	// Standard CRUD operations
	List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*integration.Integration], error)
	GetByID(ctx context.Context, opts GetIntegrationByIDOptions) (*integration.Integration, error)
	GetByType(ctx context.Context, req GetIntegrationByTypeRequest) (*integration.Integration, error)
	Update(ctx context.Context, i *integration.Integration) (*integration.Integration, error)

	// Additional methods for integration metrics
	RecordUsage(ctx context.Context, id, orgID, buID pulid.ID) error
	RecordError(ctx context.Context, id, orgID, buID pulid.ID, errorMessage string) error
	ClearError(ctx context.Context, id, orgID, buID pulid.ID) error
}
