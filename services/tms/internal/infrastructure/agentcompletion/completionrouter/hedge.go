package completionrouter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"go.uber.org/zap"
)

type hedgedResult struct {
	idx     int
	outcome *runOutcome
	err     error
}

// runHedged asks the providers in order for a caller bounded by a short
// deadline. Asked one after another, the first provider spent the scope
// guard's whole three seconds whenever it was slow, and the guard let the
// question through unclassified though a second provider was configured. Here
// the next is asked too once the one asked last has not answered within
// HedgeAfter, or at once when it fails, and the first answer wins; the others
// are canceled, which observe reads as a cancellation, not a fault.
func (s *Service) runHedged(
	ctx context.Context,
	usable []*aiprovider.Provider,
	req *runRequest,
) (*runOutcome, error) {
	hedgeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(chan hedgedResult, len(usable))
	next, inFlight := 0, 0
	launch := func() {
		candidate := &candidateAttempt{provider: usable[next], idx: next, count: len(usable)}
		next++
		inFlight++
		go func() {
			outcome, err := s.tryProvider(hedgeCtx, candidate, req)
			results <- hedgedResult{idx: candidate.idx, outcome: outcome, err: err}
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
				s.logger.Info("provider slow, asking the next as well",
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
			if result.err == nil {
				if result.idx > 0 {
					s.logger.Info("hedged call answered by a later provider",
						zap.String("provider", usable[result.idx].Name),
						zap.String("task", string(req.Task)),
					)
				}

				return result.outcome, nil
			}
			if ctx.Err() != nil {
				return nil, result.err
			}
			if errors.Is(result.err, errRefused) {
				return nil, errDeclined()
			}

			lastErr = result.err
			s.fellThrough(usable[result.idx], req, result.err)
			if next < len(usable) {
				launch()
				timer.Reset(req.HedgeAfter)
			}
		}
	}

	return nil, fmt.Errorf("every configured provider for %s failed: %w", req.Task, lastErr)
}
