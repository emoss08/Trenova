package aiproviderservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.AIProviderRepository
	Encryption   *encryptionservice.Service
	Prober       *Prober
	AuditService services.AuditService
	// Tester runs a test on a worker, which calls back into RunTest.
	Tester services.AIProviderTester
	Config *config.Config
}

// EndpointProber issues a live call against a configured endpoint.
type EndpointProber interface {
	Probe(
		ctx context.Context,
		provider *aiprovider.Provider,
		apiKey string,
	) *services.TestAIProviderResult
}

type Service struct {
	l          *zap.Logger
	repo       repositories.AIProviderRepository
	encryption *encryptionservice.Service
	prober     EndpointProber
	audit      services.AuditService
	tester     services.AIProviderTester
	ai         *config.AIConfig
}

var (
	_ services.AIProviderService = (*Service)(nil)
	_ services.AIProviderProbe   = (*Service)(nil)
)

func New(p Params) *Service {
	var ai *config.AIConfig
	if p.Config != nil {
		ai = p.Config.GetAIConfig()
	}

	return &Service{
		l:          p.Logger.Named("service.aiprovider"),
		repo:       p.Repo,
		encryption: p.Encryption,
		prober:     p.Prober,
		audit:      p.AuditService,
		tester:     p.Tester,
		ai:         ai,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListAIProviderRequest,
) (*pagination.ListResult[*aiprovider.Provider], error) {
	result, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	for idx, provider := range result.Items {
		result.Items[idx] = provider.Redacted()
	}

	return result, nil
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListAIProviderConnectionRequest,
) (*pagination.CursorListResult[*aiprovider.Provider], error) {
	result, err := s.repo.ListConnection(ctx, req)
	if err != nil {
		return nil, err
	}

	for i, provider := range result.Items {
		result.Items[i] = provider.Redacted()
	}

	return result, nil
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetAIProviderByIDRequest,
) (*aiprovider.Provider, error) {
	provider, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	return provider.Redacted(), nil
}

func (s *Service) Create(
	ctx context.Context,
	req *services.SaveAIProviderRequest,
	actor *services.RequestActor,
) (*aiprovider.Provider, error) {
	provider := &aiprovider.Provider{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}
	if err := s.apply(provider, req); err != nil {
		return nil, err
	}

	multiErr := errortypes.NewMultiError()
	provider.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.repo.Create(ctx, provider)
	if err != nil {
		return nil, err
	}

	s.logAudit(&auditParams{
		provider:  created,
		previous:  nil,
		operation: permission.OpCreate,
		actor:     actor,
		comment:   "AI provider created",
	})

	return created.Redacted(), nil
}

func (s *Service) Update(
	ctx context.Context,
	req *services.SaveAIProviderRequest,
	actor *services.RequestActor,
) (*aiprovider.Provider, error) {
	existing, err := s.repo.GetByID(ctx, repositories.GetAIProviderByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	previous := existing.Redacted()

	updated := *existing
	updated.Version = req.Version
	if err = s.apply(&updated, req); err != nil {
		return nil, err
	}

	multiErr := errortypes.NewMultiError()
	if keptKeyForNewEndpoint(existing, &updated, req) {
		// The stored key belongs to the endpoint it was entered for. Kept
		// across a change of address, the next test or call would send it,
		// decrypted, to wherever the address now points.
		multiErr.Add("apiKey", errortypes.ErrRequired,
			"Enter the API key again: the endpoint changed, and the saved key is not sent to a new one")
	}
	updated.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	saved, err := s.repo.Update(ctx, &updated)
	if err != nil {
		return nil, err
	}

	s.logAudit(&auditParams{
		provider:  saved,
		previous:  previous,
		operation: permission.OpUpdate,
		actor:     actor,
		comment:   "AI provider updated",
	})

	return saved.Redacted(), nil
}

func (s *Service) Delete(
	ctx context.Context,
	req repositories.DeleteAIProviderRequest,
	actor *services.RequestActor,
) error {
	existing, err := s.repo.GetByID(ctx, repositories.GetAIProviderByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		return err
	}

	s.logAudit(&auditParams{
		provider:  existing,
		previous:  existing.Redacted(),
		operation: permission.OpDelete,
		actor:     actor,
		comment:   "AI provider deleted",
	})

	return nil
}

// Test probes a saved provider and records the outcome on it. The probe runs
// on a worker; the provider is read here first so one that does not exist is
// reported as such rather than as a failed test.
func (s *Service) Test(
	ctx context.Context,
	req repositories.GetAIProviderByIDRequest,
) (*services.TestAIProviderResult, error) {
	if _, err := s.repo.GetByID(ctx, req); err != nil {
		return nil, err
	}

	return s.tester.Test(ctx, req)
}

// RunTest issues the live probe and records its outcome. It is what the
// worker runs for Test.
func (s *Service) RunTest(
	ctx context.Context,
	req repositories.GetAIProviderByIDRequest,
) (*services.TestAIProviderResult, error) {
	provider, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	apiKey, err := s.decryptAPIKey(provider)
	if err != nil {
		return nil, err
	}

	result := s.prober.Probe(ctx, provider, apiKey)

	if err = s.repo.MarkTested(ctx, repositories.MarkAIProviderTestedRequest{
		ID:         provider.ID,
		TenantInfo: req.TenantInfo,
		Outcome: &aiprovider.TestOutcome{
			Success:         result.Success,
			Message:         result.Message,
			ModelIdentifier: result.ModelIdentifier,
			SchemaHonoured:  result.SchemaHonoured,
			LatencyMS:       result.LatencyMS,
			Detail:          result.Detail,
			TestedAt:        timeutils.NowUnix(),
		},
	}); err != nil {
		s.l.Error("failed to record ai provider test outcome",
			zap.String("providerID", provider.ID.String()),
			zap.Error(err))
	}

	return result, nil
}

// apply copies a save request onto an entity, encrypting the credential and
// defaulting the fields an administrator can reasonably leave blank.
func (s *Service) apply(
	provider *aiprovider.Provider,
	req *services.SaveAIProviderRequest,
) error {
	provider.Name = strings.TrimSpace(req.Name)
	provider.Description = strings.TrimSpace(req.Description)
	provider.Kind = req.Kind
	provider.BaseURL = strings.TrimSpace(req.BaseURL)
	provider.Model = strings.TrimSpace(req.Model)
	if req.AllowPrivateNetwork && !s.ai.PrivateNetworkProvidersAllowed() {
		return errortypes.NewValidationError(
			"allowPrivateNetwork", errortypes.ErrForbidden,
			"This server does not allow providers on private network addresses",
		)
	}
	provider.AllowPrivateNetwork = req.AllowPrivateNetwork
	provider.MaxTokens = req.MaxTokens
	provider.Tasks = req.Tasks
	provider.Priority = req.Priority
	provider.Trusted = req.Trusted
	provider.Enabled = req.Enabled

	provider.StructuredOutputMode = req.StructuredOutputMode
	if provider.StructuredOutputMode == "" {
		provider.StructuredOutputMode = req.Kind.DefaultStructuredOutputMode()
	}
	provider.ReasoningEffort = req.ReasoningEffort
	if provider.ReasoningEffort == "" {
		provider.ReasoningEffort = aiprovider.ReasoningOff
	}
	provider.ExtraBody = req.ExtraBody
	provider.InputCostPerMillion = req.InputCostPerMillion
	provider.OutputCostPerMillion = req.OutputCostPerMillion

	// A nil key means "leave what is stored alone", so an administrator can
	// retask a provider without re-entering its secret.
	if req.APIKey == nil {
		return nil
	}

	incoming := strings.TrimSpace(*req.APIKey)
	if incoming == "" {
		provider.APIKey = ""
		return nil
	}

	encrypted, err := s.encryption.EncryptString(incoming)
	if err != nil {
		return errortypes.NewBusinessError(
			"failed to encrypt the credential for this AI provider",
		).WithInternal(err)
	}
	provider.APIKey = encrypted

	return nil
}

// keptKeyForNewEndpoint reports a save that moves a provider to a different
// server, or a different kind of API, while leaving its stored key in place.
func keptKeyForNewEndpoint(
	existing, updated *aiprovider.Provider,
	req *services.SaveAIProviderRequest,
) bool {
	if req.APIKey != nil || !existing.HasStoredAPIKey() {
		return false
	}

	return existing.Kind != updated.Kind || !httpsafe.SameOrigin(existing.BaseURL, updated.BaseURL)
}

func (s *Service) decryptAPIKey(provider *aiprovider.Provider) (string, error) {
	if !provider.HasStoredAPIKey() {
		return "", nil
	}

	decrypted, err := s.encryption.DecryptString(provider.APIKey)
	if err != nil {
		return "", errortypes.NewBusinessError(
			"failed to decrypt the credential for AI provider {0}", provider.Name,
		).WithInternal(err)
	}

	return decrypted, nil
}

type auditParams struct {
	provider  *aiprovider.Provider
	previous  *aiprovider.Provider
	operation permission.Operation
	actor     *services.RequestActor
	comment   string
}

func (s *Service) logAudit(p *auditParams) {
	auditActor := p.actor.AuditActor()

	// The redacted form is what is recorded: an audit trail that carried the
	// encrypted credential would spread the secret into a second store.
	var previousState map[string]any
	if p.previous != nil {
		previousState = jsonutils.MustToJSON(p.previous)
	}

	if err := s.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceAIProvider,
		ResourceID:     p.provider.GetID().String(),
		Operation:      p.operation,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(p.provider.Redacted()),
		PreviousState:  previousState,
		OrganizationID: p.provider.OrganizationID,
		BusinessUnitID: p.provider.BusinessUnitID,
	}, auditservice.WithComment(p.comment)); err != nil {
		s.l.Error("failed to log ai provider audit", zap.Error(err))
	}
}
