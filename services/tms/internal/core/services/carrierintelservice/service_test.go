package carrierintelservice

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testNow = int64(1_789_000_000)

type fakeUsageRepo struct {
	repositories.CarrierIntelUsageRepository

	mu         sync.Mutex
	records    []*carrierintel.CarrierIntelUsageRecord
	dedupeKeys map[string]bool
	cost       decimal.Decimal
	billable   int
	costCalls  int
}

func newFakeUsageRepo() *fakeUsageRepo {
	return &fakeUsageRepo{dedupeKeys: make(map[string]bool)}
}

func (f *fakeUsageRepo) Insert(
	_ context.Context,
	entity *carrierintel.CarrierIntelUsageRecord,
) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if entity.DedupeKey != "" {
		if f.dedupeKeys[entity.DedupeKey] {
			return false, nil
		}
		f.dedupeKeys[entity.DedupeKey] = true
	}
	copied := *entity
	f.records = append(f.records, &copied)
	return true, nil
}

func (f *fakeUsageRepo) ExistsDedupeKey(
	_ context.Context,
	_ pagination.TenantInfo,
	_ integration.Type,
	dedupeKey string,
) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.dedupeKeys[dedupeKey], nil
}

func (f *fakeUsageRepo) CostSince(
	_ context.Context,
	_ pagination.TenantInfo,
	_ int64,
) (decimal.Decimal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.costCalls++
	return f.cost, nil
}

func (f *fakeUsageRepo) CountBillableSince(
	_ context.Context,
	_ *repositories.CountBillableUsageRequest,
) (int, error) {
	return f.billable, nil
}

type fakeNotifications struct {
	mu      sync.Mutex
	created []*notification.Notification
}

func (f *fakeNotifications) Create(
	_ context.Context,
	entity *notification.Notification,
) (*notification.Notification, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, existing := range f.created {
		if existing.CorrelationID != nil && entity.CorrelationID != nil &&
			*existing.CorrelationID == *entity.CorrelationID {
			return entity, nil
		}
	}
	f.created = append(f.created, entity)
	return entity, nil
}

func (f *fakeNotifications) ExistsRecent(
	_ context.Context,
	req repositories.ExistsRecentNotificationRequest,
) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, existing := range f.created {
		if existing.EventType == req.EventType && existing.CorrelationID != nil &&
			*existing.CorrelationID == req.CorrelationID {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeNotifications) eventTypes() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.created))
	for _, entity := range f.created {
		out = append(out, entity.EventType)
	}
	return out
}

type fakeIntegrations struct {
	failures map[integration.Type]error
}

func (f *fakeIntegrations) GetRuntimeConfig(
	_ context.Context,
	_ pagination.TenantInfo,
	typ integration.Type,
) (*integrationservice.RuntimeConfig, error) {
	if err := f.failures[typ]; err != nil {
		return nil, err
	}
	return &integrationservice.RuntimeConfig{
		Enabled:    true,
		Configured: true,
		Ready:      true,
		Config:     map[string]string{"apiKey": "sk_test_key"},
	}, nil
}

type fakeClient struct {
	provider integration.Type
	recorder services.CarrierIntelCallRecorder

	mu       sync.Mutex
	requests []*services.CarrierIntelLookupRequest
	errs     []error
	result   *services.CarrierIntelLookupResult
}

func (c *fakeClient) Provider() integration.Type { return c.provider }

func (c *fakeClient) IsSandbox() bool { return true }

func (c *fakeClient) Lookup(
	ctx context.Context,
	req *services.CarrierIntelLookupRequest,
) (*services.CarrierIntelLookupResult, error) {
	c.mu.Lock()
	c.requests = append(c.requests, req)
	var err error
	if len(c.errs) > 0 {
		err = c.errs[0]
		c.errs = c.errs[1:]
	}
	c.mu.Unlock()

	if c.recorder != nil {
		c.recorder(ctx, services.CarrierIntelCall{
			Endpoint:  depthEndpoint(req.Depth),
			DOTNumber: req.Identifier.DOTNumber,
			Found:     err == nil,
			Err:       err,
		})
	}
	if err != nil {
		return nil, err
	}
	if c.result != nil {
		return c.result, nil
	}
	return &services.CarrierIntelLookupResult{
		Profile:  &carrierintel.Profile{},
		Endpoint: depthEndpoint(req.Depth),
	}, nil
}

func (c *fakeClient) lookups() []*services.CarrierIntelLookupRequest {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]*services.CarrierIntelLookupRequest(nil), c.requests...)
}

type fakeConnector struct {
	typ    integration.Type
	caps   carrierintel.CapabilitySet
	prices carrierintel.PriceBook
	client *fakeClient
}

func (c *fakeConnector) IntegrationType() integration.Type { return c.typ }

func (c *fakeConnector) Descriptor() carrierintel.ProviderDescriptor {
	return carrierintel.ProviderDescriptor{Capabilities: c.caps}
}

func (c *fakeConnector) PriceBook() carrierintel.PriceBook { return c.prices }

func (c *fakeConnector) TestConnection(context.Context, map[string]string) error { return nil }

func (c *fakeConnector) Bind(
	params *services.CarrierIntelBindParams,
) (services.CarrierIntelClient, error) {
	c.client.recorder = params.Recorder
	return c.client, nil
}

func carrierOKPrices() carrierintel.PriceBook {
	return carrierintel.PriceBook{
		carrierintel.EndpointProfileFull: {
			Model:    carrierintel.BillingModelPerDOTMonth,
			UnitCost: decimal.RequireFromString("3"),
		},
		carrierintel.EndpointProfileLite: {
			Model:    carrierintel.BillingModelPerDOTMonth,
			UnitCost: decimal.RequireFromString("0.5"),
		},
		carrierintel.EndpointProfileFMCSA: {
			Model:    carrierintel.BillingModelPerMatch,
			UnitCost: decimal.RequireFromString("0.1"),
		},
	}
}

func newCarrierOKConnector() *fakeConnector {
	return &fakeConnector{
		typ: integration.TypeCarrierOK,
		caps: carrierintel.NewCapabilitySet(
			carrierintel.CapabilityLookupFull,
			carrierintel.CapabilityLookupLite,
			carrierintel.CapabilityLookupFMCSA,
			carrierintel.CapabilityNativeMonitoring,
		),
		prices: carrierOKPrices(),
		client: &fakeClient{provider: integration.TypeCarrierOK},
	}
}

func newFMCSAConnector() *fakeConnector {
	return &fakeConnector{
		typ:    integration.TypeFMCSAQCMobile,
		caps:   carrierintel.NewCapabilitySet(carrierintel.CapabilityLookupFMCSA),
		prices: carrierintel.PriceBook{},
		client: &fakeClient{provider: integration.TypeFMCSAQCMobile},
	}
}

type harness struct {
	svc           *Service
	usage         *fakeUsageRepo
	notifications *fakeNotifications
	integrations  *fakeIntegrations
	tenant        pagination.TenantInfo
	control       *carrierintel.CarrierIntelControl
	now           int64
}

func newHarness(t *testing.T, connectors ...*fakeConnector) *harness {
	t.Helper()

	tenant := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	h := &harness{
		usage:         newFakeUsageRepo(),
		notifications: &fakeNotifications{},
		integrations:  &fakeIntegrations{failures: make(map[integration.Type]error)},
		tenant:        tenant,
		control:       carrierintel.NewDefaultControl(tenant.OrgID, tenant.BuID),
		now:           testNow,
	}

	byType := make(map[integration.Type]services.CarrierIntelConnector, len(connectors))
	for _, connector := range connectors {
		byType[connector.typ] = connector
	}

	h.svc = &Service{
		l:               zap.NewNop(),
		integrations:    h.integrations,
		connectors:      byType,
		usageRepo:       h.usage,
		notifications:   h.notifications,
		interactiveWait: 2 * time.Second,
		breaker:         newCircuitBreaker(),
		now:             func() int64 { return h.now },
	}
	return h
}

func (h *harness) setPrimary(provider integration.Type) {
	h.control.PrimaryProvider = &provider
}

func (h *harness) setFallback(provider integration.Type) {
	h.control.FallbackProvider = &provider
}

func (h *harness) bind(t *testing.T, provider integration.Type, fallback bool) *boundProvider {
	t.Helper()
	bound, err := h.svc.bind(t.Context(), h.tenant, provider, fallback)
	require.NoError(t, err)
	return bound
}

func (h *harness) attempt(bound *boundProvider, wanted carrierintel.LookupDepth) *lookupAttempt {
	return &lookupAttempt{
		tenant:  h.tenant,
		control: h.control,
		primary: bound,
		subject: repositories.CarrierIntelSubject{
			SubjectType: carrierintel.SubjectTypeCarrier,
			SubjectID:   pulid.MustNew("car_").String(),
			DOTNumber:   "818175",
		},
		wanted: wanted,
	}
}

func providerError(kind services.CarrierIntelErrorKind) error {
	return &services.CarrierIntelProviderError{Provider: integration.TypeCarrierOK, Kind: kind}
}

func TestCircuitBreaker(t *testing.T) {
	t.Parallel()

	breaker := newCircuitBreaker()
	now := time.Unix(testNow, 0)
	key := "tenant:provider"

	for range breakerFailureThreshold - 1 {
		breaker.record(key, true, now)
	}
	assert.False(t, breaker.isOpen(key, now))

	breaker.record(key, true, now)
	assert.True(t, breaker.isOpen(key, now))
	assert.True(t, breaker.isOpen(key, now.Add(breakerOpenDuration-time.Second)))
	assert.False(t, breaker.isOpen(key, now.Add(breakerOpenDuration)))

	breaker.record(key, false, now)
	assert.False(t, breaker.isOpen(key, now))
	breaker.record(key, true, now)
	assert.False(t, breaker.isOpen(key, now))
}

func TestIsBillable(t *testing.T) {
	t.Parallel()

	perDOT := carrierintel.EndpointPrice{
		Model:    carrierintel.BillingModelPerDOTMonth,
		UnitCost: decimal.RequireFromString("3"),
	}
	perMatch := carrierintel.EndpointPrice{
		Model:    carrierintel.BillingModelPerMatch,
		UnitCost: decimal.RequireFromString("0.1"),
	}
	perRequest := carrierintel.EndpointPrice{
		Model:    carrierintel.BillingModelPerRequest,
		UnitCost: decimal.RequireFromString("0.003"),
	}
	free := carrierintel.EndpointPrice{Model: carrierintel.BillingModelFree}

	tests := []struct {
		name      string
		price     carrierintel.EndpointPrice
		call      services.CarrierIntelCall
		billable  bool
		wantUnits int
	}{
		{name: "free endpoint", price: free, call: services.CarrierIntelCall{Found: true}},
		{
			name:      "per DOT found",
			price:     perDOT,
			call:      services.CarrierIntelCall{Found: true},
			billable:  true,
			wantUnits: 1,
		},
		{name: "per match not found", price: perMatch, call: services.CarrierIntelCall{}},
		{
			name:  "failed call without a match",
			price: perMatch,
			call: services.CarrierIntelCall{
				Err: providerError(services.CarrierIntelErrorUnavailable),
			},
		},
		{
			name:      "per request success counts units",
			price:     perRequest,
			call:      services.CarrierIntelCall{Units: 4},
			billable:  true,
			wantUnits: 4,
		},
		{
			name:  "per request failure",
			price: perRequest,
			call: services.CarrierIntelCall{
				Found: true,
				Err:   providerError(services.CarrierIntelErrorInvalidRequest),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			billable, units := isBillable(tt.price, tt.call)
			assert.Equal(t, tt.billable, billable)
			if tt.billable {
				assert.Equal(t, tt.wantUnits, units)
			}
		})
	}
}

func TestUsageOutcomeAndOutageClassification(t *testing.T) {
	t.Parallel()

	limited := &restx.RateLimitedError{RetryAfter: time.Second}

	assert.Equal(t, carrierintel.UsageOutcomeSuccess,
		usageOutcome(services.CarrierIntelCall{Found: true}))
	assert.Equal(t, carrierintel.UsageOutcomeNotFound, usageOutcome(services.CarrierIntelCall{}))
	assert.Equal(t, carrierintel.UsageOutcomeDenied,
		usageOutcome(services.CarrierIntelCall{Err: limited}))
	assert.Equal(t, carrierintel.UsageOutcomeTransport,
		usageOutcome(services.CarrierIntelCall{Err: errors.New("dial tcp: timeout")}))
	assert.Equal(t, carrierintel.UsageOutcomeRateLimited, usageOutcome(services.CarrierIntelCall{
		Err: providerError(services.CarrierIntelErrorRateLimited),
	}))
	assert.Equal(t, carrierintel.UsageOutcomeServerError, usageOutcome(services.CarrierIntelCall{
		Err: providerError(services.CarrierIntelErrorUnavailable),
	}))
	assert.Equal(t, carrierintel.UsageOutcomeClientError, usageOutcome(services.CarrierIntelCall{
		Err: providerError(services.CarrierIntelErrorUnauthorized),
	}))

	assert.False(t, isOutageError(limited))
	assert.False(t, isOutageError(context.Canceled))
	assert.True(t, isOutageError(errors.New("connection reset")))
	assert.True(t, isOutageError(providerError(services.CarrierIntelErrorUnavailable)))
	assert.False(t, isOutageError(providerError(services.CarrierIntelErrorUnauthorized)))
	assert.False(t, isOutageError(providerError(services.CarrierIntelErrorPaymentRequired)))
}

func TestToBusinessError(t *testing.T) {
	t.Parallel()

	var business *errortypes.BusinessError

	assert.NoError(t, toBusinessError(integration.TypeCarrierOK, nil))

	existing := errortypes.NewBusinessError("already friendly")
	assert.Same(t, existing, toBusinessError(integration.TypeCarrierOK, existing))

	kinds := []services.CarrierIntelErrorKind{
		services.CarrierIntelErrorUnauthorized,
		services.CarrierIntelErrorPaymentRequired,
		services.CarrierIntelErrorInvalidRequest,
		services.CarrierIntelErrorUnsupported,
		services.CarrierIntelErrorUnavailable,
	}
	for _, kind := range kinds {
		err := toBusinessError(integration.TypeCarrierOK, providerError(kind))
		require.ErrorAs(t, err, &business, string(kind))
		assert.True(t, services.IsCarrierIntelErrorKind(err, kind), string(kind))
	}

	limited := toBusinessError(integration.TypeCarrierOK, &services.CarrierIntelProviderError{
		Provider:   integration.TypeCarrierOK,
		Kind:       services.CarrierIntelErrorRateLimited,
		RetryAfter: 1500 * time.Millisecond,
	})
	require.ErrorAs(t, limited, &business)
	wait, ok := retryAfter(limited)
	assert.True(t, ok)
	assert.Equal(t, 1500*time.Millisecond, wait)
}

func TestWithRateLimitWait(t *testing.T) {
	t.Parallel()

	limitedErr := &restx.RateLimitedError{RetryAfter: 10 * time.Millisecond}

	t.Run("interactive calls retry once after a short wait", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		calls := 0
		ctx := carrierintel.WithPurpose(t.Context(), carrierintel.PurposeVet)
		err := h.svc.withRateLimitWait(ctx, func() error {
			calls++
			if calls == 1 {
				return limitedErr
			}
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, 2, calls)
	})

	t.Run("background calls do not wait", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		calls := 0
		ctx := carrierintel.WithPurpose(t.Context(), carrierintel.PurposeMonitor)
		err := h.svc.withRateLimitWait(ctx, func() error {
			calls++
			return limitedErr
		})
		require.ErrorIs(t, err, limitedErr)
		assert.Equal(t, 1, calls)
	})

	t.Run("waits longer than the interactive budget are refused", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		calls := 0
		ctx := carrierintel.WithPurpose(t.Context(), carrierintel.PurposeVet)
		err := h.svc.withRateLimitWait(ctx, func() error {
			calls++
			return &restx.RateLimitedError{RetryAfter: time.Minute}
		})
		require.Error(t, err)
		assert.Equal(t, 1, calls)
	})
}

func TestGuardBudget(t *testing.T) {
	t.Parallel()

	t.Run("no caps configured never queries spend", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, newCarrierOKConnector())
		bound := h.bind(t, integration.TypeCarrierOK, false)
		err := h.svc.guardBudget(t.Context(), &budgetCheck{
			tenant:    h.tenant,
			control:   h.control,
			bound:     bound,
			endpoint:  carrierintel.EndpointProfileFull,
			dotNumber: "818175",
		})
		require.NoError(t, err)
		assert.Zero(t, h.usage.costCalls)
	})

	t.Run("a DOT already billed this month costs nothing", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, newCarrierOKConnector())
		bound := h.bind(t, integration.TypeCarrierOK, false)
		limit := decimal.RequireFromString("10")
		h.control.MonthlySpendCap = &limit
		h.usage.cost = decimal.RequireFromString("10")
		price := bound.prices.Price(carrierintel.EndpointProfileFull)
		h.usage.dedupeKeys[carrierintel.BillingDedupeKey(
			carrierintel.EndpointProfileFull, price, "818175", h.now,
		)] = true

		err := h.svc.guardBudget(t.Context(), &budgetCheck{
			tenant:    h.tenant,
			control:   h.control,
			bound:     bound,
			endpoint:  carrierintel.EndpointProfileFull,
			dotNumber: "818175",
		})
		require.NoError(t, err)
		assert.Zero(t, h.usage.costCalls)
	})

	t.Run("exceeding the monthly cap is refused and notifies once", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, newCarrierOKConnector())
		bound := h.bind(t, integration.TypeCarrierOK, false)
		limit := decimal.RequireFromString("10")
		h.control.MonthlySpendCap = &limit
		h.usage.cost = decimal.RequireFromString("8")

		check := &budgetCheck{
			tenant:    h.tenant,
			control:   h.control,
			bound:     bound,
			endpoint:  carrierintel.EndpointProfileFull,
			dotNumber: "818175",
		}
		err := h.svc.guardBudget(t.Context(), check)
		require.ErrorIs(t, err, ErrSpendCapReached)
		var business *errortypes.BusinessError
		require.ErrorAs(t, err, &business)

		require.ErrorIs(t, h.svc.guardBudget(t.Context(), check), ErrSpendCapReached)
		assert.Equal(t, []string{eventSpendHardCap}, h.notifications.eventTypes())
		assert.Equal(t, 1, h.usage.costCalls)
	})

	t.Run("crossing the soft cap allows the call and warns", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, newCarrierOKConnector())
		bound := h.bind(t, integration.TypeCarrierOK, false)
		limit := decimal.RequireFromString("10")
		h.control.MonthlySpendCap = &limit
		h.control.SoftCapPercent = 80
		h.usage.cost = decimal.RequireFromString("7.6")

		err := h.svc.guardBudget(t.Context(), &budgetCheck{
			tenant:    h.tenant,
			control:   h.control,
			bound:     bound,
			endpoint:  carrierintel.EndpointProfileLite,
			dotNumber: "818175",
		})
		require.NoError(t, err)
		assert.Equal(t, []string{eventSpendSoftCap}, h.notifications.eventTypes())
	})

	t.Run("daily full profile cap only limits full profiles", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, newCarrierOKConnector())
		bound := h.bind(t, integration.TypeCarrierOK, false)
		h.control.DailyFullProfileCap = new(5)
		h.usage.billable = 5

		full := &budgetCheck{
			tenant:    h.tenant,
			control:   h.control,
			bound:     bound,
			endpoint:  carrierintel.EndpointProfileFull,
			dotNumber: "818175",
		}
		require.ErrorIs(t, h.svc.guardBudget(t.Context(), full), ErrSpendCapReached)

		lite := *full
		lite.endpoint = carrierintel.EndpointProfileLite
		require.NoError(t, h.svc.guardBudget(t.Context(), &lite))
	})
}

func TestRecordCallDedupesMonthlyCharges(t *testing.T) {
	t.Parallel()

	h := newHarness(t, newCarrierOKConnector())
	bound := h.bind(t, integration.TypeCarrierOK, false)
	ctx := carrierintel.WithPurpose(t.Context(), carrierintel.PurposeVet)
	call := services.CarrierIntelCall{
		Endpoint:  carrierintel.EndpointProfileFull,
		DOTNumber: "818175",
		Found:     true,
	}

	h.svc.recordCall(ctx, bound, call)
	h.svc.recordCall(ctx, bound, call)

	require.Len(t, h.usage.records, 2)
	first, second := h.usage.records[0], h.usage.records[1]
	assert.True(t, first.Billable)
	assert.True(t, first.EstimatedCost.Equal(decimal.RequireFromString("3")))
	assert.NotEmpty(t, first.DedupeKey)
	assert.Equal(t, carrierintel.PurposeVet, first.Purpose)
	assert.Equal(t, h.tenant.UserID, first.InitiatedByID)

	assert.False(t, second.Billable)
	assert.True(t, second.EstimatedCost.IsZero())
	assert.Empty(t, second.DedupeKey)
}

func TestChooseDepth(t *testing.T) {
	t.Parallel()

	t.Run("upgrades to a full profile already paid for this month", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, newCarrierOKConnector())
		bound := h.bind(t, integration.TypeCarrierOK, false)
		price := bound.prices.Price(carrierintel.EndpointProfileFull)
		h.usage.dedupeKeys[carrierintel.BillingDedupeKey(
			carrierintel.EndpointProfileFull, price, "818175", h.now,
		)] = true

		depth := h.svc.chooseDepth(
			t.Context(),
			h.attempt(bound, carrierintel.LookupDepthFMCSA),
			bound,
		)
		assert.Equal(t, carrierintel.LookupDepthFull, depth)
	})

	t.Run("keeps the requested depth when nothing was paid", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, newCarrierOKConnector())
		bound := h.bind(t, integration.TypeCarrierOK, false)
		depth := h.svc.chooseDepth(
			t.Context(),
			h.attempt(bound, carrierintel.LookupDepthLite),
			bound,
		)
		assert.Equal(t, carrierintel.LookupDepthLite, depth)
	})

	t.Run("falls back to the deepest depth the provider supports", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, newFMCSAConnector())
		bound := h.bind(t, integration.TypeFMCSAQCMobile, false)
		depth := h.svc.chooseDepth(
			t.Context(),
			h.attempt(bound, carrierintel.LookupDepthFull),
			bound,
		)
		assert.Equal(t, carrierintel.LookupDepthFMCSA, depth)
	})
}

func TestLookupWithFallback(t *testing.T) {
	t.Parallel()

	t.Run("primary success does not touch the fallback", func(t *testing.T) {
		t.Parallel()
		primary, fallback := newCarrierOKConnector(), newFMCSAConnector()
		h := newHarness(t, primary, fallback)
		h.setPrimary(integration.TypeCarrierOK)
		h.setFallback(integration.TypeFMCSAQCMobile)
		bound := h.bind(t, integration.TypeCarrierOK, false)

		result, used, err := h.svc.lookupWithFallback(
			t.Context(), h.attempt(bound, carrierintel.LookupDepthFull),
		)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, integration.TypeCarrierOK, used.provider)
		assert.Empty(t, fallback.client.lookups())
		require.Len(t, primary.client.lookups(), 1)
		assert.Empty(t, primary.client.lookups()[0].Identifier.DocketNumber)
	})

	t.Run("primary outage answers from the fallback", func(t *testing.T) {
		t.Parallel()
		primary, fallback := newCarrierOKConnector(), newFMCSAConnector()
		primary.client.errs = []error{providerError(services.CarrierIntelErrorUnavailable)}
		h := newHarness(t, primary, fallback)
		h.setPrimary(integration.TypeCarrierOK)
		h.setFallback(integration.TypeFMCSAQCMobile)
		bound := h.bind(t, integration.TypeCarrierOK, false)

		_, used, err := h.svc.lookupWithFallback(
			t.Context(), h.attempt(bound, carrierintel.LookupDepthFull),
		)
		require.NoError(t, err)
		assert.True(t, used.fallback)
		assert.Equal(t, integration.TypeFMCSAQCMobile, used.provider)
		require.Len(t, fallback.client.lookups(), 1)
		assert.Equal(t, carrierintel.LookupDepthFMCSA, fallback.client.lookups()[0].Depth)
	})

	t.Run("rejected credentials never fall back", func(t *testing.T) {
		t.Parallel()
		primary, fallback := newCarrierOKConnector(), newFMCSAConnector()
		primary.client.errs = []error{providerError(services.CarrierIntelErrorUnauthorized)}
		h := newHarness(t, primary, fallback)
		h.setPrimary(integration.TypeCarrierOK)
		h.setFallback(integration.TypeFMCSAQCMobile)
		bound := h.bind(t, integration.TypeCarrierOK, false)

		_, _, err := h.svc.lookupWithFallback(
			t.Context(), h.attempt(bound, carrierintel.LookupDepthFull),
		)
		var business *errortypes.BusinessError
		require.ErrorAs(t, err, &business)
		assert.True(
			t,
			services.IsCarrierIntelErrorKind(err, services.CarrierIntelErrorUnauthorized),
		)
		assert.Empty(t, fallback.client.lookups())
	})

	t.Run("spend cap refusal never falls back", func(t *testing.T) {
		t.Parallel()
		primary, fallback := newCarrierOKConnector(), newFMCSAConnector()
		h := newHarness(t, primary, fallback)
		h.setPrimary(integration.TypeCarrierOK)
		h.setFallback(integration.TypeFMCSAQCMobile)
		limit := decimal.RequireFromString("1")
		h.control.MonthlySpendCap = &limit
		h.usage.cost = decimal.RequireFromString("1")
		bound := h.bind(t, integration.TypeCarrierOK, false)

		_, _, err := h.svc.lookupWithFallback(
			t.Context(), h.attempt(bound, carrierintel.LookupDepthFull),
		)
		var business *errortypes.BusinessError
		require.ErrorAs(t, err, &business)
		assert.Empty(t, primary.client.lookups())
		assert.Empty(t, fallback.client.lookups())
	})

	t.Run("an open breaker skips straight to the fallback", func(t *testing.T) {
		t.Parallel()
		primary, fallback := newCarrierOKConnector(), newFMCSAConnector()
		h := newHarness(t, primary, fallback)
		h.setPrimary(integration.TypeCarrierOK)
		h.setFallback(integration.TypeFMCSAQCMobile)
		key := breakerKey(h.tenant, integration.TypeCarrierOK)
		for range breakerFailureThreshold {
			h.svc.breaker.record(key, true, time.Unix(h.now, 0))
		}
		bound := h.bind(t, integration.TypeCarrierOK, false)

		_, used, err := h.svc.lookupWithFallback(
			t.Context(), h.attempt(bound, carrierintel.LookupDepthFull),
		)
		require.NoError(t, err)
		assert.Equal(t, integration.TypeFMCSAQCMobile, used.provider)
		assert.Empty(t, primary.client.lookups())
	})

	t.Run("outage without a fallback reports the provider unavailable", func(t *testing.T) {
		t.Parallel()
		primary := newCarrierOKConnector()
		primary.client.errs = []error{providerError(services.CarrierIntelErrorUnavailable)}
		h := newHarness(t, primary)
		h.setPrimary(integration.TypeCarrierOK)
		bound := h.bind(t, integration.TypeCarrierOK, false)

		_, _, err := h.svc.lookupWithFallback(
			t.Context(), h.attempt(bound, carrierintel.LookupDepthFull),
		)
		var business *errortypes.BusinessError
		require.ErrorAs(t, err, &business)
		assert.True(t, services.IsCarrierIntelErrorKind(err, services.CarrierIntelErrorUnavailable))
	})

	t.Run("not found is a result, not an error", func(t *testing.T) {
		t.Parallel()
		primary := newCarrierOKConnector()
		primary.client.errs = []error{providerError(services.CarrierIntelErrorNotFound)}
		h := newHarness(t, primary)
		h.setPrimary(integration.TypeCarrierOK)
		bound := h.bind(t, integration.TypeCarrierOK, false)

		result, _, err := h.svc.lookupWithFallback(
			t.Context(), h.attempt(bound, carrierintel.LookupDepthFull),
		)
		require.NoError(t, err)
		assert.True(t, result.NotFound)
		assert.Equal(t, carrierintel.EndpointProfileFull, result.Endpoint)
	})
}

func TestResolvePrimaryRequiresProvider(t *testing.T) {
	t.Parallel()

	h := newHarness(t, newCarrierOKConnector())
	_, err := h.svc.resolvePrimary(t.Context(), h.tenant, h.control)
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)

	h.setFallback(integration.TypeFMCSAQCMobile)
	_, ok := h.svc.resolveFallback(t.Context(), h.tenant, h.control)
	assert.False(t, ok)
}
