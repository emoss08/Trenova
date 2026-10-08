package completionrouter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	slotMargin         = 30 * time.Second
	slotStoreTimeout   = 2 * time.Second
	keyTouchInterval   = 5 * time.Minute
	maxTouchedProvider = 4096
)

var errProviderTimedOut = fmt.Errorf("the AI provider did not answer within its timeout: %w",
	context.DeadlineExceeded)

type providerLimits struct {
	spend serviceports.AIProviderSpendService
	slots repositories.ProviderSlotRepository
	keys  repositories.AIProviderKeyRepository

	mu      sync.Mutex
	touched map[pulid.ID]time.Time
}

func newProviderLimits(p *Params) *providerLimits {
	return &providerLimits{
		spend:   p.Spend,
		slots:   p.Slots,
		keys:    p.Keys,
		touched: make(map[pulid.ID]time.Time),
	}
}

func (s *Service) underCap(
	ctx context.Context,
	tenant pagination.TenantInfo,
	providers []*aiprovider.Provider,
) ([]*aiprovider.Provider, error) {
	if s.limits == nil || s.limits.spend == nil {
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

	spend, err := s.limits.spend.MonthSpend(ctx, tenant, capped)
	if err != nil {
		s.logger.Warn("could not read provider spend; monthly caps not applied to this call",
			zap.Error(err))
		return providers, nil
	}

	within := make([]*aiprovider.Provider, 0, len(providers))
	for _, provider := range providers {
		if !provider.CapReached(spend[provider.ID]) {
			within = append(within, provider)
			continue
		}
		s.logger.Debug("skipping provider at its monthly cap",
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
	).WithInternal(serviceports.ErrProviderCapReached)
}

func (s *Service) claimSlot(
	ctx context.Context,
	provider *aiprovider.Provider,
	timeout time.Duration,
) (func(), error) {
	if s.limits == nil || s.limits.slots == nil || provider.ID.IsNil() {
		return func() {}, nil
	}

	ttl := timeout + slotMargin
	token := pulid.MustNew("slot_").String()
	storeCtx, cancel := context.WithTimeout(ctx, slotStoreTimeout)
	acquired, err := s.limits.slots.Acquire(storeCtx, repositories.AcquireProviderSlotRequest{
		ProviderID: provider.ID,
		Token:      token,
		Limit:      provider.ResolvedMaxConcurrent(),
		TTL:        ttl,
	})
	cancel()
	if err != nil {
		s.logger.Warn("could not claim a provider slot; calling without one",
			zap.String("provider", provider.Name),
			zap.Error(err))
		return func() {}, nil
	}
	if !acquired {
		return nil, fmt.Errorf("%s: %w", provider.Name, serviceports.ErrProviderBusy)
	}

	stop := make(chan struct{})
	var once sync.Once
	go s.holdSlot(provider.ID, token, ttl, stop)

	return func() {
		once.Do(func() {
			close(stop)
			releaseCtx, release := context.WithTimeout(context.WithoutCancel(ctx), slotStoreTimeout)
			defer release()
			if releaseErr := s.limits.slots.Release(releaseCtx, repositories.ProviderSlotRequest{
				ProviderID: provider.ID,
				Token:      token,
			}); releaseErr != nil {
				s.logger.Warn("could not release a provider slot; it expires on its own",
					zap.String("provider", provider.Name),
					zap.Error(releaseErr))
			}
		})
	}, nil
}

func (s *Service) holdSlot(providerID pulid.ID, token string, ttl time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(ttl / 3)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			refreshCtx, cancel := context.WithTimeout(context.Background(), slotStoreTimeout)
			_, err := s.limits.slots.Refresh(refreshCtx, repositories.ProviderSlotRequest{
				ProviderID: providerID,
				Token:      token,
				TTL:        ttl,
			})
			cancel()
			if err != nil {
				s.logger.Debug("could not refresh a provider slot", zap.Error(err))
			}
		}
	}
}

type deadline struct {
	ctx    context.Context
	cancel context.CancelCauseFunc
	timer  *time.Timer
}

func (s *Service) withDeadline(ctx context.Context, timeout time.Duration) *deadline {
	limited, cancel := context.WithCancelCause(ctx)
	d := &deadline{ctx: limited, cancel: cancel}
	d.timer = time.AfterFunc(timeout, func() { cancel(errProviderTimedOut) })

	return d
}

func (d *deadline) disarm() {
	d.timer.Stop()
}

func (d *deadline) finish(err error) error {
	d.timer.Stop()
	timedOut := errors.Is(context.Cause(d.ctx), errProviderTimedOut)
	d.cancel(nil)
	if err != nil && timedOut {
		return fmt.Errorf("%w: %w", errProviderTimedOut, err)
	}

	return err
}

func (s *Service) withKeyFallback(
	provider *aiprovider.Provider,
	call func(apiKey string) error,
) error {
	apiKey, err := s.resolveAPIKey(provider)
	if err != nil {
		return err
	}

	err = call(apiKey)
	if err == nil || !keyRefused(err) || !provider.PreviousKeyUsable(timeutils.NowUnix()) {
		return err
	}

	previous, decryptErr := s.encryption.DecryptString(provider.PreviousAPIKey)
	if decryptErr != nil {
		return err
	}
	s.logger.Info("provider refused its new key; trying the key it replaced",
		zap.String("provider", provider.Name))

	return call(previous)
}

func keyRefused(err error) bool {
	var failure serviceports.ProviderFailure
	if !errors.As(err, &failure) {
		return false
	}
	status := failure.ProviderStatus()

	return status == http.StatusUnauthorized || status == http.StatusForbidden
}

func (s *Service) touchKey(ctx context.Context, provider *aiprovider.Provider, tenant pagination.TenantInfo) {
	if s.limits == nil || s.limits.keys == nil || !provider.HasStoredAPIKey() ||
		tenant.OrgID.IsNil() {
		return
	}

	now := time.Now()
	s.limits.mu.Lock()
	last, seen := s.limits.touched[provider.ID]
	if seen && now.Sub(last) < keyTouchInterval {
		s.limits.mu.Unlock()
		return
	}
	if len(s.limits.touched) >= maxTouchedProvider {
		clear(s.limits.touched)
	}
	s.limits.touched[provider.ID] = now
	s.limits.mu.Unlock()

	touchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), slotStoreTimeout)
	defer cancel()
	if err := s.limits.keys.TouchUsed(touchCtx, repositories.TouchAIProviderKeyRequest{
		ID:         provider.ID,
		TenantInfo: tenant,
		UsedAt:     now.Unix(),
	}); err != nil {
		s.logger.Debug("could not record when a provider key was last used",
			zap.String("provider", provider.Name),
			zap.Error(err))
	}
}
