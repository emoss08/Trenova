package carrierintelservice

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelcontrol"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelusage"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/integrationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
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
	carrierintelports.CarrierIntelUsageRepository

	mu         sync.Mutex
	records    []*carrierintelusage.CarrierIntelUsageRecord
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
	entity *carrierintelusage.CarrierIntelUsageRecord,
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
	_ *carrierintelports.CountBillableUsageRequest,
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
	req notificationports.ExistsRecentNotificationRequest,
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
	recorder carrierintelports.CarrierIntelCallRecorder

	mu       sync.Mutex
	requests []*carrierintelports.CarrierIntelLookupRequest
	errs     []error
	result   *carrierintelports.CarrierIntelLookupResult
}

func (c *fakeClient) Provider() integration.Type { return c.provider }

func (c *fakeClient) IsSandbox() bool { return true }

func (c *fakeClient) Lookup(
	ctx context.Context,
	req *carrierintelports.CarrierIntelLookupRequest,
) (*carrierintelports.CarrierIntelLookupResult, error) {
	c.mu.Lock()
	c.requests = append(c.requests, req)
	var err error
	if len(c.errs) > 0 {
		err = c.errs[0]
		c.errs = c.errs[1:]
	}
	c.mu.Unlock()

	if c.recorder != nil {
		c.recorder(ctx, carrierintelports.CarrierIntelCall{
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
	return &carrierintelports.CarrierIntelLookupResult{
		Profile:  &carrierintel.Profile{},
		Endpoint: depthEndpoint(req.Depth),
	}, nil
}

func (c *fakeClient) lookups() []*carrierintelports.CarrierIntelLookupRequest {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]*carrierintelports.CarrierIntelLookupRequest(nil), c.requests...)
}

type fakeConnector struct {
	typ    integration.Type
	caps   carrierintel.CapabilitySet
	prices carrierintelusage.PriceBook
	client *fakeClient
}

func (c *fakeConnector) IntegrationType() integration.Type { return c.typ }

func (c *fakeConnector) Descriptor() carrierintel.ProviderDescriptor {
	return carrierintel.ProviderDescriptor{Capabilities: c.caps}
}

func (c *fakeConnector) PriceBook() carrierintelusage.PriceBook { return c.prices }

func (c *fakeConnector) TestConnection(context.Context, map[string]string) error { return nil }

func (c *fakeConnector) Bind(
	params *carrierintelports.CarrierIntelBindParams,
) (carrierintelports.CarrierIntelClient, error) {
	c.client.recorder = params.Recorder
	return c.client, nil
}

func carrierOKPrices() carrierintelusage.PriceBook {
	return carrierintelusage.PriceBook{
		carrierintelusage.EndpointProfileFull: {
			Model:    carrierintelusage.BillingModelPerDOTMonth,
			UnitCost: decimal.RequireFromString("3"),
		},
		carrierintelusage.EndpointProfileLite: {
			Model:    carrierintelusage.BillingModelPerDOTMonth,
			UnitCost: decimal.RequireFromString("0.5"),
		},
		carrierintelusage.EndpointProfileFMCSA: {
			Model:    carrierintelusage.BillingModelPerMatch,
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
		prices: carrierintelusage.PriceBook{},
		client: &fakeClient{provider: integration.TypeFMCSAQCMobile},
	}
}

type harness struct {
	svc           *Service
	usage         *fakeUsageRepo
	notifications *fakeNotifications
	integrations  *fakeIntegrations
	tenant        pagination.TenantInfo
	control       *carrierintelcontrol.CarrierIntelControl
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
		control:       carrierintelcontrol.NewDefaultControl(tenant.OrgID, tenant.BuID),
		now:           testNow,
	}

	byType := make(map[integration.Type]carrierintelports.CarrierIntelConnector, len(connectors))
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

func (h *harness) Bind(t *testing.T, provider integration.Type, fallback bool) *BoundProvider {
	t.Helper()
	bound, err := h.svc.Bind(t.Context(), h.tenant, provider, fallback)
	require.NoError(t, err)
	return bound
}

func (h *harness) attempt(bound *BoundProvider, wanted carrierintel.LookupDepth) *lookupAttempt {
	return &lookupAttempt{
		tenant:  h.tenant,
		control: h.control,
		primary: bound,
		subject: carrierintelports.CarrierIntelSubject{
			SubjectType: carrierintel.SubjectTypeCarrier,
			SubjectID:   pulid.MustNew("car_").String(),
			DOTNumber:   "818175",
		},
		wanted: wanted,
	}
}

func providerError(kind carrierintelports.CarrierIntelErrorKind) error {
	return &carrierintelports.CarrierIntelProviderError{
		Provider: integration.TypeCarrierOK,
		Kind:     kind,
	}
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

	perDOT := carrierintelusage.EndpointPrice{
		Model:    carrierintelusage.BillingModelPerDOTMonth,
		UnitCost: decimal.RequireFromString("3"),
	}
	perMatch := carrierintelusage.EndpointPrice{
		Model:    carrierintelusage.BillingModelPerMatch,
		UnitCost: decimal.RequireFromString("0.1"),
	}
	perRequest := carrierintelusage.EndpointPrice{
		Model:    carrierintelusage.BillingModelPerRequest,
		UnitCost: decimal.RequireFromString("0.003"),
	}
	free := carrierintelusage.EndpointPrice{Model: carrierintelusage.BillingModelFree}

	tests := []struct {
		name      string
		price     carrierintelusage.EndpointPrice
		call      carrierintelports.CarrierIntelCall
		billable  bool
		wantUnits int
	}{
		{name: "free endpoint", price: free, call: carrierintelports.CarrierIntelCall{Found: true}},
		{
			name:      "per DOT found",
			price:     perDOT,
			call:      carrierintelports.CarrierIntelCall{Found: true},
			billable:  true,
			wantUnits: 1,
		},
		{name: "per match not found", price: perMatch, call: carrierintelports.CarrierIntelCall{}},
		{
			name:  "failed call without a match",
			price: perMatch,
			call: carrierintelports.CarrierIntelCall{
				Err: providerError(carrierintelports.CarrierIntelErrorUnavailable),
			},
		},
		{
			name:      "per request success counts units",
			price:     perRequest,
			call:      carrierintelports.CarrierIntelCall{Units: 4},
			billable:  true,
			wantUnits: 4,
		},
		{
			name:  "per request failure",
			price: perRequest,
			call: carrierintelports.CarrierIntelCall{
				Found: true,
				Err:   providerError(carrierintelports.CarrierIntelErrorInvalidRequest),
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

	assert.Equal(t, carrierintelusage.UsageOutcomeSuccess,
		usageOutcome(carrierintelports.CarrierIntelCall{Found: true}))
	assert.Equal(
		t,
		carrierintelusage.UsageOutcomeNotFound,
		usageOutcome(carrierintelports.CarrierIntelCall{}),
	)
	assert.Equal(t, carrierintelusage.UsageOutcomeDenied,
		usageOutcome(carrierintelports.CarrierIntelCall{Err: limited}))
	assert.Equal(t, carrierintelusage.UsageOutcomeTransport,
		usageOutcome(carrierintelports.CarrierIntelCall{Err: errors.New("dial tcp: timeout")}))
	assert.Equal(
		t,
		carrierintelusage.UsageOutcomeRateLimited,
		usageOutcome(carrierintelports.CarrierIntelCall{
			Err: providerError(carrierintelports.CarrierIntelErrorRateLimited),
		}),
	)
	assert.Equal(
		t,
		carrierintelusage.UsageOutcomeServerError,
		usageOutcome(carrierintelports.CarrierIntelCall{
			Err: providerError(carrierintelports.CarrierIntelErrorUnavailable),
		}),
	)
	assert.Equal(
		t,
		carrierintelusage.UsageOutcomeClientError,
		usageOutcome(carrierintelports.CarrierIntelCall{
			Err: providerError(carrierintelports.CarrierIntelErrorUnauthorized),
		}),
	)

	assert.False(t, isOutageError(limited))
	assert.False(t, isOutageError(context.Canceled))
	assert.True(t, isOutageError(errors.New("connection reset")))
	assert.True(t, isOutageError(providerError(carrierintelports.CarrierIntelErrorUnavailable)))
	assert.False(t, isOutageError(providerError(carrierintelports.CarrierIntelErrorUnauthorized)))
	assert.False(
		t,
		isOutageError(providerError(carrierintelports.CarrierIntelErrorPaymentRequired)),
	)
}

func TestToBusinessError(t *testing.T) {
	t.Parallel()

	var business *errortypes.BusinessError

	assert.NoError(t, ToBusinessError(integration.TypeCarrierOK, nil))

	existing := errortypes.NewBusinessError("already friendly")
	assert.Same(t, existing, ToBusinessError(integration.TypeCarrierOK, existing))

	kinds := []carrierintelports.CarrierIntelErrorKind{
		carrierintelports.CarrierIntelErrorUnauthorized,
		carrierintelports.CarrierIntelErrorPaymentRequired,
		carrierintelports.CarrierIntelErrorInvalidRequest,
		carrierintelports.CarrierIntelErrorUnsupported,
		carrierintelports.CarrierIntelErrorUnavailable,
	}
	for _, kind := range kinds {
		err := ToBusinessError(integration.TypeCarrierOK, providerError(kind))
		require.ErrorAs(t, err, &business, string(kind))
		assert.True(t, carrierintelports.IsCarrierIntelErrorKind(err, kind), string(kind))
	}

	limited := ToBusinessError(
		integration.TypeCarrierOK,
		&carrierintelports.CarrierIntelProviderError{
			Provider:   integration.TypeCarrierOK,
			Kind:       carrierintelports.CarrierIntelErrorRateLimited,
			RetryAfter: 1500 * time.Millisecond,
		},
	)
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
		err := h.svc.WithRateLimitWait(ctx, func() error {
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
		err := h.svc.WithRateLimitWait(ctx, func() error {
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
		err := h.svc.WithRateLimitWait(ctx, func() error {
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
		bound := h.Bind(t, integration.TypeCarrierOK, false)
		err := h.svc.GuardBudget(
			t.Context(),
			&BudgetCheck{
				Tenant:    h.tenant,
				Control:   h.control,
				Bound:     bound,
				Endpoint:  carrierintelusage.EndpointProfileFull,
				DOTNumber: "818175",
			},
		)
		require.NoError(t, err)
		assert.Zero(t, h.usage.costCalls)
	})

	t.Run("a DOT already billed this month costs nothing", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, newCarrierOKConnector())
		bound := h.Bind(t, integration.TypeCarrierOK, false)
		limit := decimal.RequireFromString("10")
		h.control.MonthlySpendCap = &limit
		h.usage.cost = decimal.RequireFromString("10")
		price := bound.prices.Price(carrierintelusage.EndpointProfileFull)
		h.usage.dedupeKeys[carrierintelusage.BillingDedupeKey(
			carrierintelusage.EndpointProfileFull, price, "818175", h.now,
		)] = true

		err := h.svc.GuardBudget(
			t.Context(),
			&BudgetCheck{
				Tenant:    h.tenant,
				Control:   h.control,
				Bound:     bound,
				Endpoint:  carrierintelusage.EndpointProfileFull,
				DOTNumber: "818175",
			},
		)
		require.NoError(t, err)
		assert.Zero(t, h.usage.costCalls)
	})

	t.Run("exceeding the monthly cap is refused and notifies once", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, newCarrierOKConnector())
		bound := h.Bind(t, integration.TypeCarrierOK, false)
		limit := decimal.RequireFromString("10")
		h.control.MonthlySpendCap = &limit
		h.usage.cost = decimal.RequireFromString("8")

		check := &BudgetCheck{
			Tenant:    h.tenant,
			Control:   h.control,
			Bound:     bound,
			Endpoint:  carrierintelusage.EndpointProfileFull,
			DOTNumber: "818175",
		}

		err := h.svc.GuardBudget(t.Context(), check)
		require.ErrorIs(t, err, ErrSpendCapReached)
		var business *errortypes.BusinessError
		require.ErrorAs(t, err, &business)

		require.ErrorIs(t, h.svc.GuardBudget(t.Context(), check), ErrSpendCapReached)
		assert.Equal(t, []string{eventSpendHardCap}, h.notifications.eventTypes())
		assert.Equal(t, 1, h.usage.costCalls)
	})

	t.Run("crossing the soft cap allows the call and warns", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, newCarrierOKConnector())
		bound := h.Bind(t, integration.TypeCarrierOK, false)
		limit := decimal.RequireFromString("10")
		h.control.MonthlySpendCap = &limit
		h.control.SoftCapPercent = 80
		h.usage.cost = decimal.RequireFromString("7.6")

		err := h.svc.GuardBudget(
			t.Context(),
			&BudgetCheck{
				Tenant:    h.tenant,
				Control:   h.control,
				Bound:     bound,
				Endpoint:  carrierintelusage.EndpointProfileLite,
				DOTNumber: "818175",
			},
		)
		require.NoError(t, err)
		assert.Equal(t, []string{eventSpendSoftCap}, h.notifications.eventTypes())
	})

	t.Run("daily full profile cap only limits full profiles", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, newCarrierOKConnector())
		bound := h.Bind(t, integration.TypeCarrierOK, false)
		h.control.DailyFullProfileCap = new(5)
		h.usage.billable = 5

		full := &BudgetCheck{
			Tenant:    h.tenant,
			Control:   h.control,
			Bound:     bound,
			Endpoint:  carrierintelusage.EndpointProfileFull,
			DOTNumber: "818175",
		}

		require.ErrorIs(t, h.svc.GuardBudget(t.Context(), full), ErrSpendCapReached)

		lite := *full
		lite.Endpoint = carrierintelusage.EndpointProfileLite
		require.NoError(t, h.svc.GuardBudget(t.Context(), &lite))
	})
}

func TestRecordCallDedupesMonthlyCharges(t *testing.T) {
	t.Parallel()

	h := newHarness(t, newCarrierOKConnector())
	bound := h.Bind(t, integration.TypeCarrierOK, false)
	ctx := carrierintel.WithPurpose(t.Context(), carrierintel.PurposeVet)
	call := carrierintelports.CarrierIntelCall{
		Endpoint:  carrierintelusage.EndpointProfileFull,
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
		bound := h.Bind(t, integration.TypeCarrierOK, false)
		price := bound.prices.Price(carrierintelusage.EndpointProfileFull)
		h.usage.dedupeKeys[carrierintelusage.BillingDedupeKey(
			carrierintelusage.EndpointProfileFull, price, "818175", h.now,
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
		bound := h.Bind(t, integration.TypeCarrierOK, false)
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
		bound := h.Bind(t, integration.TypeFMCSAQCMobile, false)
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
		bound := h.Bind(t, integration.TypeCarrierOK, false)

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
		primary.client.errs = []error{providerError(carrierintelports.CarrierIntelErrorUnavailable)}
		h := newHarness(t, primary, fallback)
		h.setPrimary(integration.TypeCarrierOK)
		h.setFallback(integration.TypeFMCSAQCMobile)
		bound := h.Bind(t, integration.TypeCarrierOK, false)

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
		primary.client.errs = []error{
			providerError(carrierintelports.CarrierIntelErrorUnauthorized),
		}
		h := newHarness(t, primary, fallback)
		h.setPrimary(integration.TypeCarrierOK)
		h.setFallback(integration.TypeFMCSAQCMobile)
		bound := h.Bind(t, integration.TypeCarrierOK, false)

		_, _, err := h.svc.lookupWithFallback(
			t.Context(), h.attempt(bound, carrierintel.LookupDepthFull),
		)
		var business *errortypes.BusinessError
		require.ErrorAs(t, err, &business)
		assert.True(
			t,
			carrierintelports.IsCarrierIntelErrorKind(
				err,
				carrierintelports.CarrierIntelErrorUnauthorized,
			),
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
		bound := h.Bind(t, integration.TypeCarrierOK, false)

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
		bound := h.Bind(t, integration.TypeCarrierOK, false)

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
		primary.client.errs = []error{providerError(carrierintelports.CarrierIntelErrorUnavailable)}
		h := newHarness(t, primary)
		h.setPrimary(integration.TypeCarrierOK)
		bound := h.Bind(t, integration.TypeCarrierOK, false)

		_, _, err := h.svc.lookupWithFallback(
			t.Context(), h.attempt(bound, carrierintel.LookupDepthFull),
		)
		var business *errortypes.BusinessError
		require.ErrorAs(t, err, &business)
		assert.True(
			t,
			carrierintelports.IsCarrierIntelErrorKind(
				err,
				carrierintelports.CarrierIntelErrorUnavailable,
			),
		)
	})

	t.Run("not found is a result, not an error", func(t *testing.T) {
		t.Parallel()
		primary := newCarrierOKConnector()
		primary.client.errs = []error{providerError(carrierintelports.CarrierIntelErrorNotFound)}
		h := newHarness(t, primary)
		h.setPrimary(integration.TypeCarrierOK)
		bound := h.Bind(t, integration.TypeCarrierOK, false)

		result, _, err := h.svc.lookupWithFallback(
			t.Context(), h.attempt(bound, carrierintel.LookupDepthFull),
		)
		require.NoError(t, err)
		assert.True(t, result.NotFound)
		assert.Equal(t, carrierintelusage.EndpointProfileFull, result.Endpoint)
	})
}

func TestResolvePrimaryRequiresProvider(t *testing.T) {
	t.Parallel()

	h := newHarness(t, newCarrierOKConnector())
	_, err := h.svc.ResolvePrimary(t.Context(), h.tenant, h.control)
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)

	h.setFallback(integration.TypeFMCSAQCMobile)
	_, ok := h.svc.resolveFallback(t.Context(), h.tenant, h.control)
	assert.False(t, ok)
}
