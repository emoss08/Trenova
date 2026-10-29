package completionengine

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/shared/pulid"
)

// busyNotice tells the reader a busy provider is being asked again.
type BusyNotice func(attempt int, wait time.Duration, cause error)

// executeWithRetryNoticed is executeWithRetry with the reader told about
// each wait on a busy provider.
func (s *Engine) ExecuteWithRetryNoticed(
	ctx context.Context,
	adapter modeladapter.Adapter,
	call *modeladapter.Call,
	busy BusyNotice,
) (*modeladapter.Response, error) {
	var resp *modeladapter.Response
	err := s.Retrying(ctx, func() error {
		var callErr error
		resp, callErr = adapter.Complete(ctx, call)

		return callErr
	}, busy)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *Engine) Retrying(ctx context.Context, attemptOnce func() error, busy BusyNotice) error {
	var waited time.Duration
	for attempt := 0; ; attempt++ {
		err := attemptOnce()
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return err
		}

		wait, again := s.RetryWait(err, attempt, waited)
		if !again {
			return err
		}
		if Unavailability(err) {
			RecordBusyWait(ctx, wait)
			if busy != nil {
				busy(attempt+1, wait, err)
			}
		}
		if waitErr := s.Wait(ctx, wait); waitErr != nil {
			return waitErr
		}
		waited += wait
	}
}

func PreferFirst(providers []*aiprovider.Provider, preferred pulid.ID) []*aiprovider.Provider {
	if preferred.IsNil() {
		return providers
	}

	for idx, provider := range providers {
		if provider.ID != preferred {
			continue
		}
		if idx == 0 {
			return providers
		}

		ordered := make([]*aiprovider.Provider, 0, len(providers))
		ordered = append(ordered, provider)
		ordered = append(ordered, providers[:idx]...)
		ordered = append(ordered, providers[idx+1:]...)

		return ordered
	}

	return providers
}
