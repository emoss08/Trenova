package completionengine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"go.uber.org/zap"
)

type hedgedResult struct {
	idx     int
	Outcome *RunOutcome
	Err     error
}

// runHedged asks the providers in order for a caller bounded by a short
// deadline. Asked one after another, the first provider spent the scope
// guard's whole three seconds whenever it was slow, and the guard let the
// question through unclassified though a second provider was configured. Here
// the next is asked too once the one asked last has not answered within
// HedgeAfter, or at once when it fails, and the first answer wins; the others
// are canceled, which observe reads as a cancellation, not a fault.
func (s *Engine) runHedged(
	ctx context.Context,
	usable []*aiprovider.Provider,
	req *RunRequest,
) (*RunOutcome, error) {
	hedgeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(chan hedgedResult, len(usable))
	next, inFlight := 0, 0
	launch := func() {
		candidate := &candidateAttempt{Provider: usable[next], idx: next, count: len(usable)}
		next++
		inFlight++
		go func() {
			outcome, err := s.tryProvider(hedgeCtx, candidate, req)
			results <- hedgedResult{idx: candidate.idx, Outcome: outcome, Err: err}
		}()
	}

	launch()
	timer := time.NewTimer(req.HedgeAfter)
	defer timer.Stop()

	var lastErr error
	for inFlight > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			if next < len(usable) {
				s.Logger.Info("provider slow, asking the next as well",
					zap.String("provider", usable[next-1].Name),
					zap.String("next", usable[next].Name),
					zap.String("task", string(req.Task)),
					zap.Duration("after", req.HedgeAfter),
				)
				launch()
				timer.Reset(req.HedgeAfter)
			}
		case result := <-results:
			inFlight--
			if result.Err == nil {
				if result.idx > 0 {
					s.Logger.Info("hedged call answered by a later provider",
						zap.String("provider", usable[result.idx].Name),
						zap.String("task", string(req.Task)),
					)
				}

				return result.Outcome, nil
			}
			if ctx.Err() != nil {
				return nil, result.Err
			}
			if errors.Is(result.Err, ErrRefused) {
				return nil, errDeclined()
			}

			lastErr = result.Err
			s.fellThrough(usable[result.idx], req, result.Err)
			if next < len(usable) {
				launch()
				timer.Reset(req.HedgeAfter)
			}
		}
	}

	return nil, fmt.Errorf("every configured provider for %s failed: %w", req.Task, lastErr)
}
