package aiproviderservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/httpsafe"
)

func (s *Service) ListModels(
	ctx context.Context,
	req *services.ListAIProviderModelsRequest,
) ([]services.AIProviderModelOption, error) {
	sealed, err := s.seal(ctx, req.TenantInfo, &req.Endpoint)
	if err != nil {
		return nil, err
	}

	return s.tester.ListModels(ctx, &services.ProbeAIProviderModelsRequest{
		TenantInfo: req.TenantInfo,
		Endpoint:   sealed,
	})
}

func (s *Service) TestDraft(
	ctx context.Context,
	req *services.TestAIProviderDraftRequest,
) (*services.AIProviderDraftTestResult, error) {
	multiErr := errortypes.NewMultiError()
	validateDraft(req, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	sealed, err := s.seal(ctx, req.TenantInfo, &req.Endpoint)
	if err != nil {
		return nil, err
	}

	timeout := aiprovider.DefaultTimeoutSeconds
	if req.TimeoutSeconds != nil {
		timeout = *req.TimeoutSeconds
	}

	return s.tester.TestDraft(ctx, &services.ProbeAIProviderDraftRequest{
		TenantInfo:           req.TenantInfo,
		Endpoint:             sealed,
		Model:                strings.TrimSpace(req.Model),
		StructuredOutputMode: req.StructuredOutputMode,
		Tasks:                req.Tasks,
		EmbeddingDimensions:  req.EmbeddingDimensions,
		EmbeddingInputStyle:  req.EmbeddingInputStyle,
		TimeoutSeconds:       timeout,
	})
}

func (s *Service) RunListModels(
	ctx context.Context,
	req *services.ProbeAIProviderModelsRequest,
) ([]services.AIProviderModelOption, error) {
	provider := endpointProvider(&req.Endpoint)
	apiKey, err := s.unseal(&req.Endpoint)
	if err != nil {
		return nil, err
	}

	return s.prober.ListModels(ctx, provider, apiKey)
}

func (s *Service) RunTestDraft(
	ctx context.Context,
	req *services.ProbeAIProviderDraftRequest,
) (*services.AIProviderDraftTestResult, error) {
	provider := endpointProvider(&req.Endpoint)
	provider.Name = "Draft"
	provider.Model = req.Model
	provider.Tasks = req.Tasks
	provider.StructuredOutputMode = req.StructuredOutputMode
	if provider.StructuredOutputMode == "" {
		provider.StructuredOutputMode = provider.Kind.DefaultStructuredOutputMode()
	}
	provider.ReasoningEffort = aiprovider.ReasoningOff
	provider.ThinkingStyle = aiprovider.ThinkingStyleAuto
	provider.EmbeddingDimensions = req.EmbeddingDimensions
	provider.EmbeddingInputStyle = req.EmbeddingInputStyle
	if provider.EmbeddingInputStyle == "" {
		provider.EmbeddingInputStyle = aiprovider.EmbeddingInputStyleNone
	}
	provider.TimeoutSeconds = req.TimeoutSeconds

	apiKey, err := s.unseal(&req.Endpoint)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, provider.ResolvedTimeout())
	defer cancel()

	return s.prober.Probe(ctx, provider, apiKey), nil
}

func validateDraft(req *services.TestAIProviderDraftRequest, multiErr *errortypes.MultiError) {
	if strings.TrimSpace(req.Model) == "" {
		multiErr.Add("model", errortypes.ErrRequired, "Model is required")
	}
	if len(req.Tasks) == 0 {
		multiErr.Add("tasks", errortypes.ErrRequired, "Choose at least one task")
	}
	for _, task := range req.Tasks {
		if !task.IsValid() {
			multiErr.Add("tasks", errortypes.ErrInvalid, "Unknown task")
			break
		}
	}
	if req.StructuredOutputMode != "" && !req.StructuredOutputMode.IsValid() {
		multiErr.Add("structuredOutputMode", errortypes.ErrInvalid, "Unknown structured output mode")
	}
	if req.EmbeddingInputStyle != "" && !req.EmbeddingInputStyle.IsValid() {
		multiErr.Add("embeddingInputStyle", errortypes.ErrInvalid, "Unknown embedding input style")
	}
	if req.TimeoutSeconds != nil &&
		(*req.TimeoutSeconds < aiprovider.MinTimeoutSeconds ||
			*req.TimeoutSeconds > aiprovider.MaxTimeoutSeconds) {
		multiErr.Add("timeoutSeconds", errortypes.ErrInvalid,
			"Timeout must be between 5 and 600 seconds")
	}
}

func (s *Service) seal(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	endpoint *services.AIProviderEndpoint,
) (services.SealedAIProviderEndpoint, error) {
	if err := s.checkPrivateNetwork(endpoint.AllowPrivateNetwork); err != nil {
		return services.SealedAIProviderEndpoint{}, err
	}

	sealed := services.SealedAIProviderEndpoint{
		Kind:                endpoint.Kind,
		BaseURL:             strings.TrimSpace(endpoint.BaseURL),
		AllowPrivateNetwork: endpoint.AllowPrivateNetwork,
	}

	key, err := s.sealedKey(ctx, tenantInfo, endpoint, &sealed)
	if err != nil {
		return services.SealedAIProviderEndpoint{}, err
	}
	sealed.SealedAPIKey = key

	draft := endpointProvider(&sealed)
	draft.APIKey = key
	multiErr := errortypes.NewMultiError()
	draft.ValidateEndpoint(multiErr)
	if multiErr.HasErrors() {
		return services.SealedAIProviderEndpoint{}, multiErr
	}

	return sealed, nil
}

func (s *Service) sealedKey(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	endpoint *services.AIProviderEndpoint,
	sealed *services.SealedAIProviderEndpoint,
) (string, error) {
	if endpoint.APIKey != nil {
		if typed := strings.TrimSpace(*endpoint.APIKey); typed != "" {
			encrypted, err := s.encryption.EncryptString(typed)
			if err != nil {
				return "", errortypes.NewBusinessError(
					"failed to encrypt the credential for this AI provider",
				).WithInternal(err)
			}

			return encrypted, nil
		}
	}
	if endpoint.ProviderID.IsNil() {
		return "", nil
	}

	stored, err := s.repo.GetByID(ctx, repositories.GetAIProviderByIDRequest{
		ID:         endpoint.ProviderID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return "", err
	}
	draft := endpointProvider(sealed)
	if stored.Kind != sealed.Kind ||
		!httpsafe.SameOrigin(stored.ResolvedBaseURL(), draft.ResolvedBaseURL()) {
		return "", nil
	}

	return stored.APIKey, nil
}

func (s *Service) unseal(endpoint *services.SealedAIProviderEndpoint) (string, error) {
	if strings.TrimSpace(endpoint.SealedAPIKey) == "" {
		return "", nil
	}

	apiKey, err := s.encryption.DecryptString(endpoint.SealedAPIKey)
	if err != nil {
		return "", errortypes.NewBusinessError(
			"failed to decrypt the credential for this AI provider",
		).WithInternal(err)
	}

	return apiKey, nil
}

func endpointProvider(endpoint *services.SealedAIProviderEndpoint) *aiprovider.Provider {
	return &aiprovider.Provider{
		Kind:                endpoint.Kind,
		BaseURL:             endpoint.BaseURL,
		AllowPrivateNetwork: endpoint.AllowPrivateNetwork,
		TimeoutSeconds:      aiprovider.DefaultTimeoutSeconds,
	}
}
