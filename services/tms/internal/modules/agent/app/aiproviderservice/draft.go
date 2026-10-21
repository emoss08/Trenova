package aiproviderservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) ListModels(
	ctx context.Context,
	req *agentports.ListAIProviderModelsRequest,
) ([]agentports.AIProviderModelOption, error) {
	sealed, err := s.seal(ctx, req.TenantInfo, &req.Endpoint)
	if err != nil {
		return nil, err
	}

	return s.tester.ListModels(ctx, &agentports.ProbeAIProviderModelsRequest{
		TenantInfo: req.TenantInfo,
		Endpoint:   sealed,
	})
}

func (s *Service) TestDraft(
	ctx context.Context,
	req *agentports.TestAIProviderDraftRequest,
) (*agentports.AIProviderDraftTestResult, error) {
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

	return s.tester.TestDraft(ctx, &agentports.ProbeAIProviderDraftRequest{
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
	req *agentports.ProbeAIProviderModelsRequest,
) ([]agentports.AIProviderModelOption, error) {
	provider := endpointProvider(&req.Endpoint)
	apiKey, err := s.unseal(req.TenantInfo, &req.Endpoint)
	if err != nil {
		return nil, err
	}

	return s.prober.ListModels(ctx, provider, apiKey)
}

func (s *Service) RunTestDraft(
	ctx context.Context,
	req *agentports.ProbeAIProviderDraftRequest,
) (*agentports.AIProviderDraftTestResult, error) {
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

	apiKey, err := s.unseal(req.TenantInfo, &req.Endpoint)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, provider.ResolvedTimeout())
	defer cancel()

	return s.prober.Probe(ctx, provider, apiKey), nil
}

func validateDraft(req *agentports.TestAIProviderDraftRequest, multiErr *errortypes.MultiError) {
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
		multiErr.Add(
			"structuredOutputMode",
			errortypes.ErrInvalid,
			"Unknown structured output mode",
		)
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
	endpoint *agentports.AIProviderEndpoint,
) (agentports.SealedAIProviderEndpoint, error) {
	if err := s.checkPrivateNetwork(endpoint.AllowPrivateNetwork); err != nil {
		return agentports.SealedAIProviderEndpoint{}, err
	}

	sealed := agentports.SealedAIProviderEndpoint{
		Kind:                endpoint.Kind,
		Platform:            endpoint.Platform,
		PlatformSettings:    clonePlatformSettings(endpoint.PlatformSettings),
		BaseURL:             strings.TrimSpace(endpoint.BaseURL),
		AllowPrivateNetwork: endpoint.AllowPrivateNetwork,
	}
	draft := endpointProvider(&sealed)
	draft.NormalizePlatformSettings()
	sealed.Platform = draft.Platform
	sealed.PlatformSettings = draft.PlatformSettings
	if err := s.checkServerCredentials(draft); err != nil {
		return agentports.SealedAIProviderEndpoint{}, err
	}

	key, err := s.sealedKey(ctx, tenantInfo, endpoint, draft)
	if err != nil {
		return agentports.SealedAIProviderEndpoint{}, err
	}
	sealed.SealedAPIKey = key

	draft.APIKey = key
	multiErr := errortypes.NewMultiError()
	draft.ValidateEndpoint(multiErr)
	if multiErr.HasErrors() {
		return agentports.SealedAIProviderEndpoint{}, multiErr
	}

	return sealed, nil
}

func (s *Service) sealedKey(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	endpoint *agentports.AIProviderEndpoint,
	draft *aiprovider.Provider,
) (string, error) {
	if endpoint.APIKey != nil {
		if typed := strings.TrimSpace(*endpoint.APIKey); typed != "" {
			multiErr := errortypes.NewMultiError()
			draft.ValidateCredential(typed, multiErr)
			if multiErr.HasErrors() {
				return "", multiErr
			}
			return s.sealForDraft(tenantInfo, typed)
		}
	}
	if endpoint.ProviderID.IsNil() {
		return "", nil
	}

	stored, err := s.repo.GetByID(ctx, agentports.GetAIProviderByIDRequest{
		ID:         endpoint.ProviderID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return "", err
	}
	if !aiprovider.SameCredentialTarget(stored, draft) || !stored.HasStoredAPIKey() {
		return "", nil
	}

	apiKey, err := s.decryptAPIKey(ctx, stored)
	if err != nil {
		return "", err
	}

	return s.sealForDraft(tenantInfo, apiKey)
}

func (s *Service) sealForDraft(tenantInfo pagination.TenantInfo, apiKey string) (string, error) {
	encrypted, err := s.encryption.EncryptStringWithAAD(apiKey, draftAAD(tenantInfo))
	if err != nil {
		return "", errortypes.NewBusinessError(
			"failed to encrypt the credential for this AI provider",
		).WithInternal(err)
	}

	return encrypted, nil
}

func draftAAD(tenantInfo pagination.TenantInfo) encryptionservice.AAD {
	return encryptionservice.AIProviderDraftCredentialAAD(
		tenantInfo.OrgID,
		tenantInfo.BuID,
		tenantInfo.UserID,
	)
}

func (s *Service) unseal(
	tenantInfo pagination.TenantInfo,
	endpoint *agentports.SealedAIProviderEndpoint,
) (string, error) {
	if strings.TrimSpace(endpoint.SealedAPIKey) == "" {
		return "", nil
	}

	apiKey, _, err := s.encryption.DecryptStringWithLegacy(
		endpoint.SealedAPIKey,
		draftAAD(tenantInfo),
	)
	if err != nil {
		return "", errortypes.NewBusinessError(
			"failed to decrypt the credential for this AI provider",
		).WithInternal(err)
	}

	return apiKey, nil
}

func endpointProvider(endpoint *agentports.SealedAIProviderEndpoint) *aiprovider.Provider {
	return &aiprovider.Provider{
		Kind:                endpoint.Kind,
		Platform:            endpoint.Platform,
		PlatformSettings:    clonePlatformSettings(endpoint.PlatformSettings),
		BaseURL:             endpoint.BaseURL,
		AllowPrivateNetwork: endpoint.AllowPrivateNetwork,
		TimeoutSeconds:      aiprovider.DefaultTimeoutSeconds,
	}
}
