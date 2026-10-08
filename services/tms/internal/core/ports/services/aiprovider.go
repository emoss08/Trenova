package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/optional"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
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
	ReasoningEffort      aiprovider.ReasoningEffort
	ThinkingStyle        aiprovider.ThinkingStyle
	// ExtraBody carries the vendor request fields this endpoint needs that
	// the protocol does not define.
	ExtraBody            map[string]any
	InputCostPerMillion  *decimal.Decimal
	OutputCostPerMillion *decimal.Decimal
	MaxTokens            int
	// ContextWindow is the model's window in tokens; nil reads it off Model.
	ContextWindow       *int
	Tasks               []aiprovider.Task
	Priority            int
	EmbeddingDimensions *int
	EmbeddingInputStyle aiprovider.EmbeddingInputStyle
	Trusted             bool
	Enabled             bool
	// TimeoutSeconds, MaxConcurrent, MonthlyCapUSD and OnCap are the
	// provider's limits; zero values take the defaults.
	TimeoutSeconds int
	MaxConcurrent  int
	MonthlyCapUSD  *decimal.Decimal
	OnCap          aiprovider.CapAction
	// KeepPreviousKey keeps a replaced key as a fallback for a day, so calls
	// the new key is refused for still go through while it propagates.
	KeepPreviousKey bool
	Version         int64
	TenantInfo      pagination.TenantInfo
}

// PatchAIProviderRequest changes the fields of a provider that are edited on
// their own. An unset field is left alone; a set nil clears a nullable field
// and fails validation for a required one.
type PatchAIProviderRequest struct {
	ID                   pulid.ID
	Version              int64
	TenantInfo           pagination.TenantInfo
	Enabled              optional.Value[*bool]
	Trusted              optional.Value[*bool]
	AllowPrivateNetwork  optional.Value[*bool]
	Tasks                optional.Value[[]aiprovider.Task]
	APIKey               optional.Value[*string]
	InputCostPerMillion  optional.Value[*decimal.Decimal]
	OutputCostPerMillion optional.Value[*decimal.Decimal]
}

// AIProviderEndpoint is an endpoint as an editor holds it, before it is
// saved. With ProviderID and no APIKey, the provider's stored key is used,
// and only for the endpoint it was entered for.
type AIProviderEndpoint struct {
	ProviderID          pulid.ID
	Kind                aiprovider.Kind
	BaseURL             string
	APIKey              *string
	AllowPrivateNetwork bool
}

type ListAIProviderModelsRequest struct {
	TenantInfo pagination.TenantInfo
	Endpoint   AIProviderEndpoint
}

// SealedAIProviderEndpoint is an endpoint in the form it crosses to a
// worker: its key encrypted, so the job's history never holds the secret.
type SealedAIProviderEndpoint struct {
	Kind                aiprovider.Kind `json:"kind"`
	BaseURL             string          `json:"baseUrl"`
	SealedAPIKey        string          `json:"sealedApiKey"`
	AllowPrivateNetwork bool            `json:"allowPrivateNetwork"`
}

// ProbeAIProviderModelsRequest asks a sealed endpoint for its models.
type ProbeAIProviderModelsRequest struct {
	TenantInfo pagination.TenantInfo    `json:"tenantInfo"`
	Endpoint   SealedAIProviderEndpoint `json:"endpoint"`
}

// ProbeAIProviderDraftRequest probes a sealed endpoint with a draft's
// settings.
type ProbeAIProviderDraftRequest struct {
	TenantInfo           pagination.TenantInfo           `json:"tenantInfo"`
	Endpoint             SealedAIProviderEndpoint        `json:"endpoint"`
	Model                string                          `json:"model"`
	StructuredOutputMode aiprovider.StructuredOutputMode `json:"structuredOutputMode"`
	Tasks                []aiprovider.Task               `json:"tasks"`
	EmbeddingDimensions  *int                            `json:"embeddingDimensions"`
	EmbeddingInputStyle  aiprovider.EmbeddingInputStyle  `json:"embeddingInputStyle"`
	TimeoutSeconds       int                             `json:"timeoutSeconds"`
}

// AIProviderModelOption is one model an endpoint says it serves.
type AIProviderModelOption struct {
	ID                   string           `json:"id"`
	DisplayName          string           `json:"displayName"`
	ContextWindow        *int             `json:"contextWindow,omitempty"`
	SizeBytes            *int64           `json:"sizeBytes,omitempty"`
	Loaded               bool             `json:"loaded"`
	Embedding            bool             `json:"embedding"`
	InputCostPerMillion  *decimal.Decimal `json:"inputCostPerMillion,omitempty"`
	OutputCostPerMillion *decimal.Decimal `json:"outputCostPerMillion,omitempty"`
}

// TestAIProviderDraftRequest probes an endpoint that has not been saved.
type TestAIProviderDraftRequest struct {
	TenantInfo           pagination.TenantInfo
	Endpoint             AIProviderEndpoint
	Model                string
	StructuredOutputMode aiprovider.StructuredOutputMode
	Tasks                []aiprovider.Task
	EmbeddingDimensions  *int
	EmbeddingInputStyle  aiprovider.EmbeddingInputStyle
	TimeoutSeconds       *int
}

// AIProviderDraftTestResult is a probe's verdict on an unsaved endpoint.
type AIProviderDraftTestResult = TestAIProviderResult

// AIProviderSpendService reads what providers have spent this UTC calendar
// month, over the calls that carried a price.
type AIProviderSpendService interface {
	MonthSpend(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		providerIDs []pulid.ID,
	) (map[pulid.ID]decimal.Decimal, error)
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
	// Hint is the next step to take when the probe failed, and empty when it
	// worked.
	Hint string `json:"hint,omitempty"`
}

// AIProviderProbe runs a provider's test where the test runs: on a worker, in
// the activity a person's Test request waits on.
type AIProviderProbe interface {
	RunTest(
		ctx context.Context,
		req repositories.GetAIProviderByIDRequest,
	) (*TestAIProviderResult, error)
	RunListModels(
		ctx context.Context,
		req *ProbeAIProviderModelsRequest,
	) ([]AIProviderModelOption, error)
	RunTestDraft(
		ctx context.Context,
		req *ProbeAIProviderDraftRequest,
	) (*AIProviderDraftTestResult, error)
}

// AIProviderTester hands a provider's test to a worker and waits for its
// outcome.
type AIProviderTester interface {
	Test(
		ctx context.Context,
		req repositories.GetAIProviderByIDRequest,
	) (*TestAIProviderResult, error)
	// ListModels asks a sealed endpoint which models it serves.
	ListModels(
		ctx context.Context,
		req *ProbeAIProviderModelsRequest,
	) ([]AIProviderModelOption, error)
	// TestDraft probes a sealed endpoint once and records nothing.
	TestDraft(
		ctx context.Context,
		req *ProbeAIProviderDraftRequest,
	) (*AIProviderDraftTestResult, error)
}

// AIProviderRoutingDraft is a provider as its editor holds it, reduced to
// what decides where tasks go. An ID is a provider being edited; the stored
// provider supplies everything the draft does not set.
type AIProviderRoutingDraft struct {
	ID                  pulid.ID
	Name                string
	Kind                aiprovider.Kind
	Tasks               []aiprovider.Task
	Priority            int
	EmbeddingDimensions *int
	Trusted             bool
	Enabled             bool
}

type AIProviderRoutePreviewRequest struct {
	TenantInfo pagination.TenantInfo
	Draft      AIProviderRoutingDraft
}

type ReorderAIProvidersRequest struct {
	TenantInfo  pagination.TenantInfo
	ProviderIDs []pulid.ID
}

type AssignAIProviderTaskRequest struct {
	TenantInfo pagination.TenantInfo
	ProviderID pulid.ID
	Task       aiprovider.Task
}

type AIProviderService interface {
	List(
		ctx context.Context,
		req *repositories.ListAIProviderRequest,
	) (*pagination.ListResult[*aiprovider.Provider], error)
	ListConnection(
		ctx context.Context,
		req *repositories.ListAIProviderConnectionRequest,
	) (*pagination.CursorListResult[*aiprovider.Provider], error)
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
	Patch(
		ctx context.Context,
		req *PatchAIProviderRequest,
		actor *RequestActor,
	) (*aiprovider.Provider, error)
	Delete(
		ctx context.Context,
		req repositories.DeleteAIProviderRequest,
		actor *RequestActor,
	) error
	Reorder(
		ctx context.Context,
		req *ReorderAIProvidersRequest,
		actor *RequestActor,
	) ([]*aiprovider.Provider, error)
	AssignTask(
		ctx context.Context,
		req *AssignAIProviderTaskRequest,
		actor *RequestActor,
	) (*aiprovider.Provider, error)
	// RoutePreview says where each task goes now and where it would go with
	// the draft saved. It saves nothing.
	RoutePreview(
		ctx context.Context,
		req *AIProviderRoutePreviewRequest,
	) ([]aiprovider.TaskRoute, error)
	// Test issues a live probe against a saved provider and records its outcome
	// on the provider.
	Test(
		ctx context.Context,
		req repositories.GetAIProviderByIDRequest,
	) (*TestAIProviderResult, error)
	// ListModels asks an endpoint which models it serves.
	ListModels(
		ctx context.Context,
		req *ListAIProviderModelsRequest,
	) ([]AIProviderModelOption, error)
	// TestDraft probes an unsaved endpoint once and records nothing.
	TestDraft(
		ctx context.Context,
		req *TestAIProviderDraftRequest,
	) (*AIProviderDraftTestResult, error)
}
