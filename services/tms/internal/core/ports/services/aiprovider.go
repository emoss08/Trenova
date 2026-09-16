package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// SaveAIProviderRequest carries a create or update. APIKey is tri-state: nil
// leaves the stored credential alone, which is what lets an administrator edit a
// provider's routing without re-entering its secret; a pointer to an empty string
// clears it.
type SaveAIProviderRequest struct {
	ID                   pulid.ID
	Name                 string
	Description          string
	Kind                 aiprovider.Kind
	BaseURL              string
	Model                string
	APIKey               *string
	AllowPrivateNetwork  bool
	StructuredOutputMode aiprovider.StructuredOutputMode
	MaxTokens            int
	Tasks                []aiprovider.Task
	Priority             int
	Trusted              bool
	Enabled              bool
	Version              int64
	TenantInfo           pagination.TenantInfo
}

// TestAIProviderResult reports what a live call to the endpoint revealed.
type TestAIProviderResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	// ModelIdentifier is what the endpoint reported serving, which catches the
	// common misconfiguration of a model name the server does not actually have.
	ModelIdentifier string `json:"modelIdentifier,omitempty"`
	// SchemaHonoured reports whether the endpoint returned output matching a
	// probe schema. It is the check worth running: several OpenAI-compatible
	// servers accept a json_schema request and silently ignore it, which cannot
	// be detected any other way.
	SchemaHonoured bool   `json:"schemaHonoured"`
	LatencyMS      int64  `json:"latencyMs"`
	Detail         string `json:"detail,omitempty"`
}

type AIProviderService interface {
	List(
		ctx context.Context,
		req *repositories.ListAIProviderRequest,
	) (*pagination.ListResult[*aiprovider.Provider], error)
	GetByID(
		ctx context.Context,
		req repositories.GetAIProviderByIDRequest,
	) (*aiprovider.Provider, error)
	Create(
		ctx context.Context,
		req *SaveAIProviderRequest,
		actor *RequestActor,
	) (*aiprovider.Provider, error)
	Update(
		ctx context.Context,
		req *SaveAIProviderRequest,
		actor *RequestActor,
	) (*aiprovider.Provider, error)
	Delete(
		ctx context.Context,
		req repositories.DeleteAIProviderRequest,
		actor *RequestActor,
	) error
	// Test issues a live probe against a saved provider.
	Test(
		ctx context.Context,
		req repositories.GetAIProviderByIDRequest,
	) (*TestAIProviderResult, error)
}
