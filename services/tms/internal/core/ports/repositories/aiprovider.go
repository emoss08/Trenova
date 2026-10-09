package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetAIProviderByIDRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListAIProviderRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

// ListAIProvidersForTaskRequest resolves the ordered candidates for one task.
type ListAIProvidersForTaskRequest struct {
	Task       aiprovider.Task
	TenantInfo pagination.TenantInfo
}

// AIProviderSelectOptionsRequest pages the enabled providers that serve a task,
// searched by name and model, for a picker.
type AIProviderSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest
	Task               aiprovider.Task
}

type DeleteAIProviderRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListAIProviderConnectionRequest struct {
	Filter  *pagination.QueryOptions `json:"filter"`
	Cursor  pagination.CursorInfo    `json:"-"`
	Columns []string                 `json:"-"`
}

// MarkAIProviderTestedRequest records the outcome of a live probe without
// touching the provider's configuration or its version.
type MarkAIProviderTestedRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Outcome    *aiprovider.TestOutcome
}

type AIProviderRepository interface {
	List(
		ctx context.Context,
		req *ListAIProviderRequest,
	) (*pagination.ListResult[*aiprovider.Provider], error)
	ListConnection(
		ctx context.Context,
		req *ListAIProviderConnectionRequest,
	) (*pagination.CursorListResult[*aiprovider.Provider], error)
	GetByID(ctx context.Context, req GetAIProviderByIDRequest) (*aiprovider.Provider, error)
	// ListForTask returns the enabled providers assigned to a task, ordered by
	// priority ascending, so the caller can fall through the chain in order.
	ListForTask(
		ctx context.Context,
		req ListAIProvidersForTaskRequest,
	) ([]*aiprovider.Provider, error)
	// ListEnabled lists the enabled providers in the order the router tries
	// them: by priority, then by age.
	ListEnabled(ctx context.Context, tenantInfo pagination.TenantInfo) ([]*aiprovider.Provider, error)
	ListOrdered(ctx context.Context, tenantInfo pagination.TenantInfo) ([]*aiprovider.Provider, error)
	SelectOptions(
		ctx context.Context,
		req *AIProviderSelectOptionsRequest,
	) (*pagination.ListResult[*aiprovider.Provider], error)
	Create(ctx context.Context, entity *aiprovider.Provider) (*aiprovider.Provider, error)
	Update(ctx context.Context, entity *aiprovider.Provider) (*aiprovider.Provider, error)
	Delete(ctx context.Context, req DeleteAIProviderRequest) error
	MarkTested(ctx context.Context, req MarkAIProviderTestedRequest) error
}
