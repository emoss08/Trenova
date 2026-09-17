package carrierintelservice

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	breakerFailureThreshold = 5
	breakerOpenDuration     = 2 * time.Minute
)

type CarrierWriter interface {
	Create(
		ctx context.Context,
		entity *carrier.Carrier,
		actor *services.RequestActor,
	) (*carrier.Carrier, error)
	Update(
		ctx context.Context,
		entity *carrier.Carrier,
		actor *services.RequestActor,
	) (*carrier.Carrier, error)
}

type NotificationSender interface {
	Create(
		ctx context.Context,
		entity *notification.Notification,
	) (*notification.Notification, error)
	ExistsRecent(
		ctx context.Context,
		req repositories.ExistsRecentNotificationRequest,
	) (bool, error)
}

func nowUnix() int64 { return timeutils.NowUnix() }

type boundProvider struct {
	provider   integration.Type
	connector  services.CarrierIntelConnector
	client     services.CarrierIntelClient
	descriptor carrierintel.ProviderDescriptor
	prices     carrierintel.PriceBook
	fallback   bool
	tenant     pagination.TenantInfo
}

func (b *boundProvider) capabilities() carrierintel.CapabilitySet {
	return b.descriptor.Capabilities
}

type breakerState struct {
	failures  int
	openUntil time.Time
}

type circuitBreaker struct {
	mu     sync.Mutex
	states map[string]*breakerState
}

func newCircuitBreaker() *circuitBreaker {
	return &circuitBreaker{states: make(map[string]*breakerState)}
}

func breakerKey(tenantInfo pagination.TenantInfo, provider integration.Type) string {
	return tenantInfo.OrgID.String() + ":" + tenantInfo.BuID.String() + ":" + provider.String()
}

func (b *circuitBreaker) isOpen(key string, now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	state, ok := b.states[key]
	return ok && now.Before(state.openUntil)
}

func (b *circuitBreaker) record(key string, failed bool, now time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	state, ok := b.states[key]
	if !ok {
		state = &breakerState{}
		b.states[key] = state
	}
	if !failed {
		state.failures = 0
		state.openUntil = time.Time{}
		return
	}
	state.failures++
	if state.failures >= breakerFailureThreshold {
		state.openUntil = now.Add(breakerOpenDuration)
	}
}

func (s *Service) bind(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider integration.Type,
	fallback bool,
) (*boundProvider, error) {
	connector, ok := s.connectors[provider]
	if !ok {
		return nil, errortypes.NewBusinessError(
			"{0} is not available as a carrier intelligence provider", provider.String(),
		)
	}

	runtime, err := s.integrations.GetRuntimeConfig(ctx, tenantInfo, provider)
	if err != nil {
		return nil, err
	}

	bound := &boundProvider{
		provider:   provider,
		connector:  connector,
		descriptor: connector.Descriptor(),
		prices:     connector.PriceBook(),
		fallback:   fallback,
		tenant:     tenantInfo,
	}

	client, err := connector.Bind(&services.CarrierIntelBindParams{
		TenantInfo: tenantInfo,
		Config:     runtime.Config,
		Limiter:    s.limiter,
		Recorder:   s.recorderFor(bound),
	})
	if err != nil {
		return nil, errortypes.NewBusinessError(
			"{0} integration is not configured correctly", provider.String(),
		).WithInternal(err)
	}
	bound.client = client
	return bound, nil
}

func (s *Service) resolvePrimary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	control *carrierintel.CarrierIntelControl,
) (*boundProvider, error) {
	provider, ok := control.PrimaryType()
	if !ok {
		return nil, errortypes.NewBusinessError(
			"No carrier intelligence provider is enabled. Enable CarrierOk or FMCSA QCMobile in Integrations",
		)
	}
	return s.bind(ctx, tenantInfo, provider, false)
}

func (s *Service) resolveFallback(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	control *carrierintel.CarrierIntelControl,
) (*boundProvider, bool) {
	provider, ok := control.FallbackType()
	if !ok {
		return nil, false
	}
	bound, err := s.bind(ctx, tenantInfo, provider, true)
	if err != nil {
		s.l.Warn("carrier intelligence fallback provider unavailable",
			zap.String("provider", provider.String()), zap.Error(err))
		return nil, false
	}
	return bound, true
}

func (s *Service) recorderFor(bound *boundProvider) services.CarrierIntelCallRecorder {
	return func(ctx context.Context, call services.CarrierIntelCall) {
		s.recordCall(ctx, bound, call)
	}
}

func (s *Service) recordCall(
	ctx context.Context,
	bound *boundProvider,
	call services.CarrierIntelCall,
) {
	now := s.now()
	price := bound.prices.Price(call.Endpoint)
	purpose, ok := carrierintel.PurposeFromContext(ctx)
	if !ok {
		purpose = carrierintel.PurposeRefresh
	}

	record := &carrierintel.CarrierIntelUsageRecord{
		OrganizationID: bound.tenant.OrgID,
		BusinessUnitID: bound.tenant.BuID,
		Provider:       bound.provider,
		Endpoint:       call.Endpoint,
		Purpose:        purpose,
		DOTNumber:      call.DOTNumber,
		StatusCode:     call.StatusCode,
		Outcome:        usageOutcome(call),
		LatencyMS:      int(call.Latency / time.Millisecond),
		CreatedAt:      now,
	}
	if bound.tenant.UserID.IsNotNil() && purpose.IsInteractive() {
		record.InitiatedByType = string(services.PrincipalTypeUser)
		record.InitiatedByID = bound.tenant.UserID
	}

	if billable, units := isBillable(price, call); billable {
		record.Billable = true
		record.BillableUnits = units
		record.EstimatedCost = price.UnitCost.Mul(decimalFromInt(units))
		record.DedupeKey = carrierintel.BillingDedupeKey(call.Endpoint, price, call.DOTNumber, now)
	}

	inserted, err := s.usageRepo.Insert(context.WithoutCancel(ctx), record)
	if err != nil {
		s.l.Warn("failed to record carrier intelligence usage", zap.Error(err))
		return
	}
	if !inserted && record.Billable {
		record.Billable = false
		record.BillableUnits = 0
		record.EstimatedCost = decimalZero()
		record.DedupeKey = ""
		record.ID = pulid.Nil
		if _, err = s.usageRepo.Insert(context.WithoutCancel(ctx), record); err != nil {
			s.l.Warn("failed to record carrier intelligence usage", zap.Error(err))
		}
	}
	if record.Billable || record.EstimatedCost.IsPositive() {
		invalidateSpendCache(bound.tenant)
	}

	key := breakerKey(bound.tenant, bound.provider)
	failed := call.Err != nil && isOutageError(call.Err)
	s.breaker.record(key, failed, time.Unix(now, 0))
}

func isBillable(price carrierintel.EndpointPrice, call services.CarrierIntelCall) (bool, int) {
	if price.Model == carrierintel.BillingModelFree || !price.UnitCost.IsPositive() {
		return false, 0
	}
	if call.Err != nil && !call.Found {
		return false, 0
	}
	units := max(call.Units, 1)
	switch price.Model {
	case carrierintel.BillingModelPerMatch, carrierintel.BillingModelPerDOTMonth:
		return call.Found, units
	case carrierintel.BillingModelPerRequest:
		return call.Err == nil, units
	default:
		return false, 0
	}
}

func usageOutcome(call services.CarrierIntelCall) carrierintel.UsageOutcome {
	if call.Err == nil {
		if call.Found {
			return carrierintel.UsageOutcomeSuccess
		}
		return carrierintel.UsageOutcomeNotFound
	}
	kind, ok := services.CarrierIntelErrorKindOf(call.Err)
	var limited *restx.RateLimitedError
	switch {
	case errors.As(call.Err, &limited):
		return carrierintel.UsageOutcomeDenied
	case !ok:
		return carrierintel.UsageOutcomeTransport
	case kind == services.CarrierIntelErrorNotFound:
		return carrierintel.UsageOutcomeNotFound
	case kind == services.CarrierIntelErrorRateLimited:
		return carrierintel.UsageOutcomeRateLimited
	case kind == services.CarrierIntelErrorUnavailable:
		return carrierintel.UsageOutcomeServerError
	default:
		return carrierintel.UsageOutcomeClientError
	}
}

func isOutageError(err error) bool {
	var limited *restx.RateLimitedError
	if errors.As(err, &limited) {
		return false
	}
	kind, ok := services.CarrierIntelErrorKindOf(err)
	if !ok {
		return !errors.Is(err, context.Canceled)
	}
	return kind == services.CarrierIntelErrorUnavailable
}

func (s *Service) providerAvailable(bound *boundProvider) bool {
	return !s.breaker.isOpen(breakerKey(bound.tenant, bound.provider), time.Unix(s.now(), 0))
}

func (s *Service) withRateLimitWait(ctx context.Context, fn func() error) error {
	err := fn()
	if err == nil || !carrierintel.IsInteractiveContext(ctx) {
		return err
	}
	wait, limited := retryAfter(err)
	if !limited || wait > s.interactiveWait {
		return err
	}
	timer := time.NewTimer(max(wait, 50*time.Millisecond))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}
	return fn()
}

func retryAfter(err error) (time.Duration, bool) {
	var limited *restx.RateLimitedError
	if errors.As(err, &limited) {
		return limited.RetryAfter, true
	}
	var providerErr *services.CarrierIntelProviderError
	if errors.As(err, &providerErr) && providerErr.Kind == services.CarrierIntelErrorRateLimited {
		return providerErr.RetryAfter, true
	}
	return 0, false
}

func toBusinessError(provider integration.Type, err error) error {
	if err == nil {
		return nil
	}
	var business *errortypes.BusinessError
	if errors.As(err, &business) {
		return err
	}
	if wait, limited := retryAfter(err); limited {
		seconds := max(int(wait/time.Second), 1)
		return errortypes.NewBusinessError(
			"{0} rate limit reached. Try again in {1} seconds", provider.String(), seconds,
		).WithInternal(err)
	}
	kind, _ := services.CarrierIntelErrorKindOf(err)
	switch kind {
	case services.CarrierIntelErrorUnauthorized:
		return errortypes.NewBusinessError(
			"{0} rejected the API key. Update the key in Integrations", provider.String(),
		).WithInternal(err)
	case services.CarrierIntelErrorPaymentRequired:
		return errortypes.NewBusinessError(
			"{0} declined the request because the account's payment failed. Update billing with {0}",
			provider.String(),
		).WithInternal(err)
	case services.CarrierIntelErrorInvalidRequest:
		return errortypes.NewBusinessError(
			"{0} could not process the request", provider.String(),
		).WithInternal(err)
	case services.CarrierIntelErrorUnsupported:
		return errortypes.NewBusinessError(
			"{0} does not support this capability", provider.String(),
		).WithInternal(err)
	default:
		return errortypes.NewBusinessError(
			"{0} is currently unavailable. Try again shortly", provider.String(),
		).WithInternal(err)
	}
}
