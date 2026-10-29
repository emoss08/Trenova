package modelstream

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/loopguard"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

// Reply is one completion as the loop sees it.
type Reply struct {
	Completion *llmports.ChatCompletionResult `json:"completion"`
	Looped     bool                           `json:"looped"`
	// InReasoning says the loop was in the model's thinking rather than its
	// reply.
	InReasoning bool                        `json:"inReasoning,omitempty"`
	Usage       *agentruntimeports.RunUsage `json:"usage,omitempty"`
}

func (r Reply) TotalUsage() *agentruntimeports.RunUsage {
	if r.Usage != nil {
		return r.Usage
	}
	if r.Completion == nil {
		return nil
	}

	return &agentruntimeports.RunUsage{
		ModelCalls:       1,
		InputTokens:      int64(r.Completion.InputTokens),
		OutputTokens:     int64(r.Completion.OutputTokens),
		CacheReadTokens:  int64(r.Completion.CacheReadTokens),
		CacheWriteTokens: int64(r.Completion.CacheWriteTokens),
		CostUSD:          r.Completion.CostUSD,
	}
}

// StreamCompletion asks the model for one reply and streams it to emit as it
// arrives: the text, the thinking, and any restart. A reply that falls into a
// loop is cut off the moment it is seen, so the provider stops writing, and is
// reported as Looped rather than returned as an answer.
//
// It is the model call wherever a turn runs: inline for a turn in process, and
// inside the model activity for a durable one, where emit publishes to the
// run's stream.
func (s *Streamer) StreamCompletion(
	ctx context.Context,
	req *llmports.ChatCompletionRequest,
	emit kernelports.AssistantStreamEmitter,
) (Reply, error) {
	req = s.DeclareTools(req)
	ctx, tally := aitrace.WithCompletionTally(ctx)
	reply, err := s.streamCompletion(ctx, req, emit)
	reply.Usage = replyUsage(tally, reply)

	return reply, err
}

func replyUsage(tally *aitrace.CompletionTally, reply Reply) *agentruntimeports.RunUsage {
	totals := tally.Totals()
	if totals.Attempts == 0 {
		return reply.TotalUsage()
	}

	return &agentruntimeports.RunUsage{
		ModelCalls:       totals.Attempts,
		InputTokens:      totals.InputTokens,
		OutputTokens:     totals.OutputTokens,
		CacheReadTokens:  totals.CacheReadTokens,
		CacheWriteTokens: totals.CacheWriteTokens,
		CostUSD:          totals.CostUSD,
	}
}

func (s *Streamer) streamCompletion(
	ctx context.Context,
	req *llmports.ChatCompletionRequest,
	emit kernelports.AssistantStreamEmitter,
) (Reply, error) {
	origin := aitrace.CallOriginFrom(ctx)
	ctx, span := aitrace.StartModelCall(ctx, &aitrace.ModelCallSpec{
		Anchor:          aitrace.ForAttribution(&req.Attribution),
		Feature:         string(req.Attribution.Feature),
		ActivityAttempt: origin.ActivityAttempt,
		Stream:          origin.Stream,
	})
	defer span.End()
	defer aitrace.TallyFrom(ctx).Record(span)

	streamCtx, cancelStream := context.WithCancel(ctx)
	defer cancelStream()

	guard := loopguard.NewReplyGuard(cancelStream)
	// The thinking is watched the way the reply is. A small model that falls
	// into a loop does it in its reasoning as readily as in its answer, and an
	// unwatched trace streamed thousands of repeated fragments to the reader
	// before the provider ran out of tokens.
	thinking := loopguard.NewReplyGuard(cancelStream)
	req.ReasoningSink = func(delta string) {
		if thinking.Feed(delta) {
			emit(reasoningEvent(delta))
		}
	}
	req.RetrySink = func(notice llmports.ChatRetryNotice) { emit(retryEvent(notice)) }

	completion, err := s.completion.StreamChat(streamCtx, req, func(delta string) {
		if guard.Feed(delta) {
			emit(DeltaEvent(delta))
		}
	})
	cancelStream()
	// A loop in the reply or the thinking cancels the stream, so the call
	// returns the cancellation rather than a completion. The turn itself is
	// fine and is asked again.
	if (guard.Tripped || thinking.Tripped) && ctx.Err() == nil {
		if completion == nil {
			completion = &llmports.ChatCompletionResult{}
		}
		span.SetAttributes(aitrace.AILooped.Bool(true))

		return Reply{Completion: completion, Looped: true, InReasoning: thinking.Tripped}, nil
	}
	if err != nil {
		aitrace.MarkFailed(span, completionErrorType(ctx, err))

		return Reply{Completion: completion}, err
	}

	looped := guard.Looped(completion.Text)
	span.SetAttributes(aitrace.AILooped.Bool(looped))

	return Reply{Completion: completion, Looped: looped}, nil
}

func completionErrorType(ctx context.Context, err error) string {
	switch {
	case errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled):
		return "cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, llmports.ErrNoProviderConfigured):
		return "no_provider"
	case errors.Is(err, llmports.ErrProvidersResting):
		return "providers_resting"
	default:
		return "provider_error"
	}
}

func (s *Streamer) DeclareTools(
	req *llmports.ChatCompletionRequest,
) *llmports.ChatCompletionRequest {
	if len(req.DeferredTools) == 0 {
		return req
	}
	declared := *req
	declared.Tools = make([]kernelports.ToolSpec, 0, len(req.Tools)+len(req.DeferredTools))
	declared.Tools = append(declared.Tools, req.Tools...)
	for _, name := range req.DeferredTools {
		spec, ok := s.access.SpecFor(name)
		if !ok {
			continue
		}
		spec.Deferred = true
		declared.Tools = append(declared.Tools, spec)
	}
	slices.SortStableFunc(declared.Tools, func(a, b kernelports.ToolSpec) int {
		return strings.Compare(a.Name, b.Name)
	})
	declared.DeferredTools = nil

	return &declared
}

func DeltaEvent(text string) kernelports.StreamEvent {
	return kernelports.StreamEvent{
		Event: assistantports.AssistantEventDelta,
		Data:  assistantports.AssistantDeltaEvent{Text: text},
	}
}

func reasoningEvent(text string) kernelports.StreamEvent {
	return kernelports.StreamEvent{
		Event: assistantports.AssistantEventReasoning,
		Data:  assistantports.AssistantReasoningEvent{Text: text},
	}
}

func retryEvent(notice llmports.ChatRetryNotice) kernelports.StreamEvent {
	return kernelports.StreamEvent{
		Event: assistantports.AssistantEventRetrying,
		Data: assistantports.AssistantRetryingEvent{
			Attempt:     notice.Attempt,
			Provider:    notice.Provider,
			Reason:      notice.Reason,
			Kind:        notice.Kind,
			WaitSeconds: notice.WaitSeconds,
			MaxAttempts: notice.MaxAttempts,
		},
	}
}
