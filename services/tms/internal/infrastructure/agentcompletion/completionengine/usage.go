package completionengine

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aipricing"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// usageAttempt is everything one attempt is known by once it has returned.
type UsageAttempt struct {
	Provider    *aiprovider.Provider
	Task        aitask.Task
	Surface     aiusage.Surface
	Attribution aiusageports.AIUsageAttribution
	Tenant      pagination.TenantInfo
	Latency     time.Duration
	// firstToken is how long the attempt took to stream its first text or
	// thinking; zero when it streamed none.
	FirstToken time.Duration
	Streamed   bool
	Operation  string
	Attempt    int
	Failover   bool
	// outcome is nil when the attempt failed before the provider wrote
	// anything; tokens then stay zero. An attempt that failed or was stopped
	// partway carries what was counted from its stream.
	Outcome *RunOutcome
	Err     error
}

func SurfaceFor(
	surface aiusage.Surface,
	attribution *aiusageports.AIUsageAttribution,
) aiusage.Surface {
	if attribution.Evaluates() {
		return aiusage.SurfaceEvaluation
	}

	return surface
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
func (s *Engine) Record(ctx context.Context, attempt *UsageAttempt) {
	if s.Usage == nil || attempt.Provider == nil {
		return
	}

	row := UsageRow(ctx, attempt)
	providerName := attempt.Provider.Name
	detached, cancel := context.WithTimeout(
		dbscope.WithValidTenant(context.WithoutCancel(ctx), attempt.Tenant.DBTenant()),
		recordTimeout,
	)
	go func() {
		defer cancel()
		if err := s.Usage.Create(detached, row); err != nil {
			s.Logger.Warn("ai usage record dropped",
				zap.String("provider", providerName),
				zap.Error(err),
			)
		}
	}()
}

func UsageRow(ctx context.Context, attempt *UsageAttempt) *aiusage.AIUsageRecord {
	row := &aiusage.AIUsageRecord{
		OrganizationID:    attempt.Tenant.OrgID,
		BusinessUnitID:    attempt.Tenant.BuID,
		ProviderID:        attempt.Provider.ID,
		ProviderKind:      attempt.Provider.Kind,
		Model:             attempt.Provider.Model,
		Task:              attempt.Task,
		Surface:           attempt.Surface,
		UserID:            attempt.Attribution.UserID,
		AgentDefinitionID: attempt.Attribution.AgentDefinitionID,
		ThreadID:          attempt.Attribution.ThreadID,
		RunID:             attempt.Attribution.RunID,
		Feature:           attempt.Attribution.Feature,
		Succeeded:         attempt.Err == nil,
		ErrorClass:        ClassifyError(attempt.Err),
		ErrorMessage:      failureMessage(attempt.Err),
		Streamed:          attempt.Streamed,
		LatencyMs:         attempt.Latency.Milliseconds(),
		OwnerKind:         agent.RunOwnerKind(attempt.Attribution.OwnerKind),
		OwnerID:           attempt.Attribution.OwnerID,
		DelegateCallID:    attempt.Attribution.DelegateCallID,
		Attempt:           attempt.Attempt,
		Failover:          attempt.Failover,
		AgentDefinitionVersion: intutils.ClonePointer(
			attempt.Attribution.DefinitionVersion,
		),
	}
	if attempt.FirstToken > 0 {
		row.FirstTokenMs = new(attempt.FirstToken.Milliseconds())
	}
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		row.TraceID = sc.TraceID().String()
		row.SpanID = sc.SpanID().String()
	}
	if subject := attempt.Attribution.Subject; subject.Recordable() {
		row.SubjectType = subject.Type
		row.SubjectID = subject.ID
	}
	if attempt.Outcome != nil {
		row.Model = stringutils.FirstNonEmpty(attempt.Outcome.Model, row.Model)
		row.InputTokens = attempt.Outcome.InputTokens
		row.OutputTokens = attempt.Outcome.OutputTokens
		row.ReasoningTokens = attempt.Outcome.ReasoningTokens
		row.CacheReadTokens = attempt.Outcome.CacheReadTokens
		row.CacheWriteTokens = attempt.Outcome.CacheWriteTokens
		row.CostUSD = attempt.Outcome.Cost(attempt.Provider, attempt.Task)
		row.Batched = attempt.Outcome.Batched
		recordAdvisor(row, attempt.Outcome.Advisor, attempt.Provider.Kind)
	}

	return row
}

func recordAdvisor(
	row *aiusage.AIUsageRecord,
	advisor *aipricing.AdvisorUsage,
	kind aiprovider.Kind,
) {
	if !advisor.Used() {
		return
	}
	row.AdvisorModel = advisor.Model
	row.AdvisorCalls = advisor.Calls
	row.AdvisorInputTokens = advisor.Tokens.Input
	row.AdvisorOutputTokens = advisor.Tokens.Output
	row.AdvisorCacheReadTokens = advisor.Tokens.CacheRead
	row.AdvisorCacheWriteTokens = advisor.Tokens.CacheWrite
	row.AdvisorCostUSD = advisor.Cost(kind)
}

// classifyError names a failure in one of a handful of words, so a summary
// can say what went wrong without storing every message.
func ClassifyError(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrRefused):
		return aiusage.ErrorClassRefused
	case errors.Is(err, context.DeadlineExceeded):
		return aiusage.ErrorClassTimeout
	case errors.Is(err, context.Canceled):
		return aiusage.ErrorClassCancelled
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return aiusage.ErrorClassTimeout
		}

		return aiusage.ErrorClassNetwork
	}
	// A transport error carries the provider's own verdict. A 429 is the
	// provider rationing us and a 401 or 403 is a key it no longer accepts;
	// any other 5xx is the provider being unavailable, and a 4xx is the
	// request being wrong. Anything else is an error of ours, which is the
	// provider's fault only in the sense that we could not use what it sent.
	var transport *modeladapter.TransportError
	if errors.As(err, &transport) {
		switch {
		case transport.StatusCode == http.StatusTooManyRequests:
			return aiusage.ErrorClassRateLimited
		case transport.StatusCode == http.StatusUnauthorized,
			transport.StatusCode == http.StatusForbidden:
			return aiusage.ErrorClassAuth
		case transport.Retryable:
			return aiusage.ErrorClassUnavailable
		}
	}

	return aiusage.ErrorClassProvider
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
