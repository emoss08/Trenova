package completionrouter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSpend struct {
	spend map[pulid.ID]decimal.Decimal
}

func (f *fakeSpend) MonthSpend(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]decimal.Decimal, error) {
	out := make(map[pulid.ID]decimal.Decimal, len(ids))
	for _, id := range ids {
		out[id] = f.spend[id]
	}

	return out, nil
}

type fakeSlots struct {
	mu       sync.Mutex
	full     map[pulid.ID]bool
	busyFor  map[pulid.ID]int
	tries    map[pulid.ID]int
	held     map[string]pulid.ID
	released []pulid.ID
}

func newFakeSlots() *fakeSlots {
	return &fakeSlots{
		full:    map[pulid.ID]bool{},
		busyFor: map[pulid.ID]int{},
		tries:   map[pulid.ID]int{},
		held:    map[string]pulid.ID{},
	}
}

func (f *fakeSlots) Acquire(_ context.Context, req repositories.AcquireProviderSlotRequest) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tries[req.ProviderID]++
	if f.full[req.ProviderID] || f.tries[req.ProviderID] <= f.busyFor[req.ProviderID] {
		return false, nil
	}
	f.held[req.Token] = req.ProviderID

	return true, nil
}

func (f *fakeSlots) Refresh(_ context.Context, _ repositories.ProviderSlotRequest) (bool, error) {
	return true, nil
}

func (f *fakeSlots) Release(_ context.Context, req repositories.ProviderSlotRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.held, req.Token)
	f.released = append(f.released, req.ProviderID)

	return nil
}

type fakeKeys struct {
	repositories.AIProviderKeyRepository

	mu      sync.Mutex
	touched []pulid.ID
}

func (f *fakeKeys) TouchUsed(_ context.Context, req repositories.TouchAIProviderKeyRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.touched = append(f.touched, req.ID)

	return nil
}

func withLimits(service *Service, p *Params) *Service {
	service.limits = newProviderLimits(p)

	return service
}

func capped(provider *aiprovider.Provider, cap string, onCap aiprovider.CapAction) *aiprovider.Provider {
	value := decimal.RequireFromString(cap)
	provider.MonthlyCapUSD = &value
	provider.OnCap = onCap

	return provider
}

func TestCap_MovesATaskPastAProviderThatSpentItsCap(t *testing.T) {
	t.Parallel()

	firstServer, firstCalls := chatServer(t, http.StatusOK, `{"ok":true}`)
	secondServer, secondCalls := chatServer(t, http.StatusOK, `{"ok":true}`)
	first := capped(openAIChatProvider("first", firstServer.URL, 10), "50", aiprovider.CapActionNext)
	second := openAIChatProvider("second", secondServer.URL, 20)
	service := withLimits(newTestService(t, first, second), &Params{
		Spend: &fakeSpend{spend: map[pulid.ID]decimal.Decimal{first.ID: decimal.NewFromInt(50)}},
	})

	result, err := service.CompleteStructured(t.Context(), generalRequest())
	require.NoError(t, err)
	assert.Equal(t, second.ID, result.ProviderID)
	assert.Zero(t, firstCalls.Load())
	assert.Equal(t, int32(1), secondCalls.Load())
}

func TestCap_StopsATaskAtAProviderSetToStop(t *testing.T) {
	t.Parallel()

	firstServer, firstCalls := chatServer(t, http.StatusOK, `{"ok":true}`)
	secondServer, secondCalls := chatServer(t, http.StatusOK, `{"ok":true}`)
	first := capped(openAIChatProvider("first", firstServer.URL, 10), "50", aiprovider.CapActionStop)
	second := openAIChatProvider("second", secondServer.URL, 20)
	service := withLimits(newTestService(t, first, second), &Params{
		Spend: &fakeSpend{spend: map[pulid.ID]decimal.Decimal{first.ID: decimal.NewFromInt(60)}},
	})

	_, err := service.CompleteStructured(t.Context(), generalRequest())
	require.ErrorIs(t, err, serviceports.ErrProviderCapReached)
	assert.Zero(t, firstCalls.Load())
	assert.Zero(t, secondCalls.Load(), "a provider set to stop does not hand the task on")
}

func TestCap_LeavesAProviderUnderItsCapInLine(t *testing.T) {
	t.Parallel()

	server, calls := chatServer(t, http.StatusOK, `{"ok":true}`)
	provider := capped(openAIChatProvider("only", server.URL, 10), "50", aiprovider.CapActionStop)
	service := withLimits(newTestService(t, provider), &Params{
		Spend: &fakeSpend{spend: map[pulid.ID]decimal.Decimal{provider.ID: decimal.RequireFromString("49.99")}},
	})

	_, err := service.CompleteStructured(t.Context(), generalRequest())
	require.NoError(t, err)
	assert.Equal(t, int32(1), calls.Load())
}

func TestSlots_SendsWorkOnWhenAProviderIsAtItsLimit(t *testing.T) {
	t.Parallel()

	firstServer, firstCalls := chatServer(t, http.StatusOK, `{"ok":true}`)
	secondServer, secondCalls := chatServer(t, http.StatusOK, `{"ok":true}`)
	first := openAIChatProvider("first", firstServer.URL, 10)
	second := openAIChatProvider("second", secondServer.URL, 20)
	slots := newFakeSlots()
	slots.full[first.ID] = true
	service := withLimits(newTestService(t, first, second), &Params{Slots: slots})

	result, err := service.CompleteStructured(t.Context(), generalRequest())
	require.NoError(t, err)
	assert.Equal(t, second.ID, result.ProviderID)
	assert.Zero(t, firstCalls.Load())
	assert.Equal(t, int32(1), secondCalls.Load())
	assert.Equal(t, []pulid.ID{second.ID}, slots.released)
	assert.Empty(t, slots.held, "every slot taken is given back")
}

// The last provider a call can go to waits for a slot rather than failing: a
// slot frees within seconds, and refusing at once failed a person's reply
// while three conversations ran at once. An earlier one still hands on at once.
func TestSlots_TheLastProviderWaitsForASlot(t *testing.T) {
	t.Parallel()

	server, calls := chatServer(t, http.StatusOK, `{"ok":true}`)
	only := openAIChatProvider("only", server.URL, 10)
	slots := newFakeSlots()
	slots.busyFor[only.ID] = 2
	service := withLimits(newTestService(t, only), &Params{Slots: slots})

	result, err := service.CompleteStructured(t.Context(), generalRequest())
	require.NoError(t, err)
	assert.Equal(t, only.ID, result.ProviderID)
	assert.Equal(t, int32(1), calls.Load())
	assert.Equal(t, 3, slots.tries[only.ID], "it tried until a slot freed")
}

func TestTimeout_MovesOnFromAProviderThatTakesTooLong(t *testing.T) {
	t.Parallel()

	slow := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
	}))
	t.Cleanup(slow.Close)
	fastServer, fastCalls := chatServer(t, http.StatusOK, `{"ok":true}`)
	first := openAIChatProvider("slow", slow.URL, 10)
	first.TimeoutSeconds = 1
	second := openAIChatProvider("fast", fastServer.URL, 20)
	service := newTestService(t, first, second)

	started := time.Now()
	result, err := service.CompleteStructured(t.Context(), generalRequest())
	require.NoError(t, err)
	assert.Equal(t, second.ID, result.ProviderID)
	assert.Equal(t, int32(1), fastCalls.Load())
	assert.Less(t, time.Since(started), 5*time.Second)
}

func TestTimeout_NamesTheProvidersOwnLimit(t *testing.T) {
	t.Parallel()

	slow := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
		}
	}))
	t.Cleanup(slow.Close)
	provider := openAIChatProvider("slow", slow.URL, 10)
	provider.TimeoutSeconds = 1
	service := newTestService(t, provider)

	_, err := service.CompleteStructured(t.Context(), generalRequest())
	require.Error(t, err)
	assert.ErrorIs(t, err, errProviderTimedOut)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestKeyFallback_UsesTheReplacedKeyWhileTheNewOneIsRefused(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Get("Authorization"))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") != "Bearer old-key" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"invalid key"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"model":"m","choices":[{"finish_reason":"stop",` +
			`"message":{"role":"assistant","content":"{\"ok\":true}"}}]}`))
	}))
	t.Cleanup(server.Close)

	provider := openAIChatProvider("rotating", server.URL, 10)
	service := newTestService(t, provider)
	current, err := service.encryption.EncryptString("new-key")
	require.NoError(t, err)
	previous, err := service.encryption.EncryptString("old-key")
	require.NoError(t, err)
	expires := timeutils.NowUnix() + 3600
	provider.APIKey = current
	provider.PreviousAPIKey = previous
	provider.RotationExpiresAt = &expires

	_, err = service.CompleteStructured(t.Context(), generalRequest())
	require.NoError(t, err)
	assert.Equal(t, []string{"Bearer new-key", "Bearer old-key"}, seen)

	expired := timeutils.NowUnix() - 1
	provider.RotationExpiresAt = &expired
	seen = nil
	_, err = service.CompleteStructured(t.Context(), generalRequest())
	require.Error(t, err)
	assert.Equal(t, []string{"Bearer new-key"}, seen, "an expired key is never sent")
}

func TestTouchKey_RecordsUseAtMostOnceAWhile(t *testing.T) {
	t.Parallel()

	server, _ := chatServer(t, http.StatusOK, `{"ok":true}`)
	provider := openAIChatProvider("keyed", server.URL, 10)
	keys := &fakeKeys{}
	service := withLimits(newTestService(t, provider), &Params{Keys: keys})
	encrypted, err := service.encryption.EncryptString("sk-key")
	require.NoError(t, err)
	provider.APIKey = encrypted

	for range 3 {
		_, err = service.CompleteStructured(t.Context(), generalRequest())
		require.NoError(t, err)
	}
	assert.Equal(t, []pulid.ID{provider.ID}, keys.touched)
}

func TestKeyRefused_ReadsOnlyAuthStatuses(t *testing.T) {
	t.Parallel()

	assert.False(t, keyRefused(errors.New("dial tcp: refused")))
}
