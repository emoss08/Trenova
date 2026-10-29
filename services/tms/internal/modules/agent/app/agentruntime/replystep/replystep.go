package replystep

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/modelstream"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/intutils"
	"go.uber.org/zap"
)

const keyPrefix = "completion:"

type Call struct {
	Owner   agentruntimeports.RunStepOwner `json:"owner"`
	Scope   string                         `json:"scope,omitempty"`
	Ordinal int                            `json:"ordinal"`
}

func (c *Call) InLedger() bool {
	return c != nil && c.Owner.ID.IsNotNil() && c.Owner.Kind != "" && c.Ordinal > 0
}

func Key(call *Call, req *llmports.ChatCompletionRequest) string {
	if !call.InLedger() || req == nil {
		return ""
	}

	encoded, err := sonic.ConfigStd.Marshal(req)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(encoded)

	material := call.Owner.ID.String() + "\x00" + string(agentruntimeports.RunStepCompletion) +
		"\x00" + strconv.Itoa(call.Ordinal) + "\x00" + hex.EncodeToString(digest[:])
	if call.Scope != "" {
		material += "\x00" + call.Scope
	}
	sum := sha256.Sum256([]byte(material))

	return keyPrefix + hex.EncodeToString(sum[:])
}

type Ledger struct {
	steps agentruntimeports.RunStepLedger
	l     *zap.Logger
}

func NewLedger(steps agentruntimeports.RunStepLedger, logger *zap.Logger) *Ledger {
	return &Ledger{steps: steps, l: logger.Named("replystep")}
}

type Lookup struct {
	Call    *Call
	Request *llmports.ChatCompletionRequest
	Key     string
}

func (l *Ledger) Recall(ctx context.Context, lookup *Lookup) (modelstream.Reply, bool) {
	if l == nil || l.steps == nil || lookup.Key == "" {
		return modelstream.Reply{}, false
	}

	step, found, err := l.steps.Find(
		ctx,
		lookup.Request.TenantInfo,
		lookup.Call.Owner,
		lookup.Key,
	)
	if err != nil {
		l.l.Warn("could not read whether an earlier attempt kept this model reply",
			zap.String("owner", lookup.Call.Owner.ID.String()),
			zap.Int("ordinal", lookup.Call.Ordinal),
			zap.Error(err),
		)

		return modelstream.Reply{}, false
	}
	if !found || !Replayable(&step) {
		return modelstream.Reply{}, false
	}

	return modelstream.Reply{
		Completion: step.Outcome.Reply.Completion,
		Usage:      step.Outcome.Reply.Usage,
	}, true
}

func Replayable(step *agentruntimeports.RunStep) bool {
	return step.Kind == agentruntimeports.RunStepCompletion &&
		step.Status == agentruntimeports.RunStepCompleted &&
		step.Outcome.Reply != nil &&
		step.Outcome.Reply.Completion != nil
}

type Kept struct {
	Lookup  *Lookup
	Attempt int
	Reply   *modelstream.Reply
}

func (l *Ledger) Keep(ctx context.Context, kept *Kept) {
	lookup := kept.Lookup
	reply := kept.Reply
	if l == nil || l.steps == nil || lookup.Key == "" || reply == nil || reply.Looped ||
		reply.Completion == nil {
		return
	}

	attribution := lookup.Request.Attribution
	step := &agentruntimeports.RunStep{
		OwnerKind: lookup.Call.Owner.Kind,
		OwnerID:   lookup.Call.Owner.ID,
		Attempt:   kept.Attempt,
		Kind:      agentruntimeports.RunStepCompletion,
		Status:    agentruntimeports.RunStepCompleted,
		Key:       lookup.Key,
		Outcome: agentruntimeports.RunStepOutcome{
			Reply: &agentruntimeports.RunStepReply{
				Completion: reply.Completion,
				Usage:      reply.Usage,
			},
		},
		DefinitionID:      attribution.AgentDefinitionID,
		DefinitionVersion: intutils.ClonePointer(attribution.DefinitionVersion),
		DelegateCallID:    attribution.DelegateCallID,
	}
	step.TraceID, step.SpanID = aitrace.IDs(ctx)

	err := l.steps.Record(context.WithoutCancel(ctx), lookup.Request.TenantInfo, step)
	if err != nil {
		l.l.Warn("could not keep a model reply; a replay of this call asks the model again",
			zap.String("owner", lookup.Call.Owner.ID.String()),
			zap.Int("ordinal", lookup.Call.Ordinal),
			zap.Error(err),
		)
	}
}
