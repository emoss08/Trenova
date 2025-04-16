package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

// IntegrationService provides operations for managing integrations
type IntegrationService interface {
	// Core CRUD operations
	List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*integration.Integration], error)
	GetByID(ctx context.Context, req repositories.GetIntegrationByIDOptions) (*integration.Integration, error)
	GetByType(ctx context.Context, req repositories.GetIntegrationByTypeRequest) (*integration.Integration, error)
	Update(ctx context.Context, i *integration.Integration, userID pulid.ID) (*integration.Integration, error)
}
