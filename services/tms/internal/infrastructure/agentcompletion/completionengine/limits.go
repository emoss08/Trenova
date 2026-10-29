package completionengine

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiproviderkey"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	slotMargin         = 30 * time.Second
	slotStoreTimeout   = 2 * time.Second
	slotWait           = 20 * time.Second
	slotPoll           = 250 * time.Millisecond
	keyTouchInterval   = 5 * time.Minute
	maxTouchedProvider = 4096
)

var ErrProviderTimedOut = fmt.Errorf("the AI provider did not answer within its timeout: %w",
	context.DeadlineExceeded)

type providerLimits struct {
	spend aiproviderports.AIProviderSpendService
	slots llmports.ProviderSlotRepository
	keys  aiproviderports.AIProviderKeyRepository

	mu      sync.Mutex
	touched map[pulid.ID]time.Time
}

func NewProviderLimits(p *Params) *providerLimits {
	return &providerLimits{
		spend:   p.Spend,
		slots:   p.Slots,
		keys:    p.Keys,
		touched: make(map[pulid.ID]time.Time),
	}
}

func (s *Engine) UnderCap(
	ctx context.Context,
	tenant pagination.TenantInfo,
	providers []*aiprovider.Provider,
) ([]*aiprovider.Provider, error) {
	if s.Limits == nil || s.Limits.spend == nil {
		return providers, nil
	}

	capped := make([]pulid.ID, 0, len(providers))
	for _, provider := range providers {
		if provider.Capped() {
			capped = append(capped, provider.ID)
		}
	}
	if len(capped) == 0 {
		return providers, nil
	}

	spend, err := s.Limits.spend.MonthSpend(ctx, tenant, capped)
	if err != nil {
		s.Logger.Warn("could not read provider spend; monthly caps not applied to this call",
			zap.Error(err))
		return providers, nil
	}

	within := make([]*aiprovider.Provider, 0, len(providers))
	for _, provider := range providers {
		if !provider.CapReached(spend[provider.ID]) {
			within = append(within, provider)
			continue
		}
		s.Logger.Debug("skipping provider at its monthly cap",
			zap.String("provider", provider.Name),
			zap.String("onCap", string(provider.ResolvedOnCap())),
		)
		if provider.ResolvedOnCap() == aiprovider.CapActionStop {
			if len(within) == 0 {
				return nil, capReachedError(provider)
			}
			break
		}
	}
	if len(within) == 0 {
		return nil, capReachedError(providers[0])
	}

	return within, nil
}

func capReachedError(provider *aiprovider.Provider) error {
	return errortypes.NewBusinessError(
		"{0} has reached its monthly spending cap. Raise the cap or add another provider for this task",
		provider.Name,
	).WithInternal(llmports.ErrProviderCapReached)
}

func (s *Engine) ClaimSlot(
	ctx context.Context,
	provider *aiprovider.Provider,
	timeout time.Duration,
) (func(), error) {
	if s.Limits == nil || s.Limits.slots == nil || provider.ID.IsNil() {
		return func() {}, nil
	}

	ttl := timeout + slotMargin
	token := pulid.MustNew("slot_").String()
	acquired, err := s.awaitSlot(ctx, provider, token, ttl, waitsForSlot(ctx))
	if err != nil {
		s.Logger.Warn("could not claim a provider slot; calling without one",
			zap.String("provider", provider.Name),
			zap.Error(err))
		return func() {}, nil
	}
	if !acquired {
		return nil, fmt.Errorf("%s: %w", provider.Name, llmports.ErrProviderBusy)
	}

	stop := make(chan struct{})
	var once sync.Once
	go s.holdSlot(provider.ID, token, ttl, stop)

	return func() {
		once.Do(func() {
			close(stop)
			releaseCtx, release := context.WithTimeout(context.WithoutCancel(ctx), slotStoreTimeout)
			defer release()
			if releaseErr := s.Limits.slots.Release(releaseCtx, llmports.ProviderSlotRequest{
				ProviderID: provider.ID,
				Token:      token,
			}); releaseErr != nil {
				s.Logger.Warn("could not release a provider slot; it expires on its own",
					zap.String("provider", provider.Name),
					zap.Error(releaseErr))
			}
		})
	}, nil
}

type slotWaitKey struct{}

// lastCandidate marks the attempt on the last provider a call can go to. A busy
// provider earlier in the order hands the call on at once; the last one has
// nowhere to hand it, so it waits a short while for a slot instead.
func LastCandidate(ctx context.Context, idx, count int) context.Context {
	if idx != count-1 {
		return ctx
	}

	return context.WithValue(ctx, slotWaitKey{}, true)
}

func waitsForSlot(ctx context.Context) bool {
	wait, _ := ctx.Value(slotWaitKey{}).(bool)

	return wait
}

// awaitSlot claims a slot, waiting up to slotWait, or the caller's deadline
// when that comes first, when asked to. Refusing the
// last provider at once spent a turn's retries in two seconds and failed the
// person's reply while three conversations ran at once, though a slot frees
// within seconds.
func (s *Engine) awaitSlot(
	ctx context.Context,
	provider *aiprovider.Provider,
	token string,
	ttl time.Duration,
	wait bool,
) (bool, error) {
	deadline := time.Now().Add(slotWait)
	if bound, ok := ctx.Deadline(); ok && bound.Before(deadline) {
		deadline = bound
	}
	for {
		storeCtx, cancel := context.WithTimeout(ctx, slotStoreTimeout)
		acquired, err := s.Limits.slots.Acquire(storeCtx, llmports.AcquireProviderSlotRequest{
			ProviderID: provider.ID,
			Token:      token,
			Limit:      provider.ResolvedMaxConcurrent(),
			TTL:        ttl,
		})
		cancel()
		if err != nil || acquired || !wait || !time.Now().Add(slotPoll).Before(deadline) {
			return acquired, err
		}

		select {
		case <-ctx.Done():
			return false, nil
		case <-time.After(slotPoll):
		}
	}
}

func (s *Engine) holdSlot(
	providerID pulid.ID,
	token string,
	ttl time.Duration,
	stop <-chan struct{},
) {
	ticker := time.NewTicker(ttl / 3)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			refreshCtx, cancel := context.WithTimeout(context.Background(), slotStoreTimeout)
			_, err := s.Limits.slots.Refresh(refreshCtx, llmports.ProviderSlotRequest{
				ProviderID: providerID,
				Token:      token,
				TTL:        ttl,
			})
			cancel()
			if err != nil {
				s.Logger.Debug("could not refresh a provider slot", zap.Error(err))
			}
		}
	}
}

type deadline struct {
	Ctx    context.Context
	cancel context.CancelCauseFunc
	timer  *time.Timer
}

func (s *Engine) WithDeadline(ctx context.Context, timeout time.Duration) *deadline {
	limited, cancel := context.WithCancelCause(ctx)
	d := &deadline{Ctx: limited, cancel: cancel}
	d.timer = time.AfterFunc(timeout, func() { cancel(ErrProviderTimedOut) })

	return d
}

func (d *deadline) Disarm() {
	d.timer.Stop()
}

func (d *deadline) Finish(err error) error {
	d.timer.Stop()
	timedOut := errors.Is(context.Cause(d.Ctx), ErrProviderTimedOut)
	d.cancel(nil)
	if err != nil && timedOut {
		return fmt.Errorf("%w: %w", ErrProviderTimedOut, err)
	}

	return err
}

func (s *Engine) WithKeyFallback(
	provider *aiprovider.Provider,
	call func(apiKey string) error,
) error {
	apiKey, err := s.ResolveAPIKey(provider)
	if err != nil {
		return err
	}

	err = call(apiKey)
	if err == nil || !keyRefused(err) ||
		!aiproviderkey.PreviousKeyUsable(provider, timeutils.NowUnix()) {
		return err
	}

	previous, _, decryptErr := s.Encryption.DecryptStringWithLegacy(
		provider.PreviousAPIKey,
		CredentialAAD(provider),
	)
	if decryptErr != nil {
		return err
	}
	s.Logger.Info("provider refused its new key; trying the key it replaced",
		zap.String("provider", provider.Name))

	return call(previous)
}

func keyRefused(err error) bool {
	var failure llmports.ProviderFailure
	if !errors.As(err, &failure) {
		return false
	}
	status := failure.ProviderStatus()

	return status == http.StatusUnauthorized || status == http.StatusForbidden
}

func (s *Engine) TouchKey(
	ctx context.Context,
	provider *aiprovider.Provider,
	tenant pagination.TenantInfo,
) {
	if s.Limits == nil || s.Limits.keys == nil || !provider.HasStoredAPIKey() ||
		tenant.OrgID.IsNil() {
		return
	}

	now := time.Now()
	s.Limits.mu.Lock()
	last, seen := s.Limits.touched[provider.ID]
	if seen && now.Sub(last) < keyTouchInterval {
		s.Limits.mu.Unlock()
		return
	}
	if len(s.Limits.touched) >= maxTouchedProvider {
		clear(s.Limits.touched)
	}
	s.Limits.touched[provider.ID] = now
	s.Limits.mu.Unlock()

	touchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), slotStoreTimeout)
	defer cancel()
	if err := s.Limits.keys.TouchUsed(touchCtx, aiproviderports.TouchAIProviderKeyRequest{
		ID:         provider.ID,
		TenantInfo: tenant,
		UsedAt:     now.Unix(),
	}); err != nil {
		s.Logger.Debug("could not record when a provider key was last used",
			zap.String("provider", provider.Name),
			zap.Error(err))
	}
}
