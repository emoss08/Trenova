package completionrouter

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

// usageAttempt is everything one attempt is known by once it has returned.
type usageAttempt struct {
	provider    *aiprovider.Provider
	task        aiprovider.Task
	surface     aiusage.Surface
	attribution serviceports.AIUsageAttribution
	tenant      pagination.TenantInfo
	latency     time.Duration
	streamed    bool
	// outcome is nil when the attempt failed; tokens then stay zero, since a
	// provider that errored reported none.
	outcome *runOutcome
	err     error
}

// recordTimeout bounds the write. A record that cannot be written in this
// long is dropped and logged: the answer has already gone to the person, and
// a bookkeeping row is not worth holding a connection for.
const recordTimeout = 5 * time.Second

// record writes one usage row, off the request's critical path.
//
// The write rides a context the request's cancellation cannot reach and
// happens after the reply is on its way, so a slow or failing usage table
// never slows an answer or turns a success into an error. Its failure is
// logged and nothing else: the row is telemetry, not the transaction.
func (s *Service) record(ctx context.Context, attempt usageAttempt) {
	if s.usage == nil || attempt.provider == nil {
		return
	}

	row := &aiusage.AIUsageRecord{
		OrganizationID:    attempt.tenant.OrgID,
		BusinessUnitID:    attempt.tenant.BuID,
		ProviderID:        attempt.provider.ID,
		ProviderKind:      attempt.provider.Kind,
		Model:             attempt.provider.Model,
		Task:              attempt.task,
		Surface:           attempt.surface,
		UserID:            attempt.attribution.UserID,
		AgentDefinitionID: attempt.attribution.AgentDefinitionID,
		ThreadID:          attempt.attribution.ThreadID,
		RunID:             attempt.attribution.RunID,
		Succeeded:         attempt.err == nil,
		ErrorClass:        classifyError(attempt.err),
		ErrorMessage:      failureMessage(attempt.err),
		Streamed:          attempt.streamed,
		LatencyMs:         attempt.latency.Milliseconds(),
	}
	if attempt.outcome != nil {
		row.Model = firstNonEmpty(attempt.outcome.Model, row.Model)
		row.InputTokens = attempt.outcome.InputTokens
		row.OutputTokens = attempt.outcome.OutputTokens
		row.ReasoningTokens = attempt.outcome.ReasoningTokens
		row.CostUSD = attempt.provider.CostFor(row.InputTokens, row.OutputTokens)
	}

	detached, cancel := context.WithTimeout(context.WithoutCancel(ctx), recordTimeout)
	go func() {
		defer cancel()
		if err := s.usage.Create(detached, row); err != nil {
			s.logger.Warn("ai usage record dropped",
				zap.String("provider", attempt.provider.Name),
				zap.Error(err),
			)
		}
	}()
}

// classifyError names a failure in one of a handful of words, so a summary
// can say what went wrong without storing every message.
func classifyError(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, errRefused):
		return "refused"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "cancelled"
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "timeout"
		}

		return "network"
	}
	// A transport error carries the provider's own verdict: a 5xx or 429 is
	// the provider being unavailable, a 4xx is the request being wrong.
	// Anything else is an error of ours, which is the provider's fault only
	// in the sense that we could not use what it sent.
	var transport *modeladapter.TransportError
	if errors.As(err, &transport) && transport.Retryable {
		return "provider_unavailable"
	}

	return "provider_error"
}

// maxFailureMessageChars is how much of a provider's error is kept on the
// usage row: the reason, not the stack.
const maxFailureMessageChars = 500

// failureMessage is the provider's own account of a failed attempt. The
// class already says what kind of failure it was; the message says why,
// which is the difference between "the provider is down" and "the provider
// refuses this request" when both show as a failed call.
func failureMessage(err error) string {
	if err == nil {
		return ""
	}

	return stringutils.TruncateRunes(strings.TrimSpace(err.Error()), maxFailureMessageChars)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}
