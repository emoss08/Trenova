package aiproviderservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiproviderchange"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiproviderkey"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/settingversionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/secretutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	DB           kernelports.Transactor
	Repo         aiproviderports.AIProviderRepository
	Keys         aiproviderports.AIProviderKeyRepository `optional:"true"`
	Versions     settingversionports.SettingVersionRepository
	Encryption   *encryptionservice.Service
	Prober       *Prober
	AuditService auditports.AuditService
	// Tester runs a test on a worker, which calls back into RunTest.
	Tester    aiproviderports.AIProviderTester
	AIConfig  *aiconfig.Config
	Analytics aianalyticsports.AnalyticsCache `optional:"true"`
}

// EndpointProber issues a live call against a configured endpoint.
type EndpointProber interface {
	Probe(
		ctx context.Context,
		provider *aiprovider.Provider,
		apiKey string,
	) *aiproviderports.TestAIProviderResult
	ListModels(
		ctx context.Context,
		provider *aiprovider.Provider,
		apiKey string,
	) ([]aiproviderports.AIProviderModelOption, error)
}

type Service struct {
	l          *zap.Logger
	db         kernelports.Transactor
	repo       aiproviderports.AIProviderRepository
	keys       aiproviderports.AIProviderKeyRepository
	versions   settingversionports.SettingVersionRepository
	encryption *encryptionservice.Service
	prober     EndpointProber
	audit      auditports.AuditService
	tester     aiproviderports.AIProviderTester
	ai         *aiconfig.Config
	analytics  aianalyticsports.AnalyticsCache
}

var (
	_ aiproviderports.AIProviderService = (*Service)(nil)
	_ aiproviderports.AIProviderProbe   = (*Service)(nil)
)

func New(p Params) *Service {
	return &Service{
		l:          p.Logger.Named("service.aiprovider"),
		db:         p.DB,
		repo:       p.Repo,
		keys:       p.Keys,
		versions:   p.Versions,
		encryption: p.Encryption,
		prober:     p.Prober,
		audit:      p.AuditService,
		tester:     p.Tester,
		ai:         p.AIConfig,
		analytics:  p.Analytics,
	}
}

func (s *Service) analyticsChanged(ctx context.Context, tenant pagination.TenantInfo) {
	if s.analytics == nil {
		return
	}
	if err := s.analytics.BumpEpoch(ctx, tenant); err != nil {
		s.l.Warn("failed to invalidate ai analytics after a provider change", zap.Error(err))
	}
}

func (s *Service) List(
	ctx context.Context,
	req *aiproviderports.ListAIProviderRequest,
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
	req *aiproviderports.ListAIProviderConnectionRequest,
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

func (s *Service) SelectOptions(
	ctx context.Context,
	req *aiproviderports.AIProviderSelectOptionsRequest,
) (*pagination.ListResult[*aiprovider.Provider], error) {
	result, err := s.repo.SelectOptions(ctx, req)
	if err != nil {
		return nil, err
	}

	for idx, provider := range result.Items {
		result.Items[idx] = provider.Redacted()
	}

	return result, nil
}

func (s *Service) GetByID(
	ctx context.Context,
	req aiproviderports.GetAIProviderByIDRequest,
) (*aiprovider.Provider, error) {
	provider, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	return provider.Redacted(), nil
}

func (s *Service) Create(
	ctx context.Context,
	req *aiproviderports.SaveAIProviderRequest,
	actor *permissionports.RequestActor,
) (*aiprovider.Provider, error) {
	provider := &aiprovider.Provider{
		ID:             pulid.MustNew("aiprv_"),
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}
	if err := s.apply(provider, req, actor); err != nil {
		return nil, err
	}

	multiErr := errortypes.NewMultiError()
	provider.Validate(multiErr)
	if !multiErr.HasErrors() {
		s.confirmAdvisor(ctx, &advisorCheck{after: provider, req: req}, multiErr)
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.save(ctx, func(txCtx context.Context) (*aiprovider.Provider, error) {
		return s.repo.Create(txCtx, provider)
	}, actor)
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
	s.analyticsChanged(ctx, req.TenantInfo)

	return created.Redacted(), nil
}

func (s *Service) Update(
	ctx context.Context,
	req *aiproviderports.SaveAIProviderRequest,
	actor *permissionports.RequestActor,
) (*aiprovider.Provider, error) {
	existing, err := s.repo.GetByID(ctx, aiproviderports.GetAIProviderByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	previous := existing.Redacted()

	updated := *existing
	updated.Version = req.Version
	if err = s.apply(&updated, req, actor); err != nil {
		return nil, err
	}
	aiproviderkey.ForgetExpiredPreviousKey(&updated, timeutils.NowUnix())

	multiErr := errortypes.NewMultiError()
	if keptKeyForNewEndpoint(existing, &updated, req) {
		// The stored key belongs to the endpoint it was entered for. Kept
		// across a change of address, the next test or call would send it,
		// decrypted, to wherever the address now points.
		multiErr.Add(
			"apiKey",
			errortypes.ErrRequired,
			"Enter the API key again: the endpoint changed, and the saved key is not sent to a new one",
		)
	}
	updated.Validate(multiErr)
	if !multiErr.HasErrors() {
		s.confirmAdvisor(ctx, &advisorCheck{before: existing, after: &updated, req: req}, multiErr)
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	if aiproviderchange.AdvisorSettingsChanged(existing, &updated) {
		updated.AdvisorRefusal = nil
	}

	saved, err := s.save(ctx, func(txCtx context.Context) (*aiprovider.Provider, error) {
		return s.repo.Update(txCtx, &updated)
	}, actor)
	if err != nil {
		return nil, s.explainConflict(ctx, &conflictRequest{
			tenantInfo: req.TenantInfo,
			providerID: req.ID,
			loaded:     req.Version,
			cause:      err,
		})
	}

	s.logAudit(&auditParams{
		provider:  saved,
		previous:  previous,
		operation: permission.OpUpdate,
		actor:     actor,
		comment:   "AI provider updated",
	})
	s.analyticsChanged(ctx, req.TenantInfo)

	return saved.Redacted(), nil
}

func (s *Service) Delete(
	ctx context.Context,
	req aiproviderports.DeleteAIProviderRequest,
	actor *permissionports.RequestActor,
) error {
	existing, err := s.repo.GetByID(ctx, aiproviderports.GetAIProviderByIDRequest(req))
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
	s.analyticsChanged(ctx, req.TenantInfo)

	return nil
}

// Test probes a saved provider and records the outcome on it. The probe runs
// on a worker; the provider is read here first so one that does not exist is
// reported as such rather than as a failed test.
func (s *Service) Test(
	ctx context.Context,
	req aiproviderports.GetAIProviderByIDRequest,
) (*aiproviderports.TestAIProviderResult, error) {
	if _, err := s.repo.GetByID(ctx, req); err != nil {
		return nil, err
	}

	return s.tester.Test(ctx, req)
}

// RunTest issues the live probe and records its outcome. It is what the
// worker runs for Test.
func (s *Service) RunTest(
	ctx context.Context,
	req aiproviderports.GetAIProviderByIDRequest,
) (*aiproviderports.TestAIProviderResult, error) {
	provider, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	apiKey, err := s.decryptAPIKey(ctx, provider)
	if err != nil {
		return nil, err
	}

	s.tidyKey(ctx, provider, apiKey, req.TenantInfo)
	result := s.prober.Probe(ctx, provider, apiKey)

	if err = s.repo.MarkTested(ctx, aiproviderports.MarkAIProviderTestedRequest{
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

// tidyKey records the ends of a key stored before they were kept, and drops
// a replaced key whose day as a fallback is over.
func (s *Service) tidyKey(
	ctx context.Context,
	provider *aiprovider.Provider,
	apiKey string,
	tenantInfo pagination.TenantInfo,
) {
	if s.keys == nil {
		return
	}
	if aiproviderkey.NeedsKeyFingerprint(provider) && apiKey != "" {
		fingerprint := secretutils.FingerprintOf(provider.CredentialFingerprintSubject(apiKey))
		if err := s.keys.RecordFingerprint(ctx, aiproviderports.RecordAIProviderKeyFingerprintRequest{
			ID:         provider.ID,
			TenantInfo: tenantInfo,
			Prefix:     fingerprint.Prefix,
			LastFour:   fingerprint.LastFour,
		}); err != nil {
			s.l.Warn("failed to record ai provider key fingerprint", zap.Error(err))
		}
	}
	if provider.PreviousKeyExpired(timeutils.NowUnix()) {
		if _, err := s.keys.ClearExpiredPrevious(ctx, aiproviderports.ClearExpiredAIProviderKeysRequest{
			TenantInfo: tenantInfo,
			Now:        timeutils.NowUnix(),
		}); err != nil {
			s.l.Warn("failed to clear expired ai provider keys", zap.Error(err))
		}
	}
}

// apply copies a save request onto an entity, encrypting the credential and
// defaulting the fields an administrator can reasonably leave blank.
func (s *Service) apply(
	provider *aiprovider.Provider,
	req *aiproviderports.SaveAIProviderRequest,
	actor *permissionports.RequestActor,
) error {
	before := provider.ConnectionSnapshot()
	provider.Name = strings.TrimSpace(req.Name)
	provider.Description = strings.TrimSpace(req.Description)
	provider.Kind = req.Kind
	provider.BaseURL = strings.TrimSpace(req.BaseURL)
	provider.Platform = req.Platform
	provider.PlatformSettings = clonePlatformSettings(req.PlatformSettings)
	provider.NormalizePlatformSettings()
	if err := s.checkServerCredentials(provider); err != nil {
		return err
	}
	// A replaced key is kept as a fallback only for the place and identity it
	// was entered for; kept across a move, it would be sent somewhere new.
	sameEndpoint := aiproviderchange.SameCredentialTarget(before, provider)
	provider.Model = strings.TrimSpace(req.Model)
	if err := s.checkPrivateNetwork(req.AllowPrivateNetwork); err != nil {
		return err
	}
	provider.AllowPrivateNetwork = req.AllowPrivateNetwork
	provider.MaxTokens = req.MaxTokens
	provider.ContextWindowTokens = req.ContextWindow
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
		provider.ReasoningEffort = aireasoning.ReasoningOff
	}
	provider.ThinkingStyle = req.ThinkingStyle
	if provider.ThinkingStyle == "" {
		provider.ThinkingStyle = aireasoning.ThinkingStyleAuto
	}
	provider.AdvisorEnabled = req.AdvisorEnabled
	provider.AdvisorModel = strings.TrimSpace(req.AdvisorModel)
	provider.Scripts = req.Scripts
	provider.ScriptWrites = req.ScriptWrites
	provider.ExtraBody = req.ExtraBody
	provider.EmbeddingDimensions = req.EmbeddingDimensions
	provider.EmbeddingInputStyle = req.EmbeddingInputStyle
	if provider.EmbeddingInputStyle == "" {
		provider.EmbeddingInputStyle = aiprovider.EmbeddingInputStyleNone
	}
	provider.InputCostPerMillion = req.InputCostPerMillion
	provider.OutputCostPerMillion = req.OutputCostPerMillion
	provider.CacheReadCostPerMillion = req.CacheReadCostPerMillion
	provider.CacheWriteCostPerMillion = req.CacheWriteCostPerMillion
	applyLimits(provider, req)

	// A nil key means "leave what is stored alone", so an administrator can
	// retask a provider without re-entering its secret.
	if req.APIKey == nil {
		return nil
	}

	return s.replaceKey(provider, &keyChange{
		incoming:     *req.APIKey,
		keepPrevious: req.KeepPreviousKey && sameEndpoint,
		actor:        actor,
		provider:     provider,
	})
}

func clonePlatformSettings(settings *aiplatform.PlatformSettings) *aiplatform.PlatformSettings {
	if settings == nil {
		return nil
	}
	clone := *settings

	return &clone
}

// applyLimits copies the limits, taking the default for one left at zero.
func applyLimits(provider *aiprovider.Provider, req *aiproviderports.SaveAIProviderRequest) {
	provider.TimeoutSeconds = req.TimeoutSeconds
	if provider.TimeoutSeconds == 0 {
		provider.TimeoutSeconds = aiprovider.DefaultTimeoutSeconds
	}
	provider.MaxConcurrent = req.MaxConcurrent
	if provider.MaxConcurrent == 0 {
		provider.MaxConcurrent = aiprovider.DefaultMaxConcurrent
	}
	provider.MonthlyCapUSD = req.MonthlyCapUSD
	provider.OnCap = req.OnCap
	if provider.OnCap == "" {
		provider.OnCap = aiprovider.CapActionNext
	}
}

func (s *Service) checkPrivateNetwork(allow bool) error {
	if allow && !s.ai.PrivateNetworkProvidersAllowed() {
		return errortypes.NewValidationError(
			"allowPrivateNetwork", errortypes.ErrForbidden,
			"This server does not allow providers on private network addresses",
		)
	}

	return nil
}

func (s *Service) checkServerCredentials(provider *aiprovider.Provider) error {
	if provider.UsesServerCredentials() && !s.ai.ServerCloudCredentialsAllowed() {
		return errortypes.NewValidationError(
			"platformSettings.auth", errortypes.ErrForbidden,
			"This server does not let providers sign in with its own cloud credentials",
		)
	}

	return nil
}

type keyChange struct {
	incoming     string
	keepPrevious bool
	actor        *permissionports.RequestActor
	// provider is the provider the key is for, already on its platform, so
	// the key is checked against the way that platform signs in.
	provider *aiprovider.Provider
}

// replaceKey stores a new credential, or clears it when the new one is
// blank.
func (s *Service) replaceKey(provider *aiprovider.Provider, change *keyChange) error {
	replacement, err := s.keyReplacement(change)
	if err != nil {
		return err
	}
	aiproviderkey.ReplaceAPIKey(provider, replacement)

	return nil
}

// keyReplacement encrypts a new credential. Only its ends are kept in the
// clear, to tell two keys apart.
func (s *Service) keyReplacement(change *keyChange) (aiproviderkey.KeyReplacement, error) {
	incoming := strings.TrimSpace(change.incoming)
	replacement := aiproviderkey.KeyReplacement{
		AddedByID:    change.actor.PersonUserID(),
		Now:          timeutils.NowUnix(),
		KeepPrevious: change.keepPrevious,
	}
	if incoming == "" {
		return replacement, nil
	}

	subject := incoming
	if change.provider != nil {
		multiErr := errortypes.NewMultiError()
		change.provider.ValidateCredential(incoming, multiErr)
		if multiErr.HasErrors() {
			return replacement, multiErr
		}
		subject = change.provider.CredentialFingerprintSubject(incoming)
	}

	encrypted, err := s.encryption.EncryptStringWithAAD(incoming, credentialAAD(change.provider))
	if err != nil {
		return replacement, errortypes.NewBusinessError(
			"failed to encrypt the credential for this AI provider",
		).WithInternal(err)
	}
	fingerprint := secretutils.FingerprintOf(subject)
	replacement.Encrypted = encrypted
	replacement.Prefix = fingerprint.Prefix
	replacement.LastFour = fingerprint.LastFour

	return replacement, nil
}

// keptKeyForNewEndpoint reports a save that moves a provider to a different
// server, kind of API, platform, region or identity while leaving its stored
// key in place.
func keptKeyForNewEndpoint(
	existing, updated *aiprovider.Provider,
	req *aiproviderports.SaveAIProviderRequest,
) bool {
	if req.APIKey != nil || !existing.HasStoredAPIKey() {
		return false
	}

	return !aiproviderchange.SameCredentialTarget(existing, updated)
}

func credentialAAD(provider *aiprovider.Provider) encryptionservice.AAD {
	return encryptionservice.AIProviderCredentialAAD(
		provider.OrganizationID,
		provider.BusinessUnitID,
		provider.ID,
	)
}

func (s *Service) decryptAPIKey(
	ctx context.Context,
	provider *aiprovider.Provider,
) (string, error) {
	if !provider.HasStoredAPIKey() {
		return "", nil
	}

	decrypted, rebound, err := s.encryption.DecryptAndRebind(
		provider.APIKey,
		credentialAAD(provider),
	)
	if err != nil {
		return "", errortypes.NewBusinessError(
			"failed to decrypt the credential for AI provider {0}", provider.Name,
		).WithInternal(err)
	}
	s.persistRebound(ctx, provider, false, provider.APIKey, rebound)

	return decrypted, nil
}

func (s *Service) persistRebound(
	ctx context.Context,
	provider *aiprovider.Provider,
	previous bool,
	expected, rebound string,
) {
	if rebound == "" || s.keys == nil {
		return
	}
	if err := s.keys.RebindSealedKey(ctx, &aiproviderports.RebindAIProviderKeyRequest{
		ID: provider.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: provider.OrganizationID,
			BuID:  provider.BusinessUnitID,
		},
		Previous:    previous,
		Expected:    expected,
		Replacement: rebound,
	}); err != nil {
		s.l.Warn("failed to bind an ai provider key to its provider", zap.Error(err))
	}
}

type auditParams struct {
	provider  *aiprovider.Provider
	previous  *aiprovider.Provider
	operation permission.Operation
	actor     *permissionports.RequestActor
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

	if err := s.audit.LogAction(&auditports.LogActionParams{
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
