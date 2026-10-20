package deskbench

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

type ModelCallKind string

const (
	ModelCallChat       = ModelCallKind("chat")
	ModelCallStream     = ModelCallKind("stream")
	ModelCallStructured = ModelCallKind("structured")
)

type ModelCall struct {
	Seq              int                                     `json:"seq"`
	Kind             ModelCallKind                           `json:"kind"`
	StartedAt        time.Time                               `json:"startedAt"`
	FinishedAt       time.Time                               `json:"finishedAt"`
	Attribution      aiusageports.AIUsageAttribution         `json:"attribution"`
	Chat             *agentports.ChatCompletionRequest       `json:"chat,omitempty"`
	Structured       *agentports.StructuredCompletionRequest `json:"structured,omitempty"`
	ChatResult       *agentports.ChatCompletionResult        `json:"chatResult,omitempty"`
	StructuredResult *agentports.StructuredCompletionResult  `json:"structuredResult,omitempty"`
	Retries          []agentports.ChatRetryNotice            `json:"retries,omitempty"`
	Error            string                                  `json:"error,omitempty"`
	FirstTokenAt     time.Time                               `json:"firstTokenAt,omitzero"`
}

func (c *ModelCall) Duration() time.Duration {
	return c.FinishedAt.Sub(c.StartedAt)
}

func (c *ModelCall) belongsTo(threadID, turnID pulid.ID) bool {
	if c.Attribution.ThreadID.IsNotNil() {
		return c.Attribution.ThreadID == threadID
	}

	return turnID.IsNotNil() && c.Attribution.OwnerID == turnID
}

type Recorder struct {
	agentports.CompletionService

	mu    sync.Mutex
	seq   int
	calls []*ModelCall
	pins  map[pulid.ID]ProviderPin
}

type ProviderPin struct {
	ProviderID pulid.ID
	Effort     aiprovider.ReasoningEffort
	Advisor    *agentports.AdvisorSpec
}

func (p ProviderPin) empty() bool {
	return p.ProviderID.IsNil() && p.Effort == "" && p.Advisor == nil
}

func NewRecorder(inner agentports.CompletionService) *Recorder {
	return &Recorder{CompletionService: inner, pins: make(map[pulid.ID]ProviderPin, 4)}
}

func (r *Recorder) Pin(key pulid.ID, pin ProviderPin) func() {
	if key.IsNil() || pin.empty() {
		return func() {}
	}

	r.mu.Lock()
	r.pins[key] = pin
	r.mu.Unlock()

	return func() {
		r.mu.Lock()
		delete(r.pins, key)
		r.mu.Unlock()
	}
}

func (r *Recorder) pin(req *agentports.ChatCompletionRequest) {
	r.mu.Lock()
	pin, ok := r.pins[req.Attribution.ThreadID]
	if !ok {
		pin, ok = r.pins[req.Attribution.AgentDefinitionID]
	}
	r.mu.Unlock()
	if !ok {
		return
	}

	if pin.ProviderID.IsNotNil() {
		req.PreferredProviderID = pin.ProviderID
		req.PinPreferred = true
	}
	if pin.Effort != "" {
		req.Reasoning = pin.Effort
	}
	if pin.Advisor != nil {
		advisor := *pin.Advisor
		req.Advisor = &advisor
	}
}

func (r *Recorder) CompleteChat(
	ctx context.Context,
	req *agentports.ChatCompletionRequest,
) (*agentports.ChatCompletionResult, error) {
	r.pin(req)
	call := r.openChat(ModelCallChat, req)
	restore := call.watchRetries(&r.mu, req)
	defer restore()
	restoreReasoning := call.watchFirstToken(&r.mu, req)
	defer restoreReasoning()

	result, err := r.CompletionService.CompleteChat(ctx, req)
	r.closeChat(call, result, err)

	return result, err
}

func (r *Recorder) StreamChat(
	ctx context.Context,
	req *agentports.ChatCompletionRequest,
	sink agentports.ChatStreamSink,
) (*agentports.ChatCompletionResult, error) {
	r.pin(req)
	call := r.openChat(ModelCallStream, req)
	restore := call.watchRetries(&r.mu, req)
	defer restore()
	restoreReasoning := call.watchFirstToken(&r.mu, req)
	defer restoreReasoning()

	result, err := r.CompletionService.StreamChat(ctx, req, call.firstTokenSink(&r.mu, sink))
	r.closeChat(call, result, err)

	return result, err
}

func (r *Recorder) CompleteStructured(
	ctx context.Context,
	req *agentports.StructuredCompletionRequest,
) (*agentports.StructuredCompletionResult, error) {
	call := &ModelCall{
		Kind:        ModelCallStructured,
		StartedAt:   time.Now(),
		Attribution: req.Attribution,
		Structured:  snapshot(req),
	}
	r.open(call)

	result, err := r.CompletionService.CompleteStructured(ctx, req)

	r.mu.Lock()
	defer r.mu.Unlock()
	call.FinishedAt = time.Now()
	call.StructuredResult = snapshot(result)
	if err != nil {
		call.Error = err.Error()
	}

	return result, err
}

func (r *Recorder) For(threadID, turnID pulid.ID, start, end time.Time) []*ModelCall {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]*ModelCall, 0, 8)
	for _, call := range r.calls {
		if call.StartedAt.Before(start) || call.StartedAt.After(end) {
			continue
		}
		if call.belongsTo(threadID, turnID) {
			out = append(out, call)
		}
	}

	return out
}

func (r *Recorder) Unattributed(start, end time.Time) []*ModelCall {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]*ModelCall, 0, 4)
	for _, call := range r.calls {
		if call.StartedAt.Before(start) || call.StartedAt.After(end) {
			continue
		}
		if call.Attribution.ThreadID.IsNil() && call.Attribution.OwnerID.IsNil() {
			out = append(out, call)
		}
	}

	return out
}

func (r *Recorder) open(call *ModelCall) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.seq++
	call.Seq = r.seq
	r.calls = append(r.calls, call)
}

func (r *Recorder) openChat(kind ModelCallKind, req *agentports.ChatCompletionRequest) *ModelCall {
	call := &ModelCall{
		Kind:        kind,
		StartedAt:   time.Now(),
		Attribution: req.Attribution,
		Chat:        snapshot(req),
	}
	r.open(call)

	return call
}

func (r *Recorder) closeChat(
	call *ModelCall,
	result *agentports.ChatCompletionResult,
	err error,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	call.FinishedAt = time.Now()
	call.ChatResult = snapshot(result)
	if err != nil {
		call.Error = err.Error()
	}
}

func (c *ModelCall) watchRetries(mu *sync.Mutex, req *agentports.ChatCompletionRequest) func() {
	original := req.RetrySink
	req.RetrySink = func(notice agentports.ChatRetryNotice) {
		mu.Lock()
		c.Retries = append(c.Retries, notice)
		mu.Unlock()
		if original != nil {
			original(notice)
		}
	}

	return func() { req.RetrySink = original }
}

func (c *ModelCall) markFirstToken(mu *sync.Mutex) {
	mu.Lock()
	if c.FirstTokenAt.IsZero() {
		c.FirstTokenAt = time.Now()
	}
	mu.Unlock()
}

func (c *ModelCall) firstTokenSink(
	mu *sync.Mutex,
	sink agentports.ChatStreamSink,
) agentports.ChatStreamSink {
	return func(delta string) {
		if delta != "" {
			c.markFirstToken(mu)
		}
		if sink != nil {
			sink(delta)
		}
	}
}

func (c *ModelCall) watchFirstToken(mu *sync.Mutex, req *agentports.ChatCompletionRequest) func() {
	original := req.ReasoningSink
	if original == nil {
		return func() {}
	}
	req.ReasoningSink = func(delta string) {
		if delta != "" {
			c.markFirstToken(mu)
		}
		original(delta)
	}

	return func() { req.ReasoningSink = original }
}

func snapshot[T any](value *T) *T {
	if value == nil {
		return nil
	}

	copied := new(T)
	if err := jsonutils.Convert(value, copied); err != nil {
		return value
	}

	return copied
}

func sortCalls(calls []*ModelCall) {
	slices.SortFunc(calls, func(a, b *ModelCall) int { return a.Seq - b.Seq })
}
